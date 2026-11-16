package routes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
)

func liveMux(t *testing.T, mount Mount) *http.ServeMux {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	s, err := riskmanager.SpecForMount("test", "risks")
	if err != nil {
		t.Fatalf("spec: %v", err)
	}
	if err := riskmanager.Provision(db, s); err != nil {
		t.Fatalf("provision: %v", err)
	}
	manager, err := riskmanager.NewRiskManager(db, s)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	routes, err := BuildFor(manager, mount)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	mux := http.NewServeMux()
	for _, route := range routes {
		mux.HandleFunc(route.Pattern(""), route.Handler)
	}
	return mux
}

func call(mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("body %q is not an object: %v", rec.Body.String(), err)
	}
	return out
}

func newRisk(t *testing.T, mux *http.ServeMux, title string) string {
	t.Helper()
	rec := call(mux, "POST", "/risks", `{"title":"`+title+`","top_event":"event of `+title+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create risk: %d %s", rec.Code, rec.Body.String())
	}
	id, _ := decode(t, rec)["id"].(string)
	if id == "" {
		t.Fatalf("created risk has no id: %s", rec.Body.String())
	}
	return id
}

func TestEverySentinelThisModuleExportsIsMappedToAStatus(t *testing.T) {
	unmapped, err := Table.UnmappedSentinels()
	if err != nil {
		t.Fatalf("spec: %v", err)
	}
	if len(unmapped) != 0 {
		t.Errorf("sentinels with no declared status: %v", unmapped)
	}
}

func TestNothingInTheRiskRegisterIsAnonymous(t *testing.T) {
	if len(AnonymousActions) != 0 {
		t.Fatalf("AnonymousActions = %v, want none", AnonymousActions)
	}
	if got := PublicRoutes(&recordAuthors{}); len(got) != 0 {
		t.Errorf("PublicRoutes served %d anonymous addresses", len(got))
	}
	unknown, err := Table.UnknownAnonymousActions()
	if err != nil || len(unknown) != 0 {
		t.Errorf("unknown anonymous actions %v, err %v", unknown, err)
	}
}

func TestSplitGatesEveryAddressAndAgreesWithBuild(t *testing.T) {
	built, err := Build(&recordAuthors{})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	split := Split(&recordAuthors{}, Mount{})
	if len(split.Anonymous) != 0 || len(split.Gated) != len(built) {
		t.Errorf("anonymous %d gated %d, want 0 and %d", len(split.Anonymous), len(split.Gated), len(built))
	}
	if got := OperatorRoutes(&recordAuthors{}, Mount{}); len(got) != len(built) {
		t.Errorf("OperatorRoutes %d, want %d", len(got), len(built))
	}
	if got := Routes(&recordAuthors{}); len(got) != len(built) {
		t.Errorf("Routes %d, want %d", len(got), len(built))
	}
	shape, err := Table.Shape()
	if err != nil || len(shape) != len(built) {
		t.Errorf("shape %d routes, err %v; want %d", len(shape), err, len(built))
	}
	seen := map[string]bool{}
	for _, route := range built {
		if route.Action == "" {
			t.Errorf("%s %s has no action", route.Method, route.Path)
		}
		key := route.Method + " " + route.Path
		if seen[key] {
			t.Errorf("%s declared twice", key)
		}
		seen[key] = true
	}
}

func TestARefusingAuthorizerAnswers403ForEveryAddress(t *testing.T) {
	asked := map[string]bool{}
	mount := Mount{Authorize: func(_ context.Context, action string) error {
		asked[action] = true
		return errors.New("not yours")
	}}
	routes, err := BuildFor(&recordAuthors{}, mount)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	for _, route := range routes {
		rec := httptest.NewRecorder()
		route.Handler(rec, httptest.NewRequest(route.Method, "/", strings.NewReader(`{}`)))
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s answered %d to a refused caller, want 403", route.Action, rec.Code)
		}
		if !asked[route.Action] {
			t.Errorf("the authorizer was never asked about %s", route.Action)
		}
	}
}

