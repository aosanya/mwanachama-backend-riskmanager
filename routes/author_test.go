package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
)

type recordAuthors struct {
	riskmanager.RiskManager
	evidenceBy, assessmentBy string
}

func (r *recordAuthors) RecordEvidence(_ context.Context, e riskmanager.Evidence) (riskmanager.Evidence, error) {
	r.evidenceBy = e.RecordedBy
	return e, nil
}

func (r *recordAuthors) RecordAssessment(_ context.Context, a riskmanager.Assessment) (riskmanager.Assessment, error) {
	r.assessmentBy = a.AssessedBy
	return a, nil
}

func theSession(context.Context) string { return "the-session" }

func routeFor(t *testing.T, manager riskmanager.RiskManager, action string) Route {
	t.Helper()
	for _, route := range RoutesFor(manager, Mount{Caller: theSession}) {
		if route.Action == action {
			return route
		}
	}
	t.Fatalf("no route for %s", action)
	return Route{}
}

func post(handler http.HandlerFunc, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

func toolFor(t *testing.T, manager riskmanager.RiskManager, name string) Tool {
	t.Helper()
	tools, err := BuildTools(manager, Mount{Caller: theSession})
	if err != nil {
		t.Fatalf("BuildTools: %v", err)
	}
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("no tool %s", name)
	return Tool{}
}

func TestEvidenceIsRecordedByTheCallerNotTheBody(t *testing.T) {
	manager := &recordAuthors{}
	post(routeFor(t, manager, "riskmanager.evidence.record").Handler, `{"barrier_id":"b1","kind":"test","outcome":"pass","summary":"s","recorded_by":"forged-clerk"}`)
	if manager.evidenceBy != "the-session" {
		t.Errorf("the evidence names %q as its recorder; want the caller's own id the-session, never the body's forged-clerk", manager.evidenceBy)
	}
}

func TestAnAssessmentIsRatedByTheCallerNotTheBody(t *testing.T) {
	manager := &recordAuthors{}
	post(routeFor(t, manager, "riskmanager.assessment.record").Handler, `{"risk_id":"r1","kind":"inherent","likelihood":3,"impact":3,"assessed_by":"forged-clerk"}`)
	if manager.assessmentBy != "the-session" {
		t.Errorf("the assessment names %q as its rater; want the caller's own id the-session, never the body's forged-clerk", manager.assessmentBy)
	}
}

func TestEvidenceToolRefusesAForgedRecorderAndSignsTheCaller(t *testing.T) {
	manager := &recordAuthors{}
	tool := toolFor(t, manager, "riskmanager_evidence_record")
	forged := `{"barrier_id":"b1","kind":"test","outcome":"pass","summary":"s","recorded_by":"forged-clerk"}`
	if _, err := tool.Handle(context.Background(), dispatch.Request{Arguments: json.RawMessage(forged)}); err == nil || manager.evidenceBy != "" {
		t.Errorf("a forged recorder must be refused before anything is recorded; err %v, recorded by %q", err, manager.evidenceBy)
	}
	honest := `{"barrier_id":"b1","kind":"test","outcome":"pass","summary":"s"}`
	if _, err := tool.Handle(context.Background(), dispatch.Request{Arguments: json.RawMessage(honest)}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if manager.evidenceBy != "the-session" {
		t.Errorf("the evidence names %q as its recorder; want the-session", manager.evidenceBy)
	}
}

func TestAssessmentToolRefusesAForgedRaterAndSignsTheCaller(t *testing.T) {
	manager := &recordAuthors{}
	tool := toolFor(t, manager, "riskmanager_assessment_record")
	forged := `{"risk_id":"r1","kind":"inherent","likelihood":3,"impact":3,"assessed_by":"forged-clerk"}`
	if _, err := tool.Handle(context.Background(), dispatch.Request{Arguments: json.RawMessage(forged)}); err == nil || manager.assessmentBy != "" {
		t.Errorf("a forged rater must be refused before anything is recorded; err %v, rated by %q", err, manager.assessmentBy)
	}
	honest := `{"risk_id":"r1","kind":"inherent","likelihood":3,"impact":3}`
	if _, err := tool.Handle(context.Background(), dispatch.Request{Arguments: json.RawMessage(honest)}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if manager.assessmentBy != "the-session" {
		t.Errorf("the assessment names %q as its rater; want the-session", manager.assessmentBy)
	}
}
