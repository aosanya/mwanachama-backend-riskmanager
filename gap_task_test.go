package riskmanager_test

import (
	"context"
	"errors"
	"testing"

	rm "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-riskmanager/models"
	"github.com/aosanya/mwanachama-backend-shared/identity"
)

type fakeTasks map[string]bool

func (f fakeTasks) StableIdentity(ctx context.Context, object, id string) (string, error) {
	return id, nil
}

func (f fakeTasks) Resolve(ctx context.Context, raw string) (string, string, error) {
	if !f[raw] {
		return "", "", identity.ErrNotFound
	}
	return "task", raw, nil
}

func gapKinds(gaps []rm.Gap, subjectID string) []rm.GapKind {
	var kinds []rm.GapKind
	for _, g := range gaps {
		if g.SubjectID == subjectID {
			kinds = append(kinds, g.Kind)
		}
	}
	return kinds
}

func TestAGapBecomesATaskAndLeavesTheListWhenTheSafeguardIsAssessed(t *testing.T) {
	ctx := context.Background()
	tasks := fakeTasks{"task-1": true}
	m := newManager(t, rm.WithOwners(fakeOwners{"a-1": true}, "actor"), rm.WithTasks(tasks))
	risk, err := m.CreateRisk(ctx, rm.Risk{Title: "Water tower freezes", TopEvent: "Water tower freezes in winter", Owner: "a-1"})
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

	gaps, err := m.Gaps(ctx, rm.GapFilter{RiskID: risk.ID})
	if err != nil {
		t.Fatalf("Gaps: %v", err)
	}
	if kinds := gapKinds(gaps, barrier.ID); len(kinds) == 0 {
		t.Fatalf("no gap on the unassessed safeguard: %+v", gaps)
	}

	if _, err := m.CreateAction(ctx, rm.Action{RiskID: risk.ID, BarrierID: barrier.ID, Title: "Assess the cable", Owner: "a-1", TaskRef: "task-ghost", DueAt: "2026-12-01T00:00:00Z"}); !errors.Is(err, rm.ErrInvalidReference) {
		t.Fatalf("unknown task err = %v", err)
	}
	action, err := m.CreateAction(ctx, rm.Action{RiskID: risk.ID, BarrierID: barrier.ID, Title: "Assess the cable", Owner: "a-1", TaskRef: "task-1", DueAt: "2026-12-01T00:00:00Z"})
	if err != nil {
		t.Fatalf("CreateAction: %v", err)
	}
	byTask, err := m.ListActions(ctx, rm.ActionFilter{TaskRef: "task-1"})
	if err != nil || len(byTask) != 1 || byTask[0].ID != action.ID || byTask[0].Status != rm.ActionOpen || byTask[0].Owner != "a-1" || byTask[0].DueAt == "" {
		t.Fatalf("actions of the task = %+v, %v", byTask, err)
	}

	done, err := m.SetActionStatus(ctx, action.ID, rm.ActionDone)
	if err != nil || done.Status != rm.ActionDone || done.ClosedAt == "" {
		t.Fatalf("SetActionStatus = %+v, %v", done, err)
	}
	closed, err := m.ListActions(ctx, rm.ActionFilter{TaskRef: "task-1", Status: rm.ActionDone})
	if err != nil || len(closed) != 1 || closed[0].ID != action.ID {
		t.Fatalf("closed actions of the task = %+v, %v", closed, err)
	}
	if open, _ := m.ListActions(ctx, rm.ActionFilter{TaskRef: "task-1", Status: rm.ActionOpen}); len(open) != 0 {
		t.Fatalf("open actions after closing = %+v", open)
	}

	adequate := rm.JudgementAdequate
	if _, err := m.UpdateBarrier(ctx, rm.BarrierChange{ID: barrier.ID, Design: &adequate, Coverage: &adequate}); err != nil {
		t.Fatalf("UpdateBarrier: %v", err)
	}
	after, err := m.Gaps(ctx, rm.GapFilter{RiskID: risk.ID})
	if err != nil {
		t.Fatalf("Gaps after: %v", err)
	}
	for _, g := range after {
		if g.SubjectID == barrier.ID && g.Kind == models.GapBarrierUnassessed {
			t.Fatalf("assessed safeguard still listed: %+v", g)
		}
	}
}
