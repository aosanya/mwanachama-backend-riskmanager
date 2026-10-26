package riskmanager_test

import (
	"testing"
	"time"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
)

func TestANewActionIsOpenAndNeedsARiskAndATitle(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	a, err := rm.CreateAction(ctx, riskmanager.Action{RiskID: r.ID, Title: "do it", Status: riskmanager.ActionDone})
	seedNoErr(t, err)
	if a.Status != riskmanager.ActionOpen || a.ClosedAt != "" {
		t.Fatalf("created as %s closed at %q, want open and not closed", a.Status, a.ClosedAt)
	}
	_, err = rm.CreateAction(ctx, riskmanager.Action{RiskID: r.ID})
	seedWantErr(t, err, riskmanager.ErrInvalid)
	_, err = rm.CreateAction(ctx, riskmanager.Action{Title: "orphan"})
	seedWantErr(t, err, riskmanager.ErrInvalid)
	_, err = rm.CreateAction(ctx, riskmanager.Action{RiskID: "nowhere", Title: "ghost"})
	seedWantErr(t, err, riskmanager.ErrInvalidReference)
}

func TestAnActionWithADueDateThatIsNotRFC3339IsRefused(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	_, err := rm.CreateAction(ctx, riskmanager.Action{RiskID: r.ID, Title: "t", DueAt: "next friday"})
	seedWantErr(t, err, riskmanager.ErrInvalid)
	a := seedAction(t, rm, ctx, r.ID, "", "t", "2026-12-31T00:00:00Z")
	_, err = rm.UpdateAction(ctx, riskmanager.Action{ID: a.ID, Title: "t", DueAt: "soon"})
	seedWantErr(t, err, riskmanager.ErrInvalid)
}

func TestAnActionAboutABarrierOfAnotherRiskOrOfNoBarrierIsAnInvalidReference(t *testing.T) {
	rm, ctx := newManager(t)
	r1 := seedRisk(t, rm, ctx, "one")
	r2 := seedRisk(t, rm, ctx, "two")
	b2 := seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r2.ID, "t").ID, seedControl(t, rm, ctx, "c", "o").ID)
	_, err := rm.CreateAction(ctx, riskmanager.Action{RiskID: r1.ID, BarrierID: b2.ID, Title: "t"})
	seedWantErr(t, err, riskmanager.ErrInvalidReference)
	_, err = rm.CreateAction(ctx, riskmanager.Action{RiskID: r1.ID, BarrierID: "nowhere", Title: "t"})
	seedWantErr(t, err, riskmanager.ErrInvalidReference)
	if _, err := rm.CreateAction(ctx, riskmanager.Action{RiskID: r2.ID, BarrierID: b2.ID, Title: "t"}); err != nil {
		t.Fatalf("an action about its own risk's barrier was refused: %v", err)
	}
}

func TestDoneAndDroppedAreTheEndOfTheRoadAndStampWhenTheyHappened(t *testing.T) {
	for _, end := range []riskmanager.ActionStatus{riskmanager.ActionDone, riskmanager.ActionDropped} {
		t.Run(string(end), func(t *testing.T) {
			rm, ctx := newManager(t)
			r := seedRisk(t, rm, ctx, "x")
			a := seedAction(t, rm, ctx, r.ID, "", "t", "2026-12-31T00:00:00Z")
			moving, err := rm.SetActionStatus(ctx, a.ID, riskmanager.ActionInProgress)
			seedNoErr(t, err)
			if moving.ClosedAt != "" {
				t.Fatalf("closed_at %q while still going", moving.ClosedAt)
			}
			closed, err := rm.SetActionStatus(ctx, a.ID, end)
			seedNoErr(t, err)
			at, perr := time.Parse(time.RFC3339, closed.ClosedAt)
			if perr != nil || time.Since(at) > 5*time.Second {
				t.Fatalf("closed_at %q (%v) is not about now", closed.ClosedAt, perr)
			}
			for _, next := range []riskmanager.ActionStatus{
				riskmanager.ActionOpen, riskmanager.ActionInProgress, riskmanager.ActionDone, riskmanager.ActionDropped,
			} {
				_, err := rm.SetActionStatus(ctx, a.ID, next)
				seedWantErr(t, err, riskmanager.ErrInvalid)
			}
			held, _ := rm.ListActions(ctx, riskmanager.ActionFilter{RiskID: r.ID})
			if held[0].Status != end || held[0].ClosedAt != closed.ClosedAt {
				t.Fatalf("a refused move changed the action: %+v", held[0])
			}
		})
	}
}

