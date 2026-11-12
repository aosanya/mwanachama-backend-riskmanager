package routes_test

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"testing"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-riskmanager/routes"
)

func actionsOf(rts []routes.Route) []string {
	out := make([]string, 0, len(rts))
	for _, rt := range rts {
		out = append(out, rt.Action)
	}
	sort.Strings(out)
	return out
}

func TestRoutesAndRoutesForBuildTheSameThirtyTwoActions(t *testing.T) {
	rm, _ := newManager(t)
	plain := actionsOf(routes.Routes(rm))
	mounted := actionsOf(routes.RoutesFor(rm, routes.Mount{Caller: func(context.Context) string { return "actor-1" }}))
	if len(plain) != 32 || strings.Join(plain, ",") != strings.Join(mounted, ",") {
		t.Fatalf("plain %d actions, mounted %d, want the same 32", len(plain), len(mounted))
	}
}

func TestSplitLeavesNothingAnonymousAndGatesAllAddresses(t *testing.T) {
	rm, _ := newManager(t)
	split := routes.Split(rm, routes.Mount{Authorize: func(context.Context, string) error { return nil }})
	if len(split.Anonymous) != 0 || len(split.Gated) != 32 {
		t.Fatalf("anonymous %d, gated %d, want 0 and 32", len(split.Anonymous), len(split.Gated))
	}
}

func TestEveryRouteHasAToolUnderTheSameAction(t *testing.T) {
	rm, _ := newManager(t)
	tools, err := routes.BuildTools(rm, routes.Mount{})
	if err != nil {
		t.Fatal(err)
	}
	toolActions := map[string]bool{}
	for _, tool := range tools {
		if tool.Name == "" || tool.Action == "" {
			t.Errorf("tool %+v has no name or action", tool)
		}
		toolActions[tool.Action] = true
	}
	for _, action := range actionsOf(routes.Routes(rm)) {
		if !toolActions[action] {
			t.Errorf("route action %s has no tool", action)
		}
	}
}

func TestARefusedToolCallStoresNothing(t *testing.T) {
	rm, ctx := newManager(t)
	deny := func(context.Context, string) error { return errors.New("no") }
	tools, err := routes.BuildTools(rm, routes.Mount{Authorize: deny})
	if err != nil {
		t.Fatal(err)
	}
	var create routes.Tool
	for _, tool := range tools {
		if strings.HasSuffix(tool.Action, ".risk.create") {
			create = tool
		}
	}
	if create.Invoke == nil {
		t.Fatal("no risk create tool")
	}
	if _, err := create.Invoke(ctx, json.RawMessage(`{"title":"Refused","top_event":"never stored"}`)); err == nil {
		t.Fatal("a denied tool call succeeded")
	}
	held, err := rm.ListRisks(ctx, riskmanager.RiskFilter{})
	if err != nil || len(held) != 0 {
		t.Fatalf("a refused tool stored %d risk(s), err %v", len(held), err)
	}
}

func TestAPermittedToolCallCreatesADraftRisk(t *testing.T) {
	rm, ctx := newManager(t)
	allow := func(context.Context, string) error { return nil }
	tools, err := routes.BuildTools(rm, routes.Mount{Authorize: allow})
	if err != nil {
		t.Fatal(err)
	}
	var create routes.Tool
	for _, tool := range tools {
		if strings.HasSuffix(tool.Action, ".risk.create") {
			create = tool
		}
	}
	if _, err := create.Invoke(ctx, json.RawMessage(`{"title":"Leak","top_event":"data leaves","status":"open"}`)); err != nil {
		t.Fatal(err)
	}
	held, err := rm.ListRisks(ctx, riskmanager.RiskFilter{})
	if err != nil || len(held) != 1 || held[0].Title != "Leak" || held[0].Status != riskmanager.StatusDraft {
		t.Fatalf("held %+v err %v, want one draft risk titled Leak", held, err)
	}
}
