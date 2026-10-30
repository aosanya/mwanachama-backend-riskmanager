package riskmanager_test

import (
	"testing"
	"time"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-riskmanager/models"
)

func daysAgo(n int) string {
	return time.Now().UTC().AddDate(0, 0, -n).Format(time.RFC3339)
}

func staleBarrier(t *testing.T, retestDays int, outcome riskmanager.Outcome, recordedAt string) (riskmanager.RiskManager, riskmanager.Risk, riskmanager.Barrier) {
	t.Helper()
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	b, err := rm.PlaceBarrier(ctx, riskmanager.Barrier{
		PathwayID: seedThreat(t, rm, ctx, r.ID, "t").ID, ControlID: seedControl(t, rm, ctx, "c", "o").ID, RetestDays: retestDays,
	})
	seedNoErr(t, err)
	seedEvidence(t, rm, ctx, b.ID, outcome, recordedAt)
	b, err = rm.UpdateBarrier(ctx, riskmanager.BarrierChange{
		ID: b.ID, Design: seedPtr(riskmanager.JudgementAdequate), Coverage: seedPtr(riskmanager.JudgementAdequate), Maturity: seedPtr(riskmanager.MaturityTested),
	})
	seedNoErr(t, err)
	return rm, r, b
}

func standingOf(t *testing.T, rm riskmanager.RiskManager, riskID string) riskmanager.Standing {
	t.Helper()
	got, err := rm.PreventiveMeasures(t.Context(), riskmanager.MeasureFilter{RiskID: riskID})
	seedNoErr(t, err)
	if len(got) != 1 {
		t.Fatalf("measures: %+v", got)
	}
	return got[0].Standing
}

func TestAPassRecordedTwoYearsAgoIsAStaleGapWhenTheBarrierHasARetestWindow(t *testing.T) {
	rm, r, b := staleBarrier(t, 365, riskmanager.OutcomePass, daysAgo(730))
	got := seedGapsOf(t, rm, t.Context(), riskmanager.GapFilter{RiskID: r.ID, Kind: models.GapBarrierStale})
	if len(got) != 1 || got[0].SubjectID != b.ID || got[0].Subject != "barrier" || got[0].Severity != models.SeverityMedium || got[0].Message == "" {
		t.Fatalf("stale gaps: %+v", got)
	}
}

func TestAPassRecordedInsideTheRetestWindowIsNotStale(t *testing.T) {
	rm, r, _ := staleBarrier(t, 365, riskmanager.OutcomePass, daysAgo(364))
	if got := seedGapsOf(t, rm, t.Context(), riskmanager.GapFilter{RiskID: r.ID, Kind: models.GapBarrierStale}); len(got) != 0 {
		t.Fatalf("stale gaps: %+v", got)
	}
}

func TestABarrierWithNoRetestWindowIsNeverStale(t *testing.T) {
	rm, r, _ := staleBarrier(t, 0, riskmanager.OutcomePass, daysAgo(3650))
	if got := seedGapsOf(t, rm, t.Context(), riskmanager.GapFilter{RiskID: r.ID, Kind: models.GapBarrierStale}); len(got) != 0 {
		t.Fatalf("stale gaps: %+v", got)
	}
	if s := standingOf(t, rm, r.ID); s != riskmanager.StandingProven {
		t.Fatalf("standing %s, want proven", s)
	}
}

func TestANewPassClearsTheStaleGap(t *testing.T) {
	rm, r, b := staleBarrier(t, 90, riskmanager.OutcomePass, daysAgo(400))
	seedEvidence(t, rm, t.Context(), b.ID, riskmanager.OutcomePass, daysAgo(1))
	if got := seedGapsOf(t, rm, t.Context(), riskmanager.GapFilter{RiskID: r.ID, Kind: models.GapBarrierStale}); len(got) != 0 {
		t.Fatalf("stale gaps after a fresh pass: %+v", got)
	}
}

func TestAStaleBarriersStandingIsNoBetterThanUnproven(t *testing.T) {
	rm, r, _ := staleBarrier(t, 365, riskmanager.OutcomePass, daysAgo(730))
	if s := standingOf(t, rm, r.ID); s != riskmanager.StandingUnproven {
		t.Fatalf("standing %s, want unproven", s)
	}
}

func TestAStaleFailureStaysFailingAndNotUnproven(t *testing.T) {
	rm, r, _ := staleBarrier(t, 365, riskmanager.OutcomeFail, daysAgo(730))
	if s := standingOf(t, rm, r.ID); s != riskmanager.StandingFailing {
		t.Fatalf("standing %s, want failing", s)
	}
}

func TestAFreshlyTestedBarrierWithAWindowStaysProven(t *testing.T) {
	rm, r, _ := staleBarrier(t, 365, riskmanager.OutcomePass, daysAgo(10))
	if s := standingOf(t, rm, r.ID); s != riskmanager.StandingProven {
		t.Fatalf("standing %s, want proven", s)
	}
}

func TestARetestWindowCanBeSetAndClearedOnAPlacedBarrierButNotMadeNegative(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	b := seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r.ID, "t").ID, seedControl(t, rm, ctx, "c", "o").ID)
	set, err := rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: b.ID, RetestDays: seedPtr(180)})
	seedNoErr(t, err)
	if set.RetestDays != 180 {
		t.Fatalf("retest days %d", set.RetestDays)
	}
	cleared, err := rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: b.ID, RetestDays: seedPtr(0)})
	seedNoErr(t, err)
	if cleared.RetestDays != 0 {
		t.Fatalf("retest days %d", cleared.RetestDays)
	}
	_, err = rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: b.ID, RetestDays: seedPtr(-1)})
	seedWantErr(t, err, riskmanager.ErrInvalid)
}