func TestAnActionMayStepBackFromInProgressToOpenButNotPastTheEnd(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	a := seedAction(t, rm, ctx, r.ID, "", "t", "2026-12-31T00:00:00Z")
	_, err := rm.SetActionStatus(ctx, a.ID, riskmanager.ActionInProgress)
	seedNoErr(t, err)
	back, err := rm.SetActionStatus(ctx, a.ID, riskmanager.ActionOpen)
	seedNoErr(t, err)
	if back.Status != riskmanager.ActionOpen {
		t.Fatalf("status = %s", back.Status)
	}
	_, err = rm.SetActionStatus(ctx, a.ID, "finished")
	seedWantErr(t, err, riskmanager.ErrInvalid)
	_, err = rm.SetActionStatus(ctx, a.ID, "")
	seedWantErr(t, err, riskmanager.ErrInvalid)
}

func TestAClosedActionTakesNoFurtherEdits(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	a := seedAction(t, rm, ctx, r.ID, "", "t", "2026-12-31T00:00:00Z")
	_, err := rm.SetActionStatus(ctx, a.ID, riskmanager.ActionDone)
	seedNoErr(t, err)
	_, err = rm.UpdateAction(ctx, riskmanager.Action{ID: a.ID, Title: "rewritten"})
	seedWantErr(t, err, riskmanager.ErrConflict)
	held, _ := rm.ListActions(ctx, riskmanager.ActionFilter{RiskID: r.ID})
	if held[0].Title != "t" {
		t.Fatalf("a refused edit changed the title to %q", held[0].Title)
	}
}

func TestUpdatingAnActionNeverMovesItsRiskItsBarrierItsStatusOrItsClosedAt(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	other := seedRisk(t, rm, ctx, "y")
	b := seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r.ID, "t").ID, seedControl(t, rm, ctx, "c", "o").ID)
	a := seedAction(t, rm, ctx, r.ID, b.ID, "t", "2026-12-31T00:00:00Z")

	updated, err := rm.UpdateAction(ctx, riskmanager.Action{
		ID: a.ID, RiskID: other.ID, BarrierID: "", Title: "renamed", Description: "d", Owner: "o3",
		DueAt: "2027-01-01T00:00:00Z", Status: riskmanager.ActionDone, ClosedAt: "2026-01-01T00:00:00Z",
	})
	seedNoErr(t, err)
	if updated.Title != "renamed" || updated.Description != "d" || updated.Owner != "o3" || updated.DueAt != "2027-01-01T00:00:00Z" {
		t.Fatalf("the update did not land: %+v", updated)
	}
	if updated.RiskID != r.ID || updated.BarrierID != b.ID {
		t.Fatalf("the body moved the action: risk=%s barrier=%s", updated.RiskID, updated.BarrierID)
	}
	if updated.Status != riskmanager.ActionOpen || updated.ClosedAt != "" {
		t.Fatalf("the body closed the action: %s %q", updated.Status, updated.ClosedAt)
	}
}

func TestAMissingActionIsNotFound(t *testing.T) {
	rm, ctx := newManager(t)
	_, err := rm.UpdateAction(ctx, riskmanager.Action{ID: "nobody", Title: "t"})
	seedWantErr(t, err, riskmanager.ErrActionNotFound)
	_, err = rm.SetActionStatus(ctx, "nobody", riskmanager.ActionDone)
	seedWantErr(t, err, riskmanager.ErrActionNotFound)
}

func TestActionsFilterByRiskBarrierAndStatus(t *testing.T) {
	rm, ctx := newManager(t)
	r1 := seedRisk(t, rm, ctx, "one")
	r2 := seedRisk(t, rm, ctx, "two")
	b := seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r1.ID, "t").ID, seedControl(t, rm, ctx, "c", "o").ID)
	seedAction(t, rm, ctx, r1.ID, b.ID, "a", "2026-12-31T00:00:00Z")
	moving := seedAction(t, rm, ctx, r1.ID, "", "b", "2026-12-31T00:00:00Z")
	seedAction(t, rm, ctx, r2.ID, "", "c", "2026-12-31T00:00:00Z")
	_, err := rm.SetActionStatus(ctx, moving.ID, riskmanager.ActionInProgress)
	seedNoErr(t, err)

	count := func(f riskmanager.ActionFilter) int {
		got, err := rm.ListActions(ctx, f)
		seedNoErr(t, err)
		return len(got)
	}
	if n := count(riskmanager.ActionFilter{}); n != 3 {
		t.Fatalf("all: %d", n)
	}
	if n := count(riskmanager.ActionFilter{RiskID: r1.ID}); n != 2 {
		t.Fatalf("risk one: %d", n)
	}
	if n := count(riskmanager.ActionFilter{BarrierID: b.ID}); n != 1 {
		t.Fatalf("about the barrier: %d", n)
	}
	if n := count(riskmanager.ActionFilter{Status: riskmanager.ActionInProgress}); n != 1 {
		t.Fatalf("in progress: %d", n)
	}
	if n := count(riskmanager.ActionFilter{RiskID: r2.ID, Status: riskmanager.ActionInProgress}); n != 0 {
		t.Fatalf("in progress on risk two: %d", n)
	}
}
