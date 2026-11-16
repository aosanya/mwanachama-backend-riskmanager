package riskmanager_test

import (
	"context"
	"testing"

	rm "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-riskmanager/routes"
)

func seedOwnedRisks(t *testing.T, m rm.RiskManager) {
	t.Helper()
	ctx := context.Background()
	for _, r := range []rm.Risk{
		{Title: "Flooded depot", TopEvent: "Depot floods", Owner: "owner-a"},
		{Title: "Flooded archive", TopEvent: "Archive floods", Owner: "owner-b"},
	} {
		created, err := m.CreateRisk(ctx, r)
		if err != nil {
			t.Fatalf("CreateRisk: %v", err)
		}
		if _, err := m.CreatePathway(ctx, rm.Pathway{RiskID: created.ID, Side: rm.SideThreat, Title: "Heavy rain"}); err != nil {
			t.Fatalf("CreatePathway: %v", err)
		}
	}
}

func TestGapsNarrowToOneOwnersRisks(t *testing.T) {
	m := newManager(t)
	seedOwnedRisks(t, m)
	ctx := context.Background()
	all, err := m.Gaps(ctx, rm.GapFilter{})
	if err != nil {
		t.Fatalf("Gaps: %v", err)
	}
	mine, err := m.Gaps(ctx, rm.GapFilter{Owner: "owner-a"})
	if err != nil {
		t.Fatalf("Gaps by owner: %v", err)
	}
	if len(mine) == 0 || len(mine) >= len(all) {
		t.Fatalf("owner gaps = %d of %d, want some but not all", len(mine), len(all))
	}
	risks, err := m.ListRisks(ctx, rm.RiskFilter{Owner: "owner-a"})
	if err != nil || len(risks) != 1 {
		t.Fatalf("ListRisks by owner = %v, %v", risks, err)
	}
	for _, g := range mine {
		if g.RiskID != risks[0].ID {
			t.Fatalf("gap %+v belongs to another owner's risk", g)
		}
	}
	none, err := m.Gaps(ctx, rm.GapFilter{Owner: "nobody"})
	if err != nil || len(none) != 0 {
		t.Fatalf("unknown owner gaps = %v, %v", none, err)
	}
}

func TestMatchOffersOptionsWhenANameIsShared(t *testing.T) {
	m := newManager(t)
	seedOwnedRisks(t, m)
	ctx := context.Background()
	if _, err := m.CreateControl(ctx, rm.Control{Name: "Flood barrier"}); err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	many, err := m.Match(ctx, rm.MatchFilter{Query: "flood"})
	if err != nil || !many.Ambiguous || len(many.Options) != 3 {
		t.Fatalf("Match flood = %+v, %v", many, err)
	}
	kinds := map[string]int{}
	for _, o := range many.Options {
		kinds[o.Kind]++
		if o.ID == "" || o.Label == "" {
			t.Fatalf("option %+v lacks key or label", o)
		}
	}
	if kinds["risk"] != 2 || kinds["control"] != 1 {
		t.Fatalf("kinds = %v", kinds)
	}
	one, err := m.Match(ctx, rm.MatchFilter{Query: "archive"})
	if err != nil || one.Ambiguous || len(one.Options) != 1 || one.Options[0].Label != "Flooded archive" {
		t.Fatalf("Match archive = %+v, %v", one, err)
	}
	empty, err := m.Match(ctx, rm.MatchFilter{Query: "  "})
	if err != nil || len(empty.Options) != 0 || empty.Ambiguous {
		t.Fatalf("Match blank = %+v, %v", empty, err)
	}
}

func TestEveryReadQuestionIsAnAgentTool(t *testing.T) {
	tools, err := routes.BuildTools(newManager(t), routes.Mount{})
	if err != nil {
		t.Fatalf("BuildTools: %v", err)
	}
	have := map[string]bool{}
	for _, tool := range tools {
		have[tool.Name] = true
	}
	for _, name := range []string{
		"riskmanager_threat_list",
		"riskmanager_consequence_list",
		"riskmanager_preventive_measure_list",
		"riskmanager_mitigating_measure_list",
		"riskmanager_control_reliance",
		"riskmanager_gap_list",
		"riskmanager_risk_list",
		"riskmanager_match_list",
	} {
		if !have[name] {
			t.Errorf("tool %s is not offered", name)
		}
	}
}
