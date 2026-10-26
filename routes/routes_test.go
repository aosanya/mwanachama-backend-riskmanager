package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-riskmanager/routes"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func newManager(t *testing.T) (riskmanager.RiskManager, context.Context) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	s, err := riskmanager.LoadSpec(filepath.Join("..", "spec", "examples", "ai_safety.riskmanager.json"))
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	rm, err := riskmanager.NewRiskManager(db, s)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	return rm, context.Background()
}

func muxFor(t *testing.T, rm riskmanager.RiskManager, m routes.Mount) *http.ServeMux {
	t.Helper()
	built, err := routes.BuildFor(rm, m)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	mux := http.NewServeMux()
	for _, rt := range built {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	return mux
}

func newMux(t *testing.T) (*http.ServeMux, riskmanager.RiskManager, context.Context) {
	t.Helper()
	rm, ctx := newManager(t)
	return muxFor(t, rm, routes.Mount{}), rm, ctx
}

func do(t *testing.T, mux *http.ServeMux, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func seedRisk(t *testing.T, rm riskmanager.RiskManager, ctx context.Context, title string) riskmanager.Risk {
	t.Helper()
	r, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: title, TopEvent: title + " happens"})
	if err != nil {
		t.Fatalf("seed risk %s: %v", title, err)
	}
	return r
}

func seedThreat(t *testing.T, rm riskmanager.RiskManager, ctx context.Context, riskID, title string) riskmanager.Pathway {
	t.Helper()
	p, err := rm.CreatePathway(ctx, riskmanager.Pathway{RiskID: riskID, Side: riskmanager.SideThreat, Title: title})
	if err != nil {
		t.Fatalf("seed threat %s: %v", title, err)
	}
	return p
}

func TestTheDeclaredTableBuilds(t *testing.T) {
	rm, _ := newManager(t)
	built, err := routes.Build(rm)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(built) != 32 {
		t.Fatalf("built %d routes, want the 32 the operations file declares", len(built))
	}
	for _, rt := range built {
		if rt.Action == "" {
			t.Errorf("%s %s carries no action, so nothing can gate it", rt.Method, rt.Path)
		}
	}
}

func TestEveryRouteIsGatedUntilNamedAnonymous(t *testing.T) {
	rm, _ := newManager(t)
	if len(routes.AnonymousActions) != 0 {
		t.Fatalf("AnonymousActions is %v; this module publishes nothing anonymously", routes.AnonymousActions)
	}
	if public := routes.PublicRoutes(rm); len(public) != 0 {
		t.Fatalf("%d routes are reachable without a caller", len(public))
	}
	if gated := routes.OperatorRoutes(rm, routes.Mount{}); len(gated) != 32 {
		t.Fatalf("%d routes are gated, want all 32", len(gated))
	}
}

