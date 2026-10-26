package riskmanager_test

import (
	"context"
	"reflect"
	"regexp"
	"testing"
	"time"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
)

func seedGuarded(t *testing.T, rm riskmanager.RiskManager, ctx context.Context) riskmanager.Barrier {
	t.Helper()
	r := seedRisk(t, rm, ctx, "x")
	p := seedThreat(t, rm, ctx, r.ID, "t")
	return seedBarrier(t, rm, ctx, p.ID, seedControl(t, rm, ctx, "c", "o").ID)
}

func TestTheManagerOffersNoWayToEditOrDeleteEvidenceOrAssessments(t *testing.T) {
	forbidden := regexp.MustCompile(`^(Update|Delete|Remove|Edit|Amend|Purge|Set)(Evidence|Assessment)s?$`)
	ty := reflect.TypeOf((*riskmanager.RiskManager)(nil)).Elem()
	for i := 0; i < ty.NumMethod(); i++ {
		if forbidden.MatchString(ty.Method(i).Name) {
			t.Fatalf("%s lets a record that is meant to be append-only be rewritten", ty.Method(i).Name)
		}
	}
}

func TestEvidenceNamingAMissingBarrierIsAnInvalidReference(t *testing.T) {
	rm, ctx := newManager(t)
	_, err := rm.RecordEvidence(ctx, riskmanager.Evidence{
		BarrierID: "nowhere", Kind: riskmanager.EvidenceTest, Outcome: riskmanager.OutcomePass, Summary: "s",
	})
	seedWantErr(t, err, riskmanager.ErrInvalidReference)
}

func TestEvidenceWithoutASummaryAKindOrAnOutcomeIsRefused(t *testing.T) {
	rm, ctx := newManager(t)
	b := seedGuarded(t, rm, ctx)
	good := riskmanager.Evidence{BarrierID: b.ID, Kind: riskmanager.EvidenceAudit, Outcome: riskmanager.OutcomePass, Summary: "s"}
	for name, mutate := range map[string]func(e *riskmanager.Evidence){
		"no summary":    func(e *riskmanager.Evidence) { e.Summary = "" },
		"blank summary": func(e *riskmanager.Evidence) { e.Summary = "  " },
		"no kind":       func(e *riskmanager.Evidence) { e.Kind = "" },
		"a bogus kind":  func(e *riskmanager.Evidence) { e.Kind = "vibes" },
		"no outcome":    func(e *riskmanager.Evidence) { e.Outcome = "" },
		"a bogus outcome": func(e *riskmanager.Evidence) {
			e.Outcome = "mostly"
		},
		"no barrier": func(e *riskmanager.Evidence) { e.BarrierID = "" },
	} {
		t.Run(name, func(t *testing.T) {
			e := good
			mutate(&e)
			_, err := rm.RecordEvidence(ctx, e)
			seedWantErr(t, err, riskmanager.ErrInvalid)
		})
	}
	if _, err := rm.RecordEvidence(ctx, good); err != nil {
		t.Fatalf("the good evidence was refused: %v", err)
	}
}

func TestEvidenceTakesTheTimeItArrivedWhenNoneIsGiven(t *testing.T) {
	rm, ctx := newManager(t)
	b := seedGuarded(t, rm, ctx)
	before := time.Now().UTC().Add(-2 * time.Second)
	e, err := rm.RecordEvidence(ctx, riskmanager.Evidence{
		BarrierID: b.ID, Kind: riskmanager.EvidenceLog, Outcome: riskmanager.OutcomePass, Summary: "s",
	})
	seedNoErr(t, err)
	at, err := time.Parse(time.RFC3339, e.RecordedAt)
	if err != nil {
		t.Fatalf("recorded_at %q is not RFC 3339: %v", e.RecordedAt, err)
	}
	if at.Before(before) || at.After(time.Now().UTC().Add(2*time.Second)) {
		t.Fatalf("recorded_at %s is not about now", e.RecordedAt)
	}
	if e.CreatedAt == "" {
		t.Fatal("created_at is empty")
	}
}

