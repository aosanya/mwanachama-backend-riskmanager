package riskmanager_test

import (
	"encoding/json"
	"testing"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-riskmanager/models"
)

func TestABowTieShowsBothArmsWithTheBarriersOnEachInPositionOrder(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	other := seedRisk(t, rm, ctx, "elsewhere")
	t1 := seedThreat(t, rm, ctx, r.ID, "t1")
	t2 := seedThreat(t, rm, ctx, r.ID, "t2")
	c1 := seedConsequence(t, rm, ctx, r.ID, "c1")
	seedThreat(t, rm, ctx, other.ID, "not mine")

	late := seedBarrier(t, rm, ctx, t1.ID, seedControl(t, rm, ctx, "late", "o").ID)
	early := seedBarrier(t, rm, ctx, t1.ID, seedControl(t, rm, ctx, "early", "o").ID)
	middle := seedBarrier(t, rm, ctx, t1.ID, seedControl(t, rm, ctx, "middle", "o").ID)
	for b, pos := range map[string]int{late.ID: 3, early.ID: 1, middle.ID: 2} {
		_, err := rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: b, Position: seedPtr(pos)})
		seedNoErr(t, err)
	}
	seedBarrier(t, rm, ctx, c1.ID, seedControl(t, rm, ctx, "cleanup", "o").ID)

	bt, err := rm.Bowtie(ctx, r.ID)
	seedNoErr(t, err)
	if bt.Risk.ID != r.ID {
		t.Fatalf("risk = %s", bt.Risk.ID)
	}
	if len(bt.Threats) != 2 || bt.Threats[0].Pathway.ID != t1.ID || bt.Threats[1].Pathway.ID != t2.ID {
		t.Fatalf("threat arms: %+v", bt.Threats)
	}
	if len(bt.Consequences) != 1 || bt.Consequences[0].Pathway.ID != c1.ID {
		t.Fatalf("consequence arms: %+v", bt.Consequences)
	}
	names := []string{}
	for _, p := range bt.Threats[0].Barriers {
		names = append(names, p.Control.Name)
	}
	if len(names) != 3 || names[0] != "early" || names[1] != "middle" || names[2] != "late" {
		t.Fatalf("barriers on the first threat in order: %v", names)
	}
	if len(bt.Threats[1].Barriers) != 0 || len(bt.Consequences[0].Barriers) != 1 {
		t.Fatalf("barriers leaked between arms: %+v / %+v", bt.Threats[1].Barriers, bt.Consequences[0].Barriers)
	}
}

func TestEachBarrierOnTheBowTieShowsItsControlItsLatestEvidenceAndHowMuchThereIs(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	p := seedThreat(t, rm, ctx, r.ID, "t")
	c := seedControl(t, rm, ctx, "guard", "the team")
	b := seedBarrier(t, rm, ctx, p.ID, c.ID)
	seedBarrier(t, rm, ctx, p.ID, seedControl(t, rm, ctx, "bare", "o").ID)
	seedEvidence(t, rm, ctx, b.ID, riskmanager.OutcomePass, "2026-01-01T00:00:00Z")
	newest := seedEvidence(t, rm, ctx, b.ID, riskmanager.OutcomeFail, "2026-03-01T00:00:00Z")
	seedEvidence(t, rm, ctx, b.ID, riskmanager.OutcomePass, "2026-02-01T00:00:00Z")

	bt, err := rm.Bowtie(ctx, r.ID)
	seedNoErr(t, err)
	placed := map[string]models.Placed{}
	for _, pl := range bt.Threats[0].Barriers {
		placed[pl.Control.Name] = pl
	}
	guard := placed["guard"]
	if guard.Control.ID != c.ID || guard.Control.Owner != "the team" || guard.Barrier.ID != b.ID {
		t.Fatalf("the barrier lost its control: %+v", guard)
	}
	if guard.EvidenceCount != 3 || guard.Latest == nil || guard.Latest.ID != newest.ID || guard.Latest.Outcome != riskmanager.OutcomeFail {
		t.Fatalf("latest evidence = %+v of %d, want the March failure of 3", guard.Latest, guard.EvidenceCount)
	}
	bare := placed["bare"]
	if bare.EvidenceCount != 0 || bare.Latest != nil {
		t.Fatalf("a barrier with nothing on record shows %+v of %d", bare.Latest, bare.EvidenceCount)
	}
}