func TestARefusedActionNeverReachesTheManager(t *testing.T) {
	rm, ctx := newManager(t)
	deny := func(ctx context.Context, action string) error { return context.Canceled }
	mux := muxFor(t, rm, routes.Mount{Authorize: deny})

	w := do(t, mux, "POST", "/risks", `{"title":"Refused","top_event":"never stored"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("code = %d, want 403", w.Code)
	}
	held, err := rm.ListRisks(ctx, riskmanager.RiskFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(held) != 0 {
		t.Fatalf("a refused action stored %d risk(s)", len(held))
	}
}

func TestTheAddressOutranksTheBody(t *testing.T) {
	mux, rm, ctx := newMux(t)
	kept := seedRisk(t, rm, ctx, "Kept")
	other := seedRisk(t, rm, ctx, "Other")

	w := do(t, mux, "PUT", "/risks/"+kept.ID, `{"id":"`+other.ID+`","title":"renamed by the body","top_event":"changed"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d: %s", w.Code, w.Body)
	}
	var got riskmanager.Risk
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != kept.ID {
		t.Fatalf("the body's id won: updated %s, address named %s", got.ID, kept.ID)
	}

	untouched, err := rm.GetRisk(ctx, other.ID)
	if err != nil {
		t.Fatalf("read the other risk: %v", err)
	}
	if untouched.Title != "Other" {
		t.Fatalf("a PUT to one address rewrote another risk: %+v", untouched)
	}
}

func TestTheCallerOutranksTheBody(t *testing.T) {
	rm, ctx := newManager(t)
	risk := seedRisk(t, rm, ctx, "Signed")
	path := seedThreat(t, rm, ctx, risk.ID, "A route")
	control, err := rm.CreateControl(ctx, riskmanager.Control{Name: "A safeguard"})
	if err != nil {
		t.Fatalf("seed control: %v", err)
	}
	barrier, err := rm.PlaceBarrier(ctx, riskmanager.Barrier{PathwayID: path.ID, ControlID: control.ID})
	if err != nil {
		t.Fatalf("seed barrier: %v", err)
	}

	alice := func(context.Context) string { return "alice" }
	mux := muxFor(t, rm, routes.Mount{Caller: alice})

	w := do(t, mux, "POST", "/evidence", `{"barrier_id":"`+barrier.ID+`","kind":"test","outcome":"pass","summary":"ran it","recorded_by":"mallory"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("code = %d: %s", w.Code, w.Body)
	}
	held, err := rm.ListEvidence(ctx, riskmanager.EvidenceFilter{BarrierID: barrier.ID})
	if err != nil {
		t.Fatalf("list evidence: %v", err)
	}
	if len(held) != 1 || held[0].RecordedBy != "alice" {
		t.Fatalf("the body named who recorded it: %+v", held)
	}

	w = do(t, mux, "POST", "/assessments", `{"risk_id":"`+risk.ID+`","kind":"inherent","likelihood":3,"impact":4,"assessed_by":"mallory"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("code = %d: %s", w.Code, w.Body)
	}
	rated, err := rm.ListAssessments(ctx, riskmanager.AssessmentFilter{RiskID: risk.ID})
	if err != nil {
		t.Fatalf("list assessments: %v", err)
	}
	if len(rated) != 1 || rated[0].AssessedBy != "alice" {
		t.Fatalf("the body named who assessed it: %+v", rated)
	}
}

func TestSentinelsArriveAtTheirDeclaredStatus(t *testing.T) {
	mux, rm, ctx := newMux(t)
	if _, err := rm.CreateControl(ctx, riskmanager.Control{Name: "Taken"}); err != nil {
		t.Fatalf("seed control: %v", err)
	}

	for _, c := range []struct {
		name   string
		method string
		target string
		body   string
		want   int
	}{
		{"an unknown risk", "GET", "/risks/nobody", "", http.StatusNotFound},
		{"an unknown pathway", "GET", "/pathways/nobody", "", http.StatusNotFound},
		{"an unknown control", "GET", "/controls/nobody", "", http.StatusNotFound},
		{"a risk with no title", "POST", "/risks", `{"top_event":"nameless"}`, http.StatusBadRequest},
		{"a duplicate control name", "POST", "/controls", `{"name":"Taken"}`, http.StatusConflict},
		{"a pathway for a missing risk", "POST", "/pathways", `{"risk_id":"nobody","side":"threat","title":"A route"}`, http.StatusBadRequest},
		{"a malformed body", "POST", "/risks", `not json`, http.StatusBadRequest},
		{"a misspelled field", "POST", "/risks", `{"titel":"typo"}`, http.StatusBadRequest},
		{"a bowtie for nobody", "GET", "/risks/nobody/bowtie", "", http.StatusNotFound},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := do(t, mux, c.method, c.target, c.body)
			if w.Code != c.want {
				t.Fatalf("code = %d, want %d: %s", w.Code, c.want, w.Body)
			}
		})
	}
}

func TestADeleteAnswersNoContent(t *testing.T) {
	mux, rm, ctx := newMux(t)
	risk := seedRisk(t, rm, ctx, "Drafting")
	path := seedThreat(t, rm, ctx, risk.ID, "A route")

	w := do(t, mux, "DELETE", "/pathways/"+path.ID, "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("code = %d, want 204: %s", w.Code, w.Body)
	}
}

func TestAWriteAnswersCreated(t *testing.T) {
	mux, _, _ := newMux(t)
	w := do(t, mux, "POST", "/risks", `{"title":"Drafting","top_event":"Control is lost"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("code = %d, want 201: %s", w.Code, w.Body)
	}
}

func TestQueryBindsTheWholeFilter(t *testing.T) {
	mux, rm, ctx := newMux(t)
	first := seedRisk(t, rm, ctx, "First")
	second := seedRisk(t, rm, ctx, "Second")
	seedThreat(t, rm, ctx, first.ID, "Route one")
	seedThreat(t, rm, ctx, second.ID, "Route two")

	w := do(t, mux, "GET", "/threats?risk_id="+first.ID, "")
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d: %s", w.Code, w.Body)
	}
	var got []riskmanager.PathwaySummary
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0].Title != "Route one" {
		t.Fatalf("the query filter did not bind: %+v", got)
	}
}
