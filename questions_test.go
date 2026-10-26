package riskmanager_test

import (
	"context"
	"encoding/json"
	"testing"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
)

func seedThreatsOf(t *testing.T, rm riskmanager.RiskManager, ctx context.Context, f riskmanager.PathwayFilter) map[string]riskmanager.PathwaySummary {
	t.Helper()
	got, err := rm.Threats(ctx, f)
	seedNoErr(t, err)
	out := map[string]riskmanager.PathwaySummary{}
	for _, s := range got {
		out[s.Title] = s
	}
	return out
}

func seedMeasuresBy(measures []riskmanager.Measure) map[string]riskmanager.Measure {
	out := map[string]riskmanager.Measure{}
	for _, m := range measures {
		out[m.Control.Name] = m
	}
	return out
}

func TestTheThreatsOfARiskAreItsThreatPathwaysAndTheConsequencesItsConsequencePathways(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	other := seedRisk(t, rm, ctx, "y")
	seedThreat(t, rm, ctx, r.ID, "t1")
	seedThreat(t, rm, ctx, r.ID, "t2")
	seedConsequence(t, rm, ctx, r.ID, "c1")
	seedThreat(t, rm, ctx, other.ID, "elsewhere")

	threats, err := rm.Threats(ctx, riskmanager.PathwayFilter{RiskID: r.ID})
	seedNoErr(t, err)
	if len(threats) != 2 || threats[0].Title != "t1" || threats[1].Title != "t2" {
		t.Fatalf("threats of the risk: %+v", threats)
	}
	for _, s := range threats {
		if s.Side != riskmanager.SideThreat || s.RiskID != r.ID || s.RiskTitle != "x" {
			t.Fatalf("a summary that is not a threat of this risk: %+v", s)
		}
	}
	consequences, err := rm.Consequences(ctx, riskmanager.PathwayFilter{RiskID: r.ID})
	seedNoErr(t, err)
	if len(consequences) != 1 || consequences[0].Title != "c1" || consequences[0].Side != riskmanager.SideConsequence {
		t.Fatalf("consequences of the risk: %+v", consequences)
	}
}

func TestAskingAboutAMissingRiskIsNotFoundAndNotAnEmptyAnswer(t *testing.T) {
	rm, ctx := newManager(t)
	_, err := rm.Threats(ctx, riskmanager.PathwayFilter{RiskID: "nowhere"})
	seedWantErr(t, err, riskmanager.ErrRiskNotFound)
	_, err = rm.Consequences(ctx, riskmanager.PathwayFilter{RiskID: "nowhere"})
	seedWantErr(t, err, riskmanager.ErrRiskNotFound)
	_, err = rm.PreventiveMeasures(ctx, riskmanager.MeasureFilter{RiskID: "nowhere"})
	seedWantErr(t, err, riskmanager.ErrRiskNotFound)
	_, err = rm.MitigatingMeasures(ctx, riskmanager.MeasureFilter{RiskID: "nowhere"})
	seedWantErr(t, err, riskmanager.ErrRiskNotFound)
}

func TestAskingForThreatsWithNoRiskReadsEveryRisksThreatsAndHonoursTheLimit(t *testing.T) {
	rm, ctx := newManager(t)
	r1 := seedRisk(t, rm, ctx, "one")
	r2 := seedRisk(t, rm, ctx, "two")
	seedThreat(t, rm, ctx, r1.ID, "a")
	seedThreat(t, rm, ctx, r2.ID, "b")
	seedConsequence(t, rm, ctx, r2.ID, "not a threat")

	if all := seedThreatsOf(t, rm, ctx, riskmanager.PathwayFilter{}); len(all) != 2 {
		t.Fatalf("every threat: %+v", all)
	}
	if one := seedThreatsOf(t, rm, ctx, riskmanager.PathwayFilter{Limit: 1}); len(one) != 1 {
		t.Fatalf("limit 1 gave %d", len(one))
	}
}

func TestThreatsAreFoundByWordsInTheirTitleOrScenarioInAnyCase(t *testing.T) {
	rm, ctx := newManager(t)
	r1 := seedRisk(t, rm, ctx, "one")
	r2 := seedRisk(t, rm, ctx, "two")
	_, err := rm.CreatePathway(ctx, riskmanager.Pathway{RiskID: r1.ID, Side: riskmanager.SideThreat, Title: "Retrieval of restricted files", Scenario: "an assistant reads a file"})
	seedNoErr(t, err)
	_, err = rm.CreatePathway(ctx, riskmanager.Pathway{RiskID: r2.ID, Side: riskmanager.SideThreat, Title: "Pasted records", Scenario: "A person pastes RETRIEVAL results"})
	seedNoErr(t, err)
	seedThreat(t, rm, ctx, r2.ID, "Unrelated")
	seedConsequence(t, rm, ctx, r2.ID, "retrieval is also a word here")

	if got := seedThreatsOf(t, rm, ctx, riskmanager.PathwayFilter{Query: "rEtRiEvAl"}); len(got) != 2 {
		t.Fatalf("a word in one title and in another scenario: %+v", got)
	}
	got := seedThreatsOf(t, rm, ctx, riskmanager.PathwayFilter{Query: "retrieval", RiskID: r2.ID})
	if _, ok := got["Pasted records"]; !ok || len(got) != 1 {
		t.Fatalf("the word narrowed to one risk: %+v", got)
	}
	if none := seedThreatsOf(t, rm, ctx, riskmanager.PathwayFilter{Query: "no such words"}); len(none) != 0 {
		t.Fatalf("nothing should match: %+v", none)
	}
}

