package riskmanager_test

import (
	"testing"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
)

func TestABarrierStartsUnassessedAndProposedAndCopiesItsRiskFromItsPathway(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	p := seedThreat(t, rm, ctx, r.ID, "t")
	c := seedControl(t, rm, ctx, "c", "o")
	b, err := rm.PlaceBarrier(ctx, riskmanager.Barrier{PathwayID: p.ID, ControlID: c.ID, RiskID: "a-lie"})
	seedNoErr(t, err)
	held, err := rm.ListBarriers(ctx, riskmanager.BarrierFilter{PathwayID: p.ID})
	seedNoErr(t, err)
	for _, got := range []riskmanager.Barrier{b, held[0]} {
		if got.RiskID != r.ID {
			t.Fatalf("risk_id = %q, want the pathway's risk %q", got.RiskID, r.ID)
		}
		if got.Design != riskmanager.JudgementUnassessed || got.Coverage != riskmanager.JudgementUnassessed {
			t.Fatalf("judgements = %s/%s", got.Design, got.Coverage)
		}
		if got.Maturity != riskmanager.MaturityProposed {
			t.Fatalf("maturity = %s", got.Maturity)
		}
	}
}

func TestABarrierNamingAMissingPathwayOrControlIsAnInvalidReference(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	p := seedThreat(t, rm, ctx, r.ID, "t")
	c := seedControl(t, rm, ctx, "c", "o")
	_, err := rm.PlaceBarrier(ctx, riskmanager.Barrier{PathwayID: "nowhere", ControlID: c.ID})
	seedWantErr(t, err, riskmanager.ErrInvalidReference)
	_, err = rm.PlaceBarrier(ctx, riskmanager.Barrier{PathwayID: p.ID, ControlID: "nowhere"})
	seedWantErr(t, err, riskmanager.ErrInvalidReference)
	_, err = rm.PlaceBarrier(ctx, riskmanager.Barrier{ControlID: c.ID})
	seedWantErr(t, err, riskmanager.ErrInvalid)
}

func TestOneControlStandingTwiceOnOnePathwayIsAConflict(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	p := seedThreat(t, rm, ctx, r.ID, "t")
	c := seedControl(t, rm, ctx, "c", "o")
	seedBarrier(t, rm, ctx, p.ID, c.ID)
	_, err := rm.PlaceBarrier(ctx, riskmanager.Barrier{PathwayID: p.ID, ControlID: c.ID})
	seedWantErr(t, err, riskmanager.ErrConflict)
	held, _ := rm.ListBarriers(ctx, riskmanager.BarrierFilter{PathwayID: p.ID})
	if len(held) != 1 {
		t.Fatalf("%d barriers stand on the pathway", len(held))
	}
}

func TestOneControlMayStandOnManyPathways(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	p1 := seedThreat(t, rm, ctx, r.ID, "t1")
	p2 := seedThreat(t, rm, ctx, r.ID, "t2")
	c := seedControl(t, rm, ctx, "c", "o")
	seedBarrier(t, rm, ctx, p1.ID, c.ID)
	seedBarrier(t, rm, ctx, p2.ID, c.ID)
	held, err := rm.ListBarriers(ctx, riskmanager.BarrierFilter{ControlID: c.ID})
	seedNoErr(t, err)
	if len(held) != 2 {
		t.Fatalf("the control stands on %d pathways, want 2", len(held))
	}
}

func TestAControlCannotClaimToBeTestedOrMonitoredWithNothingOnRecord(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	p := seedThreat(t, rm, ctx, r.ID, "t")
	c := seedControl(t, rm, ctx, "c", "o")
	for _, m := range []riskmanager.Maturity{riskmanager.MaturityTested, riskmanager.MaturityMonitored} {
		_, err := rm.PlaceBarrier(ctx, riskmanager.Barrier{PathwayID: p.ID, ControlID: c.ID, Maturity: m})
		seedWantErr(t, err, riskmanager.ErrInvalid)
	}
	if held, _ := rm.ListBarriers(ctx, riskmanager.BarrierFilter{PathwayID: p.ID}); len(held) != 0 {
		t.Fatalf("a refused placement left %d barrier(s)", len(held))
	}

	b := seedBarrier(t, rm, ctx, p.ID, c.ID)
	for _, m := range []riskmanager.Maturity{riskmanager.MaturityTested, riskmanager.MaturityMonitored} {
		_, err := rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: b.ID, Maturity: &m})
		seedWantErr(t, err, riskmanager.ErrInvalid)
	}
}

