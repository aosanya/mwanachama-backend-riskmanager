package riskmanager_test

import (
	"context"
	"errors"
	"testing"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
)

func seedPtr[T any](v T) *T { return &v }

func seedWantErr(t *testing.T, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("error = %v, want one wrapping %v", err, target)
	}
}

func seedNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func seedRisk(t *testing.T, rm riskmanager.RiskManager, ctx context.Context, title string) riskmanager.Risk {
	t.Helper()
	r, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: title, TopEvent: "Control of " + title + " is lost", Owner: "owner-1"})
	seedNoErr(t, err)
	return r
}

func seedOpenRisk(t *testing.T, rm riskmanager.RiskManager, ctx context.Context, title string) riskmanager.Risk {
	t.Helper()
	r := seedRisk(t, rm, ctx, title)
	opened, err := rm.SetRiskStatus(ctx, r.ID, riskmanager.StatusOpen)
	seedNoErr(t, err)
	return opened
}

func seedRetire(t *testing.T, rm riskmanager.RiskManager, ctx context.Context, riskID string) {
	t.Helper()
	_, err := rm.SetRiskStatus(ctx, riskID, riskmanager.StatusRetired)
	seedNoErr(t, err)
}

func seedPathway(t *testing.T, rm riskmanager.RiskManager, ctx context.Context, riskID string, side riskmanager.Side, title string) riskmanager.Pathway {
	t.Helper()
	p, err := rm.CreatePathway(ctx, riskmanager.Pathway{RiskID: riskID, Side: side, Title: title, Scenario: "scenario of " + title})
	seedNoErr(t, err)
	return p
}

func seedThreat(t *testing.T, rm riskmanager.RiskManager, ctx context.Context, riskID, title string) riskmanager.Pathway {
	t.Helper()
	return seedPathway(t, rm, ctx, riskID, riskmanager.SideThreat, title)
}

func seedConsequence(t *testing.T, rm riskmanager.RiskManager, ctx context.Context, riskID, title string) riskmanager.Pathway {
	t.Helper()
	return seedPathway(t, rm, ctx, riskID, riskmanager.SideConsequence, title)
}

func seedControl(t *testing.T, rm riskmanager.RiskManager, ctx context.Context, name, owner string) riskmanager.Control {
	t.Helper()
	c, err := rm.CreateControl(ctx, riskmanager.Control{Name: name, Owner: owner})
	seedNoErr(t, err)
	return c
}

func seedBarrier(t *testing.T, rm riskmanager.RiskManager, ctx context.Context, pathwayID, controlID string) riskmanager.Barrier {
	t.Helper()
	b, err := rm.PlaceBarrier(ctx, riskmanager.Barrier{PathwayID: pathwayID, ControlID: controlID})
	seedNoErr(t, err)
	return b
}

func seedEvidence(t *testing.T, rm riskmanager.RiskManager, ctx context.Context, barrierID string, outcome riskmanager.Outcome, recordedAt string) riskmanager.Evidence {
	t.Helper()
	e, err := rm.RecordEvidence(ctx, riskmanager.Evidence{
		BarrierID: barrierID, Kind: riskmanager.EvidenceTest, Outcome: outcome,
		Summary: "checked, outcome " + string(outcome), RecordedAt: recordedAt,
	})
	seedNoErr(t, err)
	return e
}

func seedSoundBarrier(t *testing.T, rm riskmanager.RiskManager, ctx context.Context, pathwayID, controlID string) riskmanager.Barrier {
	t.Helper()
	b := seedBarrier(t, rm, ctx, pathwayID, controlID)
	seedEvidence(t, rm, ctx, b.ID, riskmanager.OutcomePass, "2026-01-01T00:00:00Z")
	sound, err := rm.UpdateBarrier(ctx, riskmanager.BarrierChange{
		ID:       b.ID,
		Design:   seedPtr(riskmanager.JudgementAdequate),
		Coverage: seedPtr(riskmanager.JudgementAdequate),
		Maturity: seedPtr(riskmanager.MaturityTested),
	})
	seedNoErr(t, err)
	return sound
}

func seedAssessment(t *testing.T, rm riskmanager.RiskManager, ctx context.Context, riskID string, kind riskmanager.AssessmentKind, likelihood, impact int, assessedAt string) riskmanager.Assessment {
	t.Helper()
	a, err := rm.RecordAssessment(ctx, riskmanager.Assessment{
		RiskID: riskID, Kind: kind, Likelihood: likelihood, Impact: impact, AssessedAt: assessedAt,
	})
	seedNoErr(t, err)
	return a
}

func seedAction(t *testing.T, rm riskmanager.RiskManager, ctx context.Context, riskID, barrierID, title, dueAt string) riskmanager.Action {
	t.Helper()
	a, err := rm.CreateAction(ctx, riskmanager.Action{RiskID: riskID, BarrierID: barrierID, Title: title, DueAt: dueAt, Owner: "owner-2"})
	seedNoErr(t, err)
	return a
}

func seedKinds(gaps []riskmanager.Gap) map[riskmanager.GapKind]int {
	out := map[riskmanager.GapKind]int{}
	for _, g := range gaps {
		out[g.Kind]++
	}
	return out
}

func seedSubjects(gaps []riskmanager.Gap, kind riskmanager.GapKind) map[string]bool {
	out := map[string]bool{}
	for _, g := range gaps {
		if g.Kind == kind {
			out[g.SubjectID] = true
		}
	}
	return out
}

func seedCleanRisk(t *testing.T, rm riskmanager.RiskManager, ctx context.Context, title string) riskmanager.Risk {
	t.Helper()
	r := seedOpenRisk(t, rm, ctx, title)
	seedSoundBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r.ID, title+" threat").ID, seedControl(t, rm, ctx, title+" guard", "the team").ID)
	seedSoundBarrier(t, rm, ctx, seedConsequence(t, rm, ctx, r.ID, title+" consequence").ID, seedControl(t, rm, ctx, title+" cleanup", "the team").ID)
	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentInherent, 4, 4, "2026-01-01T00:00:00Z")
	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentResidual, 2, 2, "2026-01-01T00:00:00Z")
	return r
}

func seedGapsOf(t *testing.T, rm riskmanager.RiskManager, ctx context.Context, f riskmanager.GapFilter) []riskmanager.Gap {
	t.Helper()
	got, err := rm.Gaps(ctx, f)
	seedNoErr(t, err)
	return got
}
