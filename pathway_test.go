package riskmanager_test

import (
	"testing"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
)

func TestAPathwayNeedsARiskASideAndATitle(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	for name, p := range map[string]riskmanager.Pathway{
		"no side":      {RiskID: r.ID, Title: "t"},
		"a bogus side": {RiskID: r.ID, Side: "sideways", Title: "t"},
		"no title":     {RiskID: r.ID, Side: riskmanager.SideThreat},
		"no risk":      {Side: riskmanager.SideThreat, Title: "t"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := rm.CreatePathway(ctx, p)
			seedWantErr(t, err, riskmanager.ErrInvalid)
		})
	}
}

func TestAPathwayOnAMissingRiskIsAnInvalidReference(t *testing.T) {
	rm, ctx := newManager(t)
	_, err := rm.CreatePathway(ctx, riskmanager.Pathway{RiskID: "nowhere", Side: riskmanager.SideThreat, Title: "t"})
	seedWantErr(t, err, riskmanager.ErrInvalidReference)
}

func TestUpdatingAPathwayNeverMovesItsSideOrItsRisk(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	other := seedRisk(t, rm, ctx, "y")
	p := seedThreat(t, rm, ctx, r.ID, "t")

	updated, err := rm.UpdatePathway(ctx, riskmanager.Pathway{
		ID: p.ID, RiskID: other.ID, Side: riskmanager.SideConsequence, Title: "renamed", Scenario: "new",
		Doc: map[string]any{"k": "v"},
	})
	seedNoErr(t, err)
	held, err := rm.GetPathway(ctx, p.ID)
	seedNoErr(t, err)
	for _, got := range []riskmanager.Pathway{updated, held} {
		if got.Title != "renamed" || got.Scenario != "new" || got.Doc["k"] != "v" {
			t.Fatalf("the update did not land: %+v", got)
		}
		if got.Side != riskmanager.SideThreat || got.RiskID != r.ID {
			t.Fatalf("the body moved the pathway: side=%s risk=%s", got.Side, got.RiskID)
		}
	}
}

func TestUpdatingAPathwayToAnEmptyTitleIsRefused(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	p := seedThreat(t, rm, ctx, r.ID, "t")
	_, err := rm.UpdatePathway(ctx, riskmanager.Pathway{ID: p.ID, Title: ""})
	seedWantErr(t, err, riskmanager.ErrInvalid)
	held, _ := rm.GetPathway(ctx, p.ID)
	if held.Title != "t" {
		t.Fatalf("a refused update changed the title to %q", held.Title)
	}
}

func TestAMissingPathwayIsNotFound(t *testing.T) {
	rm, ctx := newManager(t)
	_, err := rm.GetPathway(ctx, "nobody")
	seedWantErr(t, err, riskmanager.ErrPathwayNotFound)
	_, err = rm.UpdatePathway(ctx, riskmanager.Pathway{ID: "nobody", Title: "t"})
	seedWantErr(t, err, riskmanager.ErrPathwayNotFound)
	seedWantErr(t, rm.DeletePathway(ctx, "nobody"), riskmanager.ErrPathwayNotFound)
}

func TestAPathwayWithNothingUnderItCanBeDeleted(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	p := seedThreat(t, rm, ctx, r.ID, "t")
	seedNoErr(t, rm.DeletePathway(ctx, p.ID))
	_, err := rm.GetPathway(ctx, p.ID)
	seedWantErr(t, err, riskmanager.ErrPathwayNotFound)
}

func TestAPathwayWithABarrierOnItCannotBeDeleted(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	p := seedThreat(t, rm, ctx, r.ID, "t")
	seedBarrier(t, rm, ctx, p.ID, seedControl(t, rm, ctx, "c", "o").ID)
	seedWantErr(t, rm.DeletePathway(ctx, p.ID), riskmanager.ErrConflict)
	if _, err := rm.GetPathway(ctx, p.ID); err != nil {
		t.Fatalf("a refused delete still removed it: %v", err)
	}
}