func TestTheBowTieShowsTheLatestRatingOfEachKindWithItsScoreAndBand(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentInherent, 5, 4, "2026-01-01T00:00:00Z")
	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentResidual, 3, 3, "2026-01-01T00:00:00Z")
	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentTarget, 1, 3, "2026-01-01T00:00:00Z")
	bt, err := rm.Bowtie(ctx, r.ID)
	seedNoErr(t, err)
	for name, c := range map[string]struct {
		got   *models.Rated
		score int
		band  riskmanager.Band
	}{
		"inherent": {bt.Inherent, 20, models.BandCritical},
		"residual": {bt.Residual, 9, models.BandMedium},
		"target":   {bt.Target, 3, models.BandLow},
	} {
		if c.got == nil || c.got.Score != c.score || c.got.Band != c.band {
			t.Fatalf("%s = %+v, want %d %s", name, c.got, c.score, c.band)
		}
	}
}

func TestARiskThatDrillsIntoAPathwayShowsUpOnThatArmAndOnNoOther(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	drilled := seedThreat(t, rm, ctx, r.ID, "drilled")
	plain := seedThreat(t, rm, ctx, r.ID, "plain")
	child, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: "child", TopEvent: "c", ParentPathwayID: drilled.ID})
	seedNoErr(t, err)

	bt, err := rm.Bowtie(ctx, r.ID)
	seedNoErr(t, err)
	arms := map[string]models.Arm{}
	for _, a := range bt.Threats {
		arms[a.Pathway.ID] = a
	}
	if got := arms[drilled.ID].Drilldowns; len(got) != 1 || got[0].ID != child.ID {
		t.Fatalf("drill-downs of the drilled arm: %+v", got)
	}
	if got := arms[plain.ID].Drilldowns; len(got) != 0 {
		t.Fatalf("drill-downs of the plain arm: %+v", got)
	}
	childView, err := rm.Bowtie(ctx, child.ID)
	seedNoErr(t, err)
	if childView.Risk.ParentPathwayID != drilled.ID || len(childView.Threats) != 0 {
		t.Fatalf("the child's own bow-tie: %+v", childView)
	}
}

func TestTheBowTieListsTheActionsOfTheRiskInEveryStatus(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	other := seedRisk(t, rm, ctx, "y")
	open := seedAction(t, rm, ctx, r.ID, "", "open one", "2999-01-01T00:00:00Z")
	done := seedAction(t, rm, ctx, r.ID, "", "done one", "2999-01-01T00:00:00Z")
	seedAction(t, rm, ctx, other.ID, "", "someone else's", "2999-01-01T00:00:00Z")
	_, err := rm.SetActionStatus(ctx, done.ID, riskmanager.ActionDone)
	seedNoErr(t, err)

	bt, err := rm.Bowtie(ctx, r.ID)
	seedNoErr(t, err)
	got := map[string]riskmanager.ActionStatus{}
	for _, a := range bt.Actions {
		got[a.ID] = a.Status
	}
	if len(got) != 2 || got[open.ID] != riskmanager.ActionOpen || got[done.ID] != riskmanager.ActionDone {
		t.Fatalf("actions on the bow-tie: %+v", bt.Actions)
	}
}

func TestAnEmptyBowTieEncodesEveryListAsAnEmptyArrayAndNeverNull(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	bt, err := rm.Bowtie(ctx, r.ID)
	seedNoErr(t, err)
	var doc map[string]any
	raw, err := json.Marshal(bt)
	seedNoErr(t, err)
	seedNoErr(t, json.Unmarshal(raw, &doc))
	for _, key := range []string{"threats", "consequences", "actions"} {
		list, ok := doc[key].([]any)
		if !ok || len(list) != 0 {
			t.Fatalf("%s encodes as %v in %s", key, doc[key], raw)
		}
	}

	seedThreat(t, rm, ctx, r.ID, "bare")
	bt, err = rm.Bowtie(ctx, r.ID)
	seedNoErr(t, err)
	raw, err = json.Marshal(bt)
	seedNoErr(t, err)
	seedNoErr(t, json.Unmarshal(raw, &doc))
	arm := doc["threats"].([]any)[0].(map[string]any)
	for _, key := range []string{"barriers", "drilldowns"} {
		list, ok := arm[key].([]any)
		if !ok || len(list) != 0 {
			t.Fatalf("an arm's %s encodes as %v in %s", key, arm[key], raw)
		}
	}
}

