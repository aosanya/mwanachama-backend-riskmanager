package routes_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-riskmanager/routes"
)

func gapsAt(t *testing.T, target string, mux *http.ServeMux) []riskmanager.Gap {
	t.Helper()
	w := do(t, mux, "GET", target, "")
	if w.Code != http.StatusOK {
		t.Fatalf("%s answered %d: %s", target, w.Code, w.Body)
	}
	var got []riskmanager.Gap
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode %s: %v", w.Body, err)
	}
	return got
}

func TestTheGapQueryBindsSeverityLimitAndOffset(t *testing.T) {
	mux, rm, ctx := newMux(t)
	r := seedRisk(t, rm, ctx, "bare")
	for _, title := range []string{"a", "b", "c"} {
		seedThreat(t, rm, ctx, r.ID, title)
	}

	everyHigh := gapsAt(t, "/gaps?risk_id="+r.ID+"&severity=high", mux)
	if len(everyHigh) != 3 {
		t.Fatalf("high gaps of a risk with three bare threats: %d, want 3", len(everyHigh))
	}
	for _, g := range everyHigh {
		if g.Severity != riskmanager.Severity("high") || g.Kind != riskmanager.GapKind("pathway_unguarded") {
			t.Fatalf("the severity filter did not bind: %+v", g)
		}
	}

	paged := gapsAt(t, "/gaps?risk_id="+r.ID+"&severity=high&limit=2&offset=1", mux)
	if len(paged) != 2 || paged[0].SubjectID != everyHigh[1].SubjectID || paged[1].SubjectID != everyHigh[2].SubjectID {
		t.Fatalf("limit and offset did not bind: %+v from %+v", paged, everyHigh)
	}

	all := gapsAt(t, "/gaps?risk_id="+r.ID, mux)
	if first := all[0]; first.Severity != riskmanager.Severity("high") {
		t.Fatalf("the answer does not open with its high gaps: %+v", first)
	}
	if past := gapsAt(t, "/gaps?risk_id="+r.ID+"&offset=500", mux); past == nil || len(past) != 0 {
		t.Fatalf("an offset past the end answered %+v", past)
	}
}

func TestAnUnknownSeverityOrKindOnTheGapQueryAnswers400(t *testing.T) {
	mux, _, _ := newMux(t)
	for _, target := range []string{"/gaps?severity=urgent", "/gaps?kind=nonsense"} {
		if w := do(t, mux, "GET", target, ""); w.Code != http.StatusBadRequest {
			t.Fatalf("%s answered %d, want 400: %s", target, w.Code, w.Body)
		}
	}
}

func TestTheGapToolOffersTheSameFiltersAsTheRoute(t *testing.T) {
	rm, _ := newManager(t)
	tools, err := routes.BuildTools(rm, routes.Mount{})
	if err != nil {
		t.Fatalf("build tools: %v", err)
	}
	for _, tool := range tools {
		if tool.Name != "riskmanager_gap_list" {
			continue
		}
		schema := string(tool.InputSchema)
		for _, property := range []string{`"risk_id"`, `"kind"`, `"severity"`, `"limit"`, `"offset"`} {
			if !strings.Contains(schema, property) {
				t.Fatalf("the gap tool does not offer %s: %s", property, schema)
			}
		}
		return
	}
	t.Fatal("there is no riskmanager_gap_list tool")
}