func TestEvidenceWithATimeThatIsNotRFC3339IsRefused(t *testing.T) {
	rm, ctx := newManager(t)
	b := seedGuarded(t, rm, ctx)
	for _, bad := range []string{"yesterday", "2026-01-01", "2026-01-01 10:00:00", "01/02/2026"} {
		_, err := rm.RecordEvidence(ctx, riskmanager.Evidence{
			BarrierID: b.ID, Kind: riskmanager.EvidenceTest, Outcome: riskmanager.OutcomePass, Summary: "s", RecordedAt: bad,
		})
		seedWantErr(t, err, riskmanager.ErrInvalid)
	}
}

func TestEvidenceIsListedByWhenItWasTakenNotByWhenItWasEntered(t *testing.T) {
	rm, ctx := newManager(t)
	b := seedGuarded(t, rm, ctx)
	for _, at := range []string{"2026-03-01T00:00:00Z", "2026-01-01T00:00:00Z", "2026-02-01T00:00:00Z"} {
		seedEvidence(t, rm, ctx, b.ID, riskmanager.OutcomePass, at)
	}
	got, err := rm.ListEvidence(ctx, riskmanager.EvidenceFilter{BarrierID: b.ID})
	seedNoErr(t, err)
	want := []string{"2026-01-01T00:00:00Z", "2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z"}
	if len(got) != 3 {
		t.Fatalf("%d pieces of evidence", len(got))
	}
	for i, e := range got {
		if e.RecordedAt != want[i] {
			t.Fatalf("evidence %d was taken %s, want %s", i, e.RecordedAt, want[i])
		}
	}
}

func TestEvidenceFiltersByBarrierAndByTheWindowItWasTakenIn(t *testing.T) {
	rm, ctx := newManager(t)
	b1 := seedGuarded(t, rm, ctx)
	r2 := seedRisk(t, rm, ctx, "y")
	b2 := seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r2.ID, "t2").ID, seedControl(t, rm, ctx, "c2", "o").ID)
	for _, at := range []string{"2026-01-01T00:00:00Z", "2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z"} {
		seedEvidence(t, rm, ctx, b1.ID, riskmanager.OutcomePass, at)
	}
	seedEvidence(t, rm, ctx, b2.ID, riskmanager.OutcomeFail, "2026-02-01T00:00:00Z")

	count := func(f riskmanager.EvidenceFilter) int {
		got, err := rm.ListEvidence(ctx, f)
		seedNoErr(t, err)
		return len(got)
	}
	if n := count(riskmanager.EvidenceFilter{}); n != 4 {
		t.Fatalf("all: %d", n)
	}
	if n := count(riskmanager.EvidenceFilter{BarrierID: b2.ID}); n != 1 {
		t.Fatalf("second barrier: %d", n)
	}
	if n := count(riskmanager.EvidenceFilter{BarrierID: b1.ID, From: "2026-02-01T00:00:00Z"}); n != 2 {
		t.Fatalf("from february, inclusive: %d", n)
	}
	if n := count(riskmanager.EvidenceFilter{BarrierID: b1.ID, To: "2026-02-01T00:00:00Z"}); n != 2 {
		t.Fatalf("to february, inclusive: %d", n)
	}
	if n := count(riskmanager.EvidenceFilter{BarrierID: b1.ID, From: "2026-01-15T00:00:00Z", To: "2026-02-15T00:00:00Z"}); n != 1 {
		t.Fatalf("a window around february: %d", n)
	}
	if n := count(riskmanager.EvidenceFilter{BarrierID: b1.ID, From: "not a time"}); n != 3 {
		t.Fatalf("an unparseable bound should leave that side open: %d", n)
	}
}

func TestAFailingPieceOfEvidenceDoesNotRewriteThePassingOneBeforeIt(t *testing.T) {
	rm, ctx := newManager(t)
	b := seedGuarded(t, rm, ctx)
	first := seedEvidence(t, rm, ctx, b.ID, riskmanager.OutcomePass, "2026-01-01T00:00:00Z")
	second := seedEvidence(t, rm, ctx, b.ID, riskmanager.OutcomeFail, "2026-02-01T00:00:00Z")
	got, err := rm.ListEvidence(ctx, riskmanager.EvidenceFilter{BarrierID: b.ID})
	seedNoErr(t, err)
	if len(got) != 2 || got[0].ID != first.ID || got[0].Outcome != riskmanager.OutcomePass || got[1].ID != second.ID {
		t.Fatalf("history was disturbed: %+v", got)
	}
}

