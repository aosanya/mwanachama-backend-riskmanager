package riskmanager_test

import (
	"fmt"
	"testing"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
)

func TestAMeasureQueryIsMatchedInSQLSoOnlyTheMatchesAreLoaded(t *testing.T) {
	rm, ctx, meter := newMeteredManager(t)
	const risks = 60
	for i := range risks {
		r := seedRisk(t, rm, ctx, fmt.Sprintf("risk %d", i))
		b := seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r.ID, fmt.Sprintf("route %d", i)).ID, seedControl(t, rm, ctx, fmt.Sprintf("guard %d", i), "o").ID)
		seedEvidence(t, rm, ctx, b.ID, riskmanager.OutcomePass, "2026-01-01T00:00:00Z")
	}
	meter.reset()
	got, err := rm.PreventiveMeasures(ctx, riskmanager.MeasureFilter{Query: "GUARD 42"})
	seedNoErr(t, err)
	if len(got) != 1 || got[0].Control.Name != "guard 42" {
		t.Fatalf("measures: %+v", got)
	}
	if rows := meter.rows.Load(); rows > 12 {
		t.Fatalf("matching one measure of %d loaded %d rows, want only the match and what it needs", risks, rows)
	}
}

func TestAMeasureQueryMatchesThePathwayOrTheControlAndTheLimitCutsTheRankedList(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "r")
	threat := seedThreat(t, rm, ctx, r.ID, "Prompt injection")
	other := seedThreat(t, rm, ctx, r.ID, "Other")
	for i := range 5 {
		seedBarrier(t, rm, ctx, threat.ID, seedControl(t, rm, ctx, fmt.Sprintf("filter %d", i), "o").ID)
	}
	seedBarrier(t, rm, ctx, other.ID, seedControl(t, rm, ctx, "Injection scanner", "o").ID)
	seedBarrier(t, rm, ctx, other.ID, seedControl(t, rm, ctx, "unrelated", "o").ID)

	all, err := rm.PreventiveMeasures(ctx, riskmanager.MeasureFilter{Query: "injection"})
	seedNoErr(t, err)
	if len(all) != 6 {
		t.Fatalf("%d measures match injection, want 6: the 5 on the matching threat and the scanner on the other", len(all))
	}
	page, err := rm.PreventiveMeasures(ctx, riskmanager.MeasureFilter{Query: "injection", Limit: 3})
	seedNoErr(t, err)
	if len(page) != 3 || page[0].Barrier.ID != all[0].Barrier.ID || page[2].Barrier.ID != all[2].Barrier.ID {
		t.Fatalf("a limit of 3 gave %d measures that are not the first 3 of the ranked list", len(page))
	}
}