func TestPercentAndUnderscoreInAThreatSearchMatchLiterally(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	seedThreat(t, rm, ctx, r.ID, "50% of the records")
	seedThreat(t, rm, ctx, r.ID, "audit_log exposure")
	seedThreat(t, rm, ctx, r.ID, "plain")
	for query, want := range map[string]string{"%": "50% of the records", "_": "audit_log exposure", "t_l": "audit_log exposure"} {
		got := seedThreatsOf(t, rm, ctx, riskmanager.PathwayFilter{Query: query})
		if _, ok := got[want]; !ok || len(got) != 1 {
			t.Fatalf("query %q matched %+v, want only %q", query, got, want)
		}
	}
}

func TestASummaryCountsTheBarriersOnItAndTheRisksThatDrillIntoIt(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	guarded := seedThreat(t, rm, ctx, r.ID, "guarded")
	seedThreat(t, rm, ctx, r.ID, "bare")
	seedBarrier(t, rm, ctx, guarded.ID, seedControl(t, rm, ctx, "c1", "o").ID)
	seedBarrier(t, rm, ctx, guarded.ID, seedControl(t, rm, ctx, "c2", "o").ID)
	for _, title := range []string{"child one", "child two"} {
		_, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: title, TopEvent: "e", ParentPathwayID: guarded.ID})
		seedNoErr(t, err)
	}

	byTitle := seedThreatsOf(t, rm, ctx, riskmanager.PathwayFilter{RiskID: r.ID})
	if g := byTitle["guarded"]; g.BarrierCount != 2 || g.Drilldowns != 2 {
		t.Fatalf("guarded: barriers=%d drilldowns=%d, want 2 and 2", g.BarrierCount, g.Drilldowns)
	}
	if b := byTitle["bare"]; b.BarrierCount != 0 || b.Drilldowns != 0 {
		t.Fatalf("bare: barriers=%d drilldowns=%d, want 0 and 0", b.BarrierCount, b.Drilldowns)
	}
}

func TestPreventiveMeasuresAreOnlyOnThreatsAndMitigatingOnlyOnConsequences(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	threat := seedThreat(t, rm, ctx, r.ID, "t")
	consequence := seedConsequence(t, rm, ctx, r.ID, "c")
	gate := seedControl(t, rm, ctx, "gate", "o")
	cleanup := seedControl(t, rm, ctx, "cleanup", "o")
	both := seedControl(t, rm, ctx, "both", "o")
	seedBarrier(t, rm, ctx, threat.ID, gate.ID)
	seedBarrier(t, rm, ctx, consequence.ID, cleanup.ID)
	seedBarrier(t, rm, ctx, threat.ID, both.ID)
	seedBarrier(t, rm, ctx, consequence.ID, both.ID)

	prevent, err := rm.PreventiveMeasures(ctx, riskmanager.MeasureFilter{RiskID: r.ID})
	seedNoErr(t, err)
	mitigate, err := rm.MitigatingMeasures(ctx, riskmanager.MeasureFilter{RiskID: r.ID})
	seedNoErr(t, err)
	pm, mm := seedMeasuresBy(prevent), seedMeasuresBy(mitigate)
	if len(prevent) != 2 || pm["gate"].Pathway.ID != threat.ID || pm["both"].Pathway.ID != threat.ID {
		t.Fatalf("preventive measures: %+v", prevent)
	}
	if len(mitigate) != 2 || mm["cleanup"].Pathway.ID != consequence.ID || mm["both"].Pathway.ID != consequence.ID {
		t.Fatalf("mitigating measures: %+v", mitigate)
	}
	for _, m := range prevent {
		if m.Pathway.Side != riskmanager.SideThreat {
			t.Fatalf("a preventive measure on a %s", m.Pathway.Side)
		}
	}
	for _, m := range mitigate {
		if m.Pathway.Side != riskmanager.SideConsequence {
			t.Fatalf("a mitigating measure on a %s", m.Pathway.Side)
		}
	}
}