func TestAPathwayAChildRiskDrillsIntoCannotBeDeleted(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	p := seedThreat(t, rm, ctx, r.ID, "t")
	_, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: "child", TopEvent: "c", ParentPathwayID: p.ID})
	seedNoErr(t, err)
	seedWantErr(t, rm.DeletePathway(ctx, p.ID), riskmanager.ErrConflict)
}

func TestARetiredRiskTakesNoFurtherChangeAnywhereUnderIt(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedOpenRisk(t, rm, ctx, "x")
	t1 := seedThreat(t, rm, ctx, r.ID, "t1")
	t2 := seedThreat(t, rm, ctx, r.ID, "t2")
	guard := seedControl(t, rm, ctx, "guard", "o")
	spare := seedControl(t, rm, ctx, "spare", "o")
	b := seedBarrier(t, rm, ctx, t1.ID, guard.ID)
	a := seedAction(t, rm, ctx, r.ID, "", "work", "2999-01-01T00:00:00Z")
	seedRetire(t, rm, ctx, r.ID)

	for name, call := range map[string]func() error{
		"update the risk": func() error {
			_, err := rm.UpdateRisk(ctx, riskmanager.Risk{ID: r.ID, Title: "n", TopEvent: "n", Owner: "o"})
			return err
		},
		"add a pathway": func() error {
			_, err := rm.CreatePathway(ctx, riskmanager.Pathway{RiskID: r.ID, Side: riskmanager.SideThreat, Title: "late"})
			return err
		},
		"update a pathway": func() error {
			_, err := rm.UpdatePathway(ctx, riskmanager.Pathway{ID: t1.ID, Title: "n"})
			return err
		},
		"delete a pathway": func() error { return rm.DeletePathway(ctx, t2.ID) },
		"place a barrier": func() error {
			_, err := rm.PlaceBarrier(ctx, riskmanager.Barrier{PathwayID: t2.ID, ControlID: spare.ID})
			return err
		},
		"update a barrier": func() error {
			_, err := rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: b.ID, Notes: seedPtr("late")})
			return err
		},
		"remove a barrier": func() error { return rm.RemoveBarrier(ctx, b.ID) },
		"record evidence": func() error {
			_, err := rm.RecordEvidence(ctx, riskmanager.Evidence{
				BarrierID: b.ID, Kind: riskmanager.EvidenceTest, Outcome: riskmanager.OutcomePass, Summary: "late",
			})
			return err
		},
		"record an assessment": func() error {
			_, err := rm.RecordAssessment(ctx, riskmanager.Assessment{RiskID: r.ID, Kind: riskmanager.AssessmentResidual, Likelihood: 1, Impact: 1})
			return err
		},
		"create an action": func() error {
			_, err := rm.CreateAction(ctx, riskmanager.Action{RiskID: r.ID, Title: "late"})
			return err
		},
		"update an action": func() error {
			_, err := rm.UpdateAction(ctx, riskmanager.Action{ID: a.ID, Title: "late"})
			return err
		},
		"move an action": func() error {
			_, err := rm.SetActionStatus(ctx, a.ID, riskmanager.ActionDone)
			return err
		},
		"drill into a pathway": func() error {
			_, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: "c", TopEvent: "c", ParentPathwayID: t1.ID})
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			seedWantErr(t, call(), riskmanager.ErrConflict)
		})
	}
}

func TestARetiredRiskCanStillBeRead(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedOpenRisk(t, rm, ctx, "x")
	p := seedThreat(t, rm, ctx, r.ID, "t")
	seedRetire(t, rm, ctx, r.ID)
	if _, err := rm.GetRisk(ctx, r.ID); err != nil {
		t.Fatalf("get risk: %v", err)
	}
	if _, err := rm.GetPathway(ctx, p.ID); err != nil {
		t.Fatalf("get pathway: %v", err)
	}
	if _, err := rm.Bowtie(ctx, r.ID); err != nil {
		t.Fatalf("bowtie: %v", err)
	}
}
