package riskmanager_test

import (
	"context"
	"testing"
	"time"

	rm "github.com/aosanya/mwanachama-backend-riskmanager"
)

func TestRecordingEvidenceMovesASafeguardsStanding(t *testing.T) {
	ctx := context.Background()
	m := newManager(t)
	risk, err := m.CreateRisk(ctx, rm.Risk{Title: "Water tower freezes", TopEvent: "Water tower freezes in winter"})
	if err != nil {
		t.Fatalf("CreateRisk: %v", err)
	}
	cause, err := m.CreatePathway(ctx, rm.Pathway{RiskID: risk.ID, Side: rm.SideThreat, Title: "Long frost"})
	if err != nil {
		t.Fatalf("CreatePathway: %v", err)
	}
	control, err := m.CreateControl(ctx, rm.Control{Name: "Backup heating cable"})
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	barrier, err := m.PlaceBarrier(ctx, rm.Barrier{PathwayID: cause.ID, ControlID: control.ID})
	if err != nil {
		t.Fatalf("PlaceBarrier: %v", err)
	}
	adequate, days := rm.JudgementAdequate, 30
	if _, err := m.UpdateBarrier(ctx, rm.BarrierChange{ID: barrier.ID, Design: &adequate, Coverage: &adequate, RetestDays: &days}); err != nil {
		t.Fatalf("UpdateBarrier: %v", err)
	}
	standing := func() rm.Standing {
		t.Helper()
		measures, err := m.PreventiveMeasures(ctx, rm.MeasureFilter{RiskID: risk.ID})
		if err != nil || len(measures) != 1 {
			t.Fatalf("PreventiveMeasures = %v, %v", measures, err)
		}
		return measures[0].Standing
	}
	staleGaps := func() int {
		t.Helper()
		gaps, err := m.Gaps(ctx, rm.GapFilter{})
		if err != nil {
			t.Fatalf("Gaps: %v", err)
		}
		n := 0
		for _, g := range gaps {
			if string(g.Kind) == "barrier_stale" && g.SubjectID == barrier.ID {
				n++
			}
		}
		return n
	}

	if got := standing(); got != rm.StandingUnproven {
		t.Fatalf("before evidence the standing is %s, want unproven", got)
	}
	if _, err := m.RecordEvidence(ctx, rm.Evidence{BarrierID: barrier.ID, Kind: rm.EvidenceTest, Outcome: rm.OutcomePass, Summary: "Szintetikus próba"}); err != nil {
		t.Fatalf("RecordEvidence: %v", err)
	}
	if got := standing(); got != rm.StandingProven {
		t.Fatalf("after a passing report the standing is %s, want proven", got)
	}
	if n := staleGaps(); n != 0 {
		t.Fatalf("fresh evidence shows %d stale gaps", n)
	}
	if _, err := m.RecordEvidence(ctx, rm.Evidence{BarrierID: barrier.ID, Kind: rm.EvidenceTest, Outcome: rm.OutcomeFail, Summary: "Szintetikus hiba"}); err != nil {
		t.Fatalf("RecordEvidence: %v", err)
	}
	if got := standing(); got != rm.StandingFailing {
		t.Fatalf("after a failing report the standing is %s, want failing", got)
	}

	old := newManager(t)
	oldRisk, _ := old.CreateRisk(ctx, rm.Risk{Title: "Old", TopEvent: "Old"})
	oldPath, _ := old.CreatePathway(ctx, rm.Pathway{RiskID: oldRisk.ID, Side: rm.SideThreat, Title: "Cause"})
	oldControl, _ := old.CreateControl(ctx, rm.Control{Name: "Control"})
	oldBarrier, err := old.PlaceBarrier(ctx, rm.Barrier{PathwayID: oldPath.ID, ControlID: oldControl.ID})
	if err != nil {
		t.Fatalf("PlaceBarrier: %v", err)
	}
	if _, err := old.UpdateBarrier(ctx, rm.BarrierChange{ID: oldBarrier.ID, Design: &adequate, Coverage: &adequate, RetestDays: &days}); err != nil {
		t.Fatalf("UpdateBarrier: %v", err)
	}
	when := time.Now().UTC().AddDate(0, 0, -90).Format(time.RFC3339)
	if _, err := old.RecordEvidence(ctx, rm.Evidence{BarrierID: oldBarrier.ID, Kind: rm.EvidenceTest, Outcome: rm.OutcomePass, Summary: "Régi próba", RecordedAt: when}); err != nil {
		t.Fatalf("RecordEvidence: %v", err)
	}
	measures, err := old.PreventiveMeasures(ctx, rm.MeasureFilter{RiskID: oldRisk.ID})
	if err != nil || len(measures) != 1 || measures[0].Standing != rm.StandingUnproven {
		t.Fatalf("stale evidence measures = %v, %v; want unproven", measures, err)
	}
	gaps, err := old.Gaps(ctx, rm.GapFilter{})
	if err != nil {
		t.Fatalf("Gaps: %v", err)
	}
	found := false
	for _, g := range gaps {
		found = found || (string(g.Kind) == "barrier_stale" && g.SubjectID == oldBarrier.ID)
	}
	if !found {
		t.Fatalf("stale evidence is not on the gaps list: %v", gaps)
	}
}
