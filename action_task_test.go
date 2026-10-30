package riskmanager_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/spec"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
)

const taskRef = "taskmanager/hub/task-1"

func TestAnActionCanCarryTheReferenceOfTheTaskThatTracksIt(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "r")
	a, err := rm.CreateAction(ctx, riskmanager.Action{RiskID: r.ID, Title: "w", TaskRef: taskRef})
	seedNoErr(t, err)
	if a.TaskRef != taskRef {
		t.Fatalf("created %q", a.TaskRef)
	}
	plain := seedAction(t, rm, ctx, r.ID, "", "no task", "")
	a.Title = "renamed"
	a.TaskRef = "taskmanager/hub/task-2"
	moved, err := rm.UpdateAction(ctx, a)
	seedNoErr(t, err)
	if moved.TaskRef != "taskmanager/hub/task-2" {
		t.Fatalf("updated %q", moved.TaskRef)
	}
	tracked, err := rm.ListActions(ctx, riskmanager.ActionFilter{TaskRef: "taskmanager/hub/task-2"})
	seedNoErr(t, err)
	if len(tracked) != 1 || tracked[0].ID != a.ID || tracked[0].ID == plain.ID {
		t.Fatalf("actions tracked by the task: %+v", tracked)
	}
	cleared, err := rm.UpdateAction(ctx, riskmanager.Action{ID: a.ID, Title: "renamed"})
	seedNoErr(t, err)
	if cleared.TaskRef != "" {
		t.Fatalf("cleared %q", cleared.TaskRef)
	}
}

func TestATaskTheTaskRegisterDoesNotKnowIsRefusedWhenOneIsConfigured(t *testing.T) {
	db := newDB(t)
	s, err := riskmanager.LoadSpec(examplePath("ai_safety"))
	seedNoErr(t, err)
	seedNoErr(t, spec.Migrate(db, s))
	rm, err := riskmanager.NewRiskManager(db, s, riskmanager.WithTasks(directory{taskRef: "task"}))
	seedNoErr(t, err)
	ctx := context.Background()
	r, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: "r", TopEvent: "e"})
	seedNoErr(t, err)
	_, err = rm.CreateAction(ctx, riskmanager.Action{RiskID: r.ID, Title: "w", TaskRef: "taskmanager/hub/missing"})
	seedWantErr(t, err, riskmanager.ErrInvalidReference)
	_, err = rm.CreateAction(ctx, riskmanager.Action{RiskID: r.ID, Title: "w", TaskRef: taskRef})
	seedNoErr(t, err)
}