type seedEvidenceAt struct {
	outcome riskmanager.Outcome
	at      string
}

func TestAMeasureStandsProvenWeakUnprovenOrFailingByWhatTheLatestEvidenceShows(t *testing.T) {
	adequate := riskmanager.JudgementAdequate
	for _, c := range []struct {
		name             string
		design, coverage riskmanager.Judgement
		evidence         []seedEvidenceAt
		want             riskmanager.Standing
	}{
		{"nothing on record", adequate, adequate, nil, riskmanager.StandingUnproven},
		{"a failing test", adequate, adequate, []seedEvidenceAt{{riskmanager.OutcomeFail, "2026-01-01T00:00:00Z"}}, riskmanager.StandingFailing},
		{"a partial result", adequate, adequate, []seedEvidenceAt{{riskmanager.OutcomePartial, "2026-01-01T00:00:00Z"}}, riskmanager.StandingFailing},
		{"a pass with a partial design", riskmanager.JudgementPartial, adequate, []seedEvidenceAt{{riskmanager.OutcomePass, "2026-01-01T00:00:00Z"}}, riskmanager.StandingWeak},
		{"a pass with inadequate coverage", adequate, riskmanager.JudgementInadequate, []seedEvidenceAt{{riskmanager.OutcomePass, "2026-01-01T00:00:00Z"}}, riskmanager.StandingWeak},
		{"a pass with nothing assessed", riskmanager.JudgementUnassessed, riskmanager.JudgementUnassessed, []seedEvidenceAt{{riskmanager.OutcomePass, "2026-01-01T00:00:00Z"}}, riskmanager.StandingWeak},
		{"a pass and both judgements adequate", adequate, adequate, []seedEvidenceAt{{riskmanager.OutcomePass, "2026-01-01T00:00:00Z"}}, riskmanager.StandingProven},
		{"a pass and then a failure", adequate, adequate, []seedEvidenceAt{{riskmanager.OutcomePass, "2026-01-01T00:00:00Z"}, {riskmanager.OutcomeFail, "2026-02-01T00:00:00Z"}}, riskmanager.StandingFailing},
		{"a failure and then a pass", adequate, adequate, []seedEvidenceAt{{riskmanager.OutcomeFail, "2026-01-01T00:00:00Z"}, {riskmanager.OutcomePass, "2026-02-01T00:00:00Z"}}, riskmanager.StandingProven},
		{"a failure entered before the pass it followed", adequate, adequate, []seedEvidenceAt{{riskmanager.OutcomeFail, "2026-02-01T00:00:00Z"}, {riskmanager.OutcomePass, "2026-01-01T00:00:00Z"}}, riskmanager.StandingFailing},
		{"a failure beats a weak design", riskmanager.JudgementPartial, adequate, []seedEvidenceAt{{riskmanager.OutcomeFail, "2026-01-01T00:00:00Z"}}, riskmanager.StandingFailing},
		{"no evidence beats a weak design", riskmanager.JudgementPartial, adequate, nil, riskmanager.StandingUnproven},
	} {
		t.Run(c.name, func(t *testing.T) {
			rm, ctx := newManager(t)
			r := seedRisk(t, rm, ctx, "x")
			p := seedThreat(t, rm, ctx, r.ID, "t")
			b := seedBarrier(t, rm, ctx, p.ID, seedControl(t, rm, ctx, "c", "o").ID)
			_, err := rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: b.ID, Design: &c.design, Coverage: &c.coverage})
			seedNoErr(t, err)
			for _, e := range c.evidence {
				seedEvidence(t, rm, ctx, b.ID, e.outcome, e.at)
			}
			got, err := rm.PreventiveMeasures(ctx, riskmanager.MeasureFilter{PathwayID: p.ID})
			seedNoErr(t, err)
			if len(got) != 1 || got[0].Standing != c.want {
				t.Fatalf("standing = %+v, want %s", got, c.want)
			}
		})
	}
}
