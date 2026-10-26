package riskmanager_test

import (
	"testing"
	"time"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-riskmanager/models"
)

func TestARatingOffTheOneToFiveScaleIsRefused(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	for _, c := range []struct{ likelihood, impact int }{
		{0, 3}, {3, 0}, {6, 3}, {3, 6}, {-1, 3}, {3, -1}, {0, 0}, {100, 100},
	} {
		_, err := rm.RecordAssessment(ctx, riskmanager.Assessment{
			RiskID: r.ID, Kind: riskmanager.AssessmentInherent, Likelihood: c.likelihood, Impact: c.impact,
		})
		seedWantErr(t, err, riskmanager.ErrInvalid)
	}
	if got, _ := rm.ListAssessments(ctx, riskmanager.AssessmentFilter{RiskID: r.ID}); len(got) != 0 {
		t.Fatalf("refused ratings left %d record(s)", len(got))
	}
	for _, edge := range []int{1, 5} {
		seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentInherent, edge, edge, "2026-01-01T00:00:00Z")
	}
}

func TestARatingNeedsARiskThatExistsAndAKindFromTheDeclaredSet(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	_, err := rm.RecordAssessment(ctx, riskmanager.Assessment{RiskID: "nowhere", Kind: riskmanager.AssessmentResidual, Likelihood: 1, Impact: 1})
	seedWantErr(t, err, riskmanager.ErrInvalidReference)
	_, err = rm.RecordAssessment(ctx, riskmanager.Assessment{RiskID: r.ID, Kind: "gut feeling", Likelihood: 1, Impact: 1})
	seedWantErr(t, err, riskmanager.ErrInvalid)
	_, err = rm.RecordAssessment(ctx, riskmanager.Assessment{RiskID: r.ID, Likelihood: 1, Impact: 1})
	seedWantErr(t, err, riskmanager.ErrInvalid)
	_, err = rm.RecordAssessment(ctx, riskmanager.Assessment{Kind: riskmanager.AssessmentResidual, Likelihood: 1, Impact: 1})
	seedWantErr(t, err, riskmanager.ErrInvalid)
}

func TestARatingWithATimeThatIsNotRFC3339IsRefusedAndAMissingOneIsNow(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	_, err := rm.RecordAssessment(ctx, riskmanager.Assessment{
		RiskID: r.ID, Kind: riskmanager.AssessmentResidual, Likelihood: 1, Impact: 1, AssessedAt: "last week",
	})
	seedWantErr(t, err, riskmanager.ErrInvalid)

	a, err := rm.RecordAssessment(ctx, riskmanager.Assessment{RiskID: r.ID, Kind: riskmanager.AssessmentResidual, Likelihood: 1, Impact: 1})
	seedNoErr(t, err)
	at, err := time.Parse(time.RFC3339, a.AssessedAt)
	if err != nil || time.Since(at) > 5*time.Second || time.Until(at) > 5*time.Second {
		t.Fatalf("assessed_at %q (%v) is not about now", a.AssessedAt, err)
	}
}

func TestTheBandsFollowTheScoreAtEveryEdge(t *testing.T) {
	for score, want := range map[int]models.Band{
		1: models.BandLow, 4: models.BandLow,
		5: models.BandMedium, 9: models.BandMedium,
		10: models.BandHigh, 15: models.BandHigh,
		16: models.BandCritical, 25: models.BandCritical,
	} {
		if got := models.BandOf(score); got != want {
			t.Errorf("score %d is %s, want %s", score, got, want)
		}
	}
	for _, c := range []struct {
		likelihood, impact, score int
		band                      models.Band
	}{
		{1, 4, 4, models.BandLow}, {1, 5, 5, models.BandMedium}, {3, 3, 9, models.BandMedium},
		{2, 5, 10, models.BandHigh}, {3, 5, 15, models.BandHigh}, {4, 4, 16, models.BandCritical}, {5, 5, 25, models.BandCritical},
	} {
		a := models.Assessment{Likelihood: c.likelihood, Impact: c.impact}
		if a.Score() != c.score || a.Band() != c.band {
			t.Errorf("%dx%d = %d/%s, want %d/%s", c.likelihood, c.impact, a.Score(), a.Band(), c.score, c.band)
		}
	}
}

func TestTheBowTieReportsTheBandOfTheLatestRating(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentInherent, 4, 4, "2026-01-01T00:00:00Z")
	bt, err := rm.Bowtie(ctx, r.ID)
	seedNoErr(t, err)
	if bt.Inherent == nil || bt.Inherent.Score != 16 || bt.Inherent.Band != models.BandCritical {
		t.Fatalf("inherent = %+v", bt.Inherent)
	}
	if bt.Residual != nil || bt.Target != nil {
		t.Fatalf("a kind nobody rated showed up: %+v %+v", bt.Residual, bt.Target)
	}
}

func TestTheLatestRatingOfEachKindWinsByWhenItWasMadeNotWhenItWasEntered(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentResidual, 2, 2, "2026-02-01T00:00:00Z")
	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentResidual, 5, 5, "2026-01-01T00:00:00Z")
	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentInherent, 4, 3, "2026-01-01T00:00:00Z")
	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentTarget, 1, 2, "2026-03-01T00:00:00Z")

	bt, err := rm.Bowtie(ctx, r.ID)
	seedNoErr(t, err)
	if bt.Residual == nil || bt.Residual.Assessment.Likelihood != 2 || bt.Residual.Score != 4 {
		t.Fatalf("residual = %+v, want the February rating of 2x2", bt.Residual)
	}
	if bt.Inherent == nil || bt.Inherent.Score != 12 {
		t.Fatalf("inherent = %+v", bt.Inherent)
	}
	if bt.Target == nil || bt.Target.Score != 2 {
		t.Fatalf("target = %+v", bt.Target)
	}
}

func TestRatingsAreListedOldestFirstAndFilterByKindAndRisk(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	other := seedRisk(t, rm, ctx, "y")
	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentResidual, 3, 3, "2026-03-01T00:00:00Z")
	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentResidual, 4, 4, "2026-01-01T00:00:00Z")
	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentInherent, 5, 5, "2026-02-01T00:00:00Z")
	seedAssessment(t, rm, ctx, other.ID, riskmanager.AssessmentResidual, 1, 1, "2026-02-01T00:00:00Z")

	got, err := rm.ListAssessments(ctx, riskmanager.AssessmentFilter{RiskID: r.ID, Kind: riskmanager.AssessmentResidual})
	seedNoErr(t, err)
	if len(got) != 2 || got[0].AssessedAt != "2026-01-01T00:00:00Z" || got[1].AssessedAt != "2026-03-01T00:00:00Z" {
		t.Fatalf("residual history of the first risk: %+v", got)
	}
	all, err := rm.ListAssessments(ctx, riskmanager.AssessmentFilter{RiskID: r.ID})
	seedNoErr(t, err)
	if len(all) != 3 {
		t.Fatalf("every kind for the first risk: %d", len(all))
	}
	everything, err := rm.ListAssessments(ctx, riskmanager.AssessmentFilter{})
	seedNoErr(t, err)
	if len(everything) != 4 {
		t.Fatalf("everything: %d", len(everything))
	}
}