func TestAMeasureCarriesTheRiskThePathwayTheBarrierAndTheControl(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "the risk")
	p := seedThreat(t, rm, ctx, r.ID, "the threat")
	c := seedControl(t, rm, ctx, "the control", "o")
	b := seedBarrier(t, rm, ctx, p.ID, c.ID)
	e := seedEvidence(t, rm, ctx, b.ID, riskmanager.OutcomePass, "2026-01-01T00:00:00Z")

	got, err := rm.PreventiveMeasures(ctx, riskmanager.MeasureFilter{PathwayID: p.ID})
	seedNoErr(t, err)
	if len(got) != 1 {
		t.Fatalf("%d measures", len(got))
	}
	m := got[0]
	if m.RiskID != r.ID || m.RiskTitle != "the risk" || m.Pathway.ID != p.ID || m.Barrier.ID != b.ID || m.Control.ID != c.ID {
		t.Fatalf("the measure is missing its context: %+v", m)
	}
	if m.Latest == nil || m.Latest.ID != e.ID || m.EvidenceCount != 1 {
		t.Fatalf("latest=%+v count=%d", m.Latest, m.EvidenceCount)
	}
}

func TestAskingForMeasuresAgainstAPathwayOfTheOtherSideIsRefused(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	threat := seedThreat(t, rm, ctx, r.ID, "t")
	consequence := seedConsequence(t, rm, ctx, r.ID, "c")
	_, err := rm.PreventiveMeasures(ctx, riskmanager.MeasureFilter{PathwayID: consequence.ID})
	seedWantErr(t, err, riskmanager.ErrInvalid)
	_, err = rm.MitigatingMeasures(ctx, riskmanager.MeasureFilter{PathwayID: threat.ID})
	seedWantErr(t, err, riskmanager.ErrInvalid)
	_, err = rm.PreventiveMeasures(ctx, riskmanager.MeasureFilter{PathwayID: "nowhere"})
	seedWantErr(t, err, riskmanager.ErrPathwayNotFound)
}

func TestAPathwayThatBelongsToAnotherRiskAnswersNothing(t *testing.T) {
	rm, ctx := newManager(t)
	r1 := seedRisk(t, rm, ctx, "one")
	r2 := seedRisk(t, rm, ctx, "two")
	p := seedThreat(t, rm, ctx, r1.ID, "t")
	seedBarrier(t, rm, ctx, p.ID, seedControl(t, rm, ctx, "c", "o").ID)
	got, err := rm.PreventiveMeasures(ctx, riskmanager.MeasureFilter{RiskID: r2.ID, PathwayID: p.ID})
	seedNoErr(t, err)
	if len(got) != 0 {
		t.Fatalf("a pathway of one risk answered for another: %+v", got)
	}
}

func TestMeasuresAreFoundByWordsInTheControlOrThePathwayInAnyCase(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	paste := seedThreat(t, rm, ctx, r.ID, "Pasted records")
	other := seedThreat(t, rm, ctx, r.ID, "Something else")
	seedBarrier(t, rm, ctx, paste.ID, seedControl(t, rm, ctx, "Input DLP", "o").ID)
	seedBarrier(t, rm, ctx, paste.ID, seedControl(t, rm, ctx, "Approved gateway", "o").ID)
	seedBarrier(t, rm, ctx, other.ID, seedControl(t, rm, ctx, "Output dlp", "o").ID)

	count := func(f riskmanager.MeasureFilter) int {
		got, err := rm.PreventiveMeasures(ctx, f)
		seedNoErr(t, err)
		return len(got)
	}
	if n := count(riskmanager.MeasureFilter{Query: "DLP"}); n != 2 {
		t.Fatalf("a control name in any case: %d", n)
	}
	if n := count(riskmanager.MeasureFilter{Query: "pasted"}); n != 2 {
		t.Fatalf("a pathway title brings every measure on it: %d", n)
	}
	if n := count(riskmanager.MeasureFilter{Query: "dlp", RiskID: r.ID, PathwayID: paste.ID}); n != 1 {
		t.Fatalf("a word within one pathway: %d", n)
	}
	if n := count(riskmanager.MeasureFilter{Limit: 1}); n != 1 {
		t.Fatalf("limit 1: %d", n)
	}
}

func TestNothingFoundIsAnEmptyListAndNeverNull(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	c := seedControl(t, rm, ctx, "unused", "o")
	threats, err := rm.Threats(ctx, riskmanager.PathwayFilter{RiskID: r.ID})
	seedNoErr(t, err)
	prevent, err := rm.PreventiveMeasures(ctx, riskmanager.MeasureFilter{RiskID: r.ID})
	seedNoErr(t, err)
	mitigate, err := rm.MitigatingMeasures(ctx, riskmanager.MeasureFilter{})
	seedNoErr(t, err)
	reliance, err := rm.Reliance(ctx, c.ID)
	seedNoErr(t, err)
	for name, v := range map[string]any{"threats": threats, "preventive": prevent, "mitigating": mitigate, "reliance": reliance} {
		raw, err := json.Marshal(v)
		seedNoErr(t, err)
		if string(raw) != "[]" {
			t.Fatalf("%s encodes as %s, want []", name, raw)
		}
	}
}