func TestANewRiskIsADraftWhateverStatusTheBodyNames(t *testing.T) {
	mux := liveMux(t, Mount{})
	rec := call(mux, "POST", "/risks", `{"title":"Flooded depot","top_event":"Depot floods","status":"accepted"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	body := decode(t, rec)
	if body["status"] != "draft" || body["title"] != "Flooded depot" {
		t.Errorf("created risk = %v, want a draft titled Flooded depot", body)
	}
	read := call(mux, "GET", "/risks/"+body["id"].(string), "")
	if read.Code != http.StatusOK || decode(t, read)["top_event"] != "Depot floods" {
		t.Errorf("read back: %d %s", read.Code, read.Body.String())
	}
}

func TestAnUnknownRiskIsNotFoundOnReadUpdateAndStatus(t *testing.T) {
	mux := liveMux(t, Mount{})
	for _, c := range []struct{ method, path, body string }{
		{"GET", "/risks/nope", ""},
		{"PUT", "/risks/nope", `{"title":"x","top_event":"y"}`},
		{"POST", "/risks/nope/status", `{"status":"open"}`},
	} {
		if rec := call(mux, c.method, c.path, c.body); rec.Code != http.StatusNotFound {
			t.Errorf("%s %s answered %d %s, want 404", c.method, c.path, rec.Code, rec.Body.String())
		}
	}
}

func TestARiskWithoutATitleOrTopEventIsRefusedAs400(t *testing.T) {
	mux := liveMux(t, Mount{})
	for _, body := range []string{`{}`, `{"title":"only a title"}`, `{"top_event":"only an event"}`} {
		if rec := call(mux, "POST", "/risks", body); rec.Code != http.StatusBadRequest {
			t.Errorf("body %s answered %d %s, want 400", body, rec.Code, rec.Body.String())
		}
	}
	if rec := call(mux, "POST", "/risks", `{not json`); rec.Code != http.StatusBadRequest {
		t.Errorf("malformed json answered %d, want 400", rec.Code)
	}
}

func TestOpeningARiskNeedsAnOwnerAndAnOwnedRiskOpens(t *testing.T) {
	mux := liveMux(t, Mount{})
	id := newRisk(t, mux, "Unowned")
	rec := call(mux, "POST", "/risks/"+id+"/status", `{"status":"open"}`)
	if rec.Code < 400 || rec.Code >= 500 {
		t.Fatalf("opening an unowned risk answered %d %s, want a client refusal", rec.Code, rec.Body.String())
	}
	if got := decode(t, call(mux, "GET", "/risks/"+id, ""))["status"]; got != "draft" {
		t.Errorf("a refused opening left the risk %v, want draft", got)
	}
	owned := call(mux, "PUT", "/risks/"+id, `{"title":"Unowned","top_event":"event of Unowned","owner":"head of depots"}`)
	if owned.Code != http.StatusOK {
		t.Fatalf("assign owner: %d %s", owned.Code, owned.Body.String())
	}
	opened := call(mux, "POST", "/risks/"+id+"/status", `{"status":"open"}`)
	if opened.Code != http.StatusOK || decode(t, opened)["status"] != "open" {
		t.Errorf("opening an owned risk: %d %s", opened.Code, opened.Body.String())
	}
}

func TestAcceptingARiskWithNoResidualRatingIsRefusedAndRetiredIsTheEnd(t *testing.T) {
	mux := liveMux(t, Mount{})
	id := newRisk(t, mux, "Pump failure")
	call(mux, "PUT", "/risks/"+id, `{"title":"Pump failure","top_event":"event of Pump failure","owner":"ops"}`)
	if rec := call(mux, "POST", "/risks/"+id+"/status", `{"status":"open"}`); rec.Code != http.StatusOK {
		t.Fatalf("open: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(mux, "POST", "/risks/"+id+"/status", `{"status":"accepted"}`); rec.Code < 400 || rec.Code >= 500 {
		t.Errorf("accepting without a residual rating answered %d %s, want a client refusal", rec.Code, rec.Body.String())
	}
	if rec := call(mux, "POST", "/risks/"+id+"/status", `{"status":"retired"}`); rec.Code != http.StatusOK {
		t.Fatalf("retire: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(mux, "POST", "/risks/"+id+"/status", `{"status":"open"}`); rec.Code < 400 || rec.Code >= 500 {
		t.Errorf("reopening a retired risk answered %d, want a client refusal", rec.Code)
	}
	if rec := call(mux, "POST", "/risks/"+id+"/status", `{}`); rec.Code != http.StatusBadRequest {
		t.Errorf("a status change naming no status answered %d, want 400", rec.Code)
	}
}

func TestListingRisksNarrowsByStatusAndAndSearchesByTitle(t *testing.T) {
	mux := liveMux(t, Mount{})
	first := newRisk(t, mux, "Alpha")
	newRisk(t, mux, "Bravo")
	newRisk(t, mux, "Charlie")
	call(mux, "PUT", "/risks/"+first, `{"title":"Alpha","top_event":"event of Alpha","owner":"ops"}`)
	call(mux, "POST", "/risks/"+first+"/status", `{"status":"open"}`)

	var all []map[string]any
	rec := call(mux, "GET", "/risks", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &all); err != nil {
		t.Fatalf("list body: %v", err)
	}
	titles := map[any]bool{}
	for _, risk := range all {
		titles[risk["title"]] = true
	}
	if len(all) != 3 || !titles["Alpha"] || !titles["Bravo"] || !titles["Charlie"] {
		t.Errorf("all risks = %v, want Alpha, Bravo, Charlie", all)
	}
	var drafts []map[string]any
	rec = call(mux, "GET", "/risks?status=draft", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &drafts); err != nil || len(drafts) != 2 {
		t.Errorf("draft risks: %d %s (err %v), want two", rec.Code, rec.Body.String(), err)
	}
	var found []map[string]any
	rec = call(mux, "GET", "/risks?query=BRAVO", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &found); err != nil || len(found) != 1 || found[0]["title"] != "Bravo" {
		t.Errorf("search BRAVO: %d %s (err %v), want only Bravo", rec.Code, rec.Body.String(), err)
	}
}

func TestAPathwayForAnUnknownRiskIsRefusedAndNothingIsListed(t *testing.T) {
	mux := liveMux(t, Mount{})
	rec := call(mux, "POST", "/pathways", `{"risk_id":"ghost","side":"threat","title":"Heavy rain"}`)
	if rec.Code < 400 || rec.Code >= 500 {
		t.Errorf("pathway on a missing risk answered %d %s, want a client refusal", rec.Code, rec.Body.String())
	}
	if rec := call(mux, "GET", "/pathways/ghost", ""); rec.Code != http.StatusNotFound {
		t.Errorf("reading a missing pathway answered %d, want 404", rec.Code)
	}
	if rec := call(mux, "DELETE", "/pathways/ghost", ""); rec.Code != http.StatusNotFound {
		t.Errorf("deleting a missing pathway answered %d, want 404", rec.Code)
	}
}

func TestAPathwayIsAddedToItsRiskAndAnUnknownSideIsRefused(t *testing.T) {
	mux := liveMux(t, Mount{})
	id := newRisk(t, mux, "Flood")
	rec := call(mux, "POST", "/pathways", `{"risk_id":"`+id+`","side":"threat","title":"Heavy rain"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create pathway: %d %s", rec.Code, rec.Body.String())
	}
	pathway := decode(t, rec)
	if pathway["side"] != "threat" || pathway["risk_id"] != id {
		t.Errorf("pathway = %v, want a threat on %s", pathway, id)
	}
	if rec := call(mux, "POST", "/pathways", `{"risk_id":"`+id+`","side":"sideways","title":"Odd"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("a pathway on an unknown side answered %d %s, want 400", rec.Code, rec.Body.String())
	}
	if rec := call(mux, "DELETE", "/pathways/"+pathway["id"].(string), ""); rec.Code >= 300 {
		t.Errorf("deleting an empty pathway answered %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(mux, "GET", "/pathways/"+pathway["id"].(string), ""); rec.Code != http.StatusNotFound {
		t.Errorf("a deleted pathway still reads: %d", rec.Code)
	}
}

func TestTwoControlsCannotShareANameAndAMissingControlIs404(t *testing.T) {
	mux := liveMux(t, Mount{})
	if rec := call(mux, "POST", "/controls", `{"name":"Sump pump"}`); rec.Code != http.StatusCreated {
		t.Fatalf("create control: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(mux, "POST", "/controls", `{"name":"Sump pump"}`); rec.Code != http.StatusConflict {
		t.Errorf("a duplicate control name answered %d %s, want 409", rec.Code, rec.Body.String())
	}
	if rec := call(mux, "POST", "/controls", `{}`); rec.Code != http.StatusBadRequest {
		t.Errorf("a nameless control answered %d, want 400", rec.Code)
	}
	if rec := call(mux, "GET", "/controls/ghost", ""); rec.Code != http.StatusNotFound {
		t.Errorf("a missing control answered %d, want 404", rec.Code)
	}
}

func TestAnAssessmentOutsideOneToFiveIsRefusedAndTheCallerIsTheRater(t *testing.T) {
	mux := liveMux(t, Mount{Caller: theSession})
	id := newRisk(t, mux, "Scored")
	for _, body := range []string{
		`{"risk_id":"` + id + `","kind":"inherent","likelihood":6,"impact":3}`,
		`{"risk_id":"` + id + `","kind":"inherent","likelihood":3,"impact":0}`,
	} {
		if rec := call(mux, "POST", "/assessments", body); rec.Code != http.StatusBadRequest {
			t.Errorf("assessment %s answered %d %s, want 400", body, rec.Code, rec.Body.String())
		}
	}
	rec := call(mux, "POST", "/assessments", `{"risk_id":"`+id+`","kind":"inherent","likelihood":4,"impact":5}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("assessment: %d %s", rec.Code, rec.Body.String())
	}
	if got := decode(t, rec)["assessed_by"]; got != "the-session" {
		t.Errorf("assessed_by = %v, want the-session", got)
	}
}

func TestAnActionMovesOpenToDoneAndDoneIsTheEnd(t *testing.T) {
	mux := liveMux(t, Mount{})
	id := newRisk(t, mux, "Acted on")
	rec := call(mux, "POST", "/actions", `{"risk_id":"`+id+`","title":"Raise the pump"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create action: %d %s", rec.Code, rec.Body.String())
	}
	action := decode(t, rec)
	path := "/actions/" + action["id"].(string) + "/status"
	if rec := call(mux, "POST", path, `{"status":"done"}`); rec.Code != http.StatusOK || decode(t, rec)["status"] != "done" {
		t.Fatalf("complete: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(mux, "POST", path, `{"status":"in_progress"}`); rec.Code < 400 || rec.Code >= 500 {
		t.Errorf("reopening a done action answered %d %s, want a client refusal", rec.Code, rec.Body.String())
	}
	if rec := call(mux, "POST", "/actions/ghost/status", `{"status":"done"}`); rec.Code != http.StatusNotFound {
		t.Errorf("status of a missing action answered %d, want 404", rec.Code)
	}
}