func TestEvenAFailingPieceOfEvidenceLetsABarrierReachTested(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	p := seedThreat(t, rm, ctx, r.ID, "t")
	b := seedBarrier(t, rm, ctx, p.ID, seedControl(t, rm, ctx, "c", "o").ID)
	seedEvidence(t, rm, ctx, b.ID, riskmanager.OutcomeFail, "2026-01-01T00:00:00Z")
	for _, m := range []riskmanager.Maturity{riskmanager.MaturityTested, riskmanager.MaturityMonitored} {
		got, err := rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: b.ID, Maturity: &m})
		seedNoErr(t, err)
		if got.Maturity != m {
			t.Fatalf("maturity = %s, want %s", got.Maturity, m)
		}
	}
}

func TestMaturityMayFallBackAsWellAsRise(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	p := seedThreat(t, rm, ctx, r.ID, "t")
	b := seedSoundBarrier(t, rm, ctx, p.ID, seedControl(t, rm, ctx, "c", "o").ID)
	got, err := rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: b.ID, Maturity: seedPtr(riskmanager.MaturityDeployed)})
	seedNoErr(t, err)
	if got.Maturity != riskmanager.MaturityDeployed {
		t.Fatalf("maturity = %s", got.Maturity)
	}
}

func TestUpdatingABarrierLeavesWhatTheChangeDoesNotNameAlone(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	p := seedThreat(t, rm, ctx, r.ID, "t")
	b := seedBarrier(t, rm, ctx, p.ID, seedControl(t, rm, ctx, "c", "o").ID)

	_, err := rm.UpdateBarrier(ctx, riskmanager.BarrierChange{
		ID: b.ID, Position: seedPtr(5), Design: seedPtr(riskmanager.JudgementPartial), Notes: seedPtr("thin"),
	})
	seedNoErr(t, err)
	got, err := rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: b.ID, Coverage: seedPtr(riskmanager.JudgementInadequate)})
	seedNoErr(t, err)
	if got.Position != 5 || got.Design != riskmanager.JudgementPartial || got.Notes != "thin" {
		t.Fatalf("a change naming only coverage disturbed the rest: %+v", got)
	}
	if got.Coverage != riskmanager.JudgementInadequate {
		t.Fatalf("coverage = %s", got.Coverage)
	}
	cleared, err := rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: b.ID, Notes: seedPtr(""), Position: seedPtr(0)})
	seedNoErr(t, err)
	if cleared.Notes != "" || cleared.Position != 0 {
		t.Fatalf("an explicit empty value could not be set: %+v", cleared)
	}
}

func TestABarrierCannotBeGivenAJudgementOutsideTheDeclaredSet(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	p := seedThreat(t, rm, ctx, r.ID, "t")
	b := seedBarrier(t, rm, ctx, p.ID, seedControl(t, rm, ctx, "c", "o").ID)
	bogus := riskmanager.Judgement("great")
	_, err := rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: b.ID, Design: &bogus})
	seedWantErr(t, err, riskmanager.ErrInvalid)
	_, err = rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: b.ID, Coverage: &bogus})
	seedWantErr(t, err, riskmanager.ErrInvalid)
	badMaturity := riskmanager.Maturity("perfect")
	_, err = rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: b.ID, Maturity: &badMaturity})
	seedWantErr(t, err, riskmanager.ErrInvalid)
}

func TestAMissingBarrierIsNotFound(t *testing.T) {
	rm, ctx := newManager(t)
	_, err := rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: "nobody", Notes: seedPtr("x")})
	seedWantErr(t, err, riskmanager.ErrBarrierNotFound)
	seedWantErr(t, rm.RemoveBarrier(ctx, "nobody"), riskmanager.ErrBarrierNotFound)
}

