package riskmanager_test

import (
	"context"
	"testing"

	rm "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-riskmanager/models"
)

func TestThePackCanListTopRisksGapsAndOverdueActions(t *testing.T) {
	ctx := context.Background()
	m := newManager(t, rm.WithOwners(fakeOwners{"a-1": true}, "actor"))
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
	if _, err := m.PlaceBarrier(ctx, rm.Barrier{PathwayID: cause.ID, ControlID: control.ID}); err != nil {
		t.Fatalf("PlaceBarrier: %v", err)
	}
	overdue, err := m.CreateAction(ctx, rm.Action{RiskID: risk.ID, Title: "Test the cable", Owner: "a-1", DueAt: "2000-01-01T00:00:00Z"})
	if err != nil {
		t.Fatalf("CreateAction: %v", err)
	}
	if _, err := m.CreateAction(ctx, rm.Action{RiskID: risk.ID, Title: "Order spare cable", Owner: "a-1", DueAt: "2999-01-01T00:00:00Z"}); err != nil {
		t.Fatalf("CreateAction later: %v", err)
	}

	risks, err := m.ListRisks(ctx, rm.RiskFilter{Owner: "a-1"})
	if err != nil || len(risks) != 1 || risks[0].ID != risk.ID {
		t.Fatalf("risks of the owner = %+v, %v", risks, err)
	}
	high, err := m.Gaps(ctx, rm.GapFilter{Severity: models.SeverityHigh})
	if err != nil {
		t.Fatalf("Gaps high: %v", err)
	}
	if len(high) == 0 {
		t.Fatalf("no high gaps listed")
	}
	for _, g := range high {
		if g.Severity != models.SeverityHigh {
			t.Fatalf("non-high gap in high list: %+v", g)
		}
	}
	late, err := m.Gaps(ctx, rm.GapFilter{Kind: models.GapActionOverdue})
	if err != nil || len(late) != 1 || late[0].SubjectID != overdue.ID || late[0].RiskID != risk.ID {
		t.Fatalf("overdue actions = %+v, %v", late, err)
	}
	owned, err := m.ListActions(ctx, rm.ActionFilter{Owner: "a-1", Status: rm.ActionOpen})
	if err != nil || len(owned) != 2 {
		t.Fatalf("open actions of the owner = %+v, %v", owned, err)
	}
}