func TestTheBowTieOfAMissingRiskIsNotFound(t *testing.T) {
	rm, ctx := newManager(t)
	_, err := rm.Bowtie(ctx, "nowhere")
	seedWantErr(t, err, riskmanager.ErrRiskNotFound)
}

func TestWhatReliesOnAControlIsEveryPathwayAcrossEveryRiskItStandsOn(t *testing.T) {
	rm, ctx := newManager(t)
	r1 := seedRisk(t, rm, ctx, "first risk")
	r2 := seedRisk(t, rm, ctx, "second risk")
	r3 := seedRisk(t, rm, ctx, "third risk")
	egress := seedControl(t, rm, ctx, "egress", "o")
	other := seedControl(t, rm, ctx, "other", "o")
	p1 := seedThreat(t, rm, ctx, r1.ID, "p1")
	p2 := seedThreat(t, rm, ctx, r2.ID, "p2")
	p3 := seedConsequence(t, rm, ctx, r3.ID, "p3")
	for _, p := range []riskmanager.Pathway{p1, p2, p3} {
		seedBarrier(t, rm, ctx, p.ID, egress.ID)
	}
	seedBarrier(t, rm, ctx, p1.ID, other.ID)

	got, err := rm.Reliance(ctx, egress.ID)
	seedNoErr(t, err)
	if len(got) != 3 {
		t.Fatalf("%d places rely on the control, want 3: %+v", len(got), got)
	}
	titles := map[string]riskmanager.Side{}
	for _, m := range got {
		if m.Control.ID != egress.ID {
			t.Fatalf("another control's barrier came back: %+v", m.Control)
		}
		titles[m.RiskTitle+"/"+m.Pathway.Title] = m.Pathway.Side
	}
	want := map[string]riskmanager.Side{
		"first risk/p1": riskmanager.SideThreat, "second risk/p2": riskmanager.SideThreat, "third risk/p3": riskmanager.SideConsequence,
	}
	for k, side := range want {
		if titles[k] != side {
			t.Fatalf("reliance = %v, want %v", titles, want)
		}
	}
}

func TestWhatReliesOnAMissingControlIsNotFoundAndOnAnUnusedOneIsNothing(t *testing.T) {
	rm, ctx := newManager(t)
	_, err := rm.Reliance(ctx, "nowhere")
	seedWantErr(t, err, riskmanager.ErrControlNotFound)
	idle := seedControl(t, rm, ctx, "idle", "o")
	got, err := rm.Reliance(ctx, idle.ID)
	seedNoErr(t, err)
	if len(got) != 0 {
		t.Fatalf("an unused control is relied on in %d place(s)", len(got))
	}
}

func TestACommonCauseFailureIsVisibleFromTheOneControlThatFailsEverywhere(t *testing.T) {
	rm, ctx := newManager(t)
	r1 := seedRisk(t, rm, ctx, "first")
	r2 := seedRisk(t, rm, ctx, "second")
	shared := seedControl(t, rm, ctx, "shared", "o")
	b1 := seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r1.ID, "p1").ID, shared.ID)
	b2 := seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r2.ID, "p2").ID, shared.ID)
	seedEvidence(t, rm, ctx, b1.ID, riskmanager.OutcomePass, "2026-01-01T00:00:00Z")
	seedEvidence(t, rm, ctx, b2.ID, riskmanager.OutcomeFail, "2026-01-02T00:00:00Z")

	got, err := rm.Reliance(ctx, shared.ID)
	seedNoErr(t, err)
	standing := map[string]riskmanager.Standing{}
	for _, m := range got {
		standing[m.RiskTitle] = m.Standing
	}
	if standing["second"] != riskmanager.StandingFailing || standing["first"] == riskmanager.StandingFailing {
		t.Fatalf("standing by risk = %v: the failing placement must show on its own risk only", standing)
	}
}