func TestABarrierWithEvidenceCannotBeRemoved(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	p := seedThreat(t, rm, ctx, r.ID, "t")
	b := seedBarrier(t, rm, ctx, p.ID, seedControl(t, rm, ctx, "c", "o").ID)
	seedEvidence(t, rm, ctx, b.ID, riskmanager.OutcomePass, "2026-01-01T00:00:00Z")
	seedWantErr(t, rm.RemoveBarrier(ctx, b.ID), riskmanager.ErrConflict)
	if held, _ := rm.ListBarriers(ctx, riskmanager.BarrierFilter{PathwayID: p.ID}); len(held) != 1 {
		t.Fatalf("a refused removal left %d barrier(s)", len(held))
	}
}

func TestABarrierAnActionIsAboutCannotBeRemoved(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	p := seedThreat(t, rm, ctx, r.ID, "t")
	b := seedBarrier(t, rm, ctx, p.ID, seedControl(t, rm, ctx, "c", "o").ID)
	seedAction(t, rm, ctx, r.ID, b.ID, "strengthen it", "2999-01-01T00:00:00Z")
	seedWantErr(t, rm.RemoveBarrier(ctx, b.ID), riskmanager.ErrConflict)
}

func TestABarrierWithNothingOnRecordCanBeRemovedAndPlacedAgain(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	p := seedThreat(t, rm, ctx, r.ID, "t")
	c := seedControl(t, rm, ctx, "c", "o")
	b := seedBarrier(t, rm, ctx, p.ID, c.ID)
	seedNoErr(t, rm.RemoveBarrier(ctx, b.ID))
	if held, _ := rm.ListBarriers(ctx, riskmanager.BarrierFilter{PathwayID: p.ID}); len(held) != 0 {
		t.Fatalf("%d barrier(s) remain", len(held))
	}
	seedBarrier(t, rm, ctx, p.ID, c.ID)
}

func TestBarriersAreListedByPositionThenByWhenTheyWerePlaced(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	p := seedThreat(t, rm, ctx, r.ID, "t")
	names := []string{"first", "second", "third", "fourth"}
	var placed []riskmanager.Barrier
	for _, n := range names {
		placed = append(placed, seedBarrier(t, rm, ctx, p.ID, seedControl(t, rm, ctx, n, "o").ID))
	}
	for i, pos := range []int{3, 1, 3, 2} {
		_, err := rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: placed[i].ID, Position: seedPtr(pos)})
		seedNoErr(t, err)
	}
	got, err := rm.ListBarriers(ctx, riskmanager.BarrierFilter{PathwayID: p.ID})
	seedNoErr(t, err)
	want := []string{placed[1].ID, placed[3].ID, placed[0].ID, placed[2].ID}
	for i, b := range got {
		if b.ID != want[i] {
			t.Fatalf("barrier %d is %s, want %s (positions 1,2,3,3 with ties in placement order)", i, b.ID, want[i])
		}
	}
}

func TestBarriersFilterByRiskPathwayAndControl(t *testing.T) {
	rm, ctx := newManager(t)
	r1 := seedRisk(t, rm, ctx, "one")
	r2 := seedRisk(t, rm, ctx, "two")
	p1 := seedThreat(t, rm, ctx, r1.ID, "t1")
	p2 := seedThreat(t, rm, ctx, r2.ID, "t2")
	shared := seedControl(t, rm, ctx, "shared", "o")
	only := seedControl(t, rm, ctx, "only", "o")
	seedBarrier(t, rm, ctx, p1.ID, shared.ID)
	seedBarrier(t, rm, ctx, p1.ID, only.ID)
	seedBarrier(t, rm, ctx, p2.ID, shared.ID)

	count := func(f riskmanager.BarrierFilter) int {
		got, err := rm.ListBarriers(ctx, f)
		seedNoErr(t, err)
		return len(got)
	}
	if n := count(riskmanager.BarrierFilter{}); n != 3 {
		t.Fatalf("all: %d", n)
	}
	if n := count(riskmanager.BarrierFilter{RiskID: r1.ID}); n != 2 {
		t.Fatalf("risk one: %d", n)
	}
	if n := count(riskmanager.BarrierFilter{PathwayID: p2.ID}); n != 1 {
		t.Fatalf("pathway two: %d", n)
	}
	if n := count(riskmanager.BarrierFilter{ControlID: shared.ID}); n != 2 {
		t.Fatalf("shared control: %d", n)
	}
	if n := count(riskmanager.BarrierFilter{RiskID: r2.ID, ControlID: only.ID}); n != 0 {
		t.Fatalf("a control that stands nowhere on risk two: %d", n)
	}
}
