package riskmanager_test

import (
	"context"
	"reflect"
	"sort"
	"testing"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-riskmanager/models"
)

type leakRoute struct {
	title, scenario string
	controls        []string
}

var leakThreats = []leakRoute{
	{"Prompt input leakage", "Employee pastes customer records, source code, or strategy documents into an unapproved AI service.", []string{"approved AI gateway", "input DLP"}},
	{"RAG / retrieval leakage", "An AI assistant retrieves documents the requesting user should not be permitted to see.", []string{"document-level authorisation", "permission-aware retrieval"}},
	{"Prompt injection", "A malicious document or webpage instructs an AI agent to send confidential data to an external destination.", []string{"treat retrieved content as untrusted", "tool authorisation", "restricted network egress", "scoped credentials", "token revocation"}},
	{"Agent/tool misuse", "An AI agent with excessive permissions sends sensitive data via email, API, or external tools.", []string{"least-privilege tools", "approval gates", "restricted network egress", "scoped credentials", "agent kill switch"}},
	{"Model output leakage", "An AI system reveals confidential information through its responses, including memorized training data.", []string{"output DLP"}},
	{"Provider and integration leakage", "Data is exposed through third-party processing, logging, retention, or an insecure connector.", nil},
	{"Logging and observability leakage", "Sensitive prompts, responses, or retrieved content are exposed in telemetry or debugging systems.", nil},
	{"Cross-tenant / session leakage", "A configuration or isolation failure makes one user's information accessible to another.", nil},
}

var leakIllustrativeConsequences = []leakRoute{
	{"Regulatory penalty", "A regulator fines the organisation for the disclosure.", []string{"notify affected parties"}},
	{"Customer harm and notification duty", "Customers whose records were exposed must be told and may be harmed.", []string{"notify affected parties", "contain exposure"}},
	{"Loss of intellectual property", "Source code or strategy documents reach a competitor.", []string{"revoke access", "contain exposure"}},
	{"Contract breach", "A confidentiality clause with a partner is broken.", []string{"recover"}},
}

var leakSubThreats = []string{"Inherited document permissions", "Stale access-control lists", "Overly broad retrieval", "Cross-user memory", "Prompt injection through a retrieved page", "Insecure connectors"}

type leakWorld struct {
	rm       riskmanager.RiskManager
	ctx      context.Context
	risk     riskmanager.Risk
	child    riskmanager.Risk
	pathways map[string]riskmanager.Pathway
	controls map[string]riskmanager.Control
	barriers map[string]riskmanager.Barrier
}

func (w *leakWorld) control(t *testing.T, name string) riskmanager.Control {
	t.Helper()
	if c, ok := w.controls[name]; ok {
		return c
	}
	w.controls[name] = seedControl(t, w.rm, w.ctx, name, "security team")
	return w.controls[name]
}

func (w *leakWorld) route(t *testing.T, side riskmanager.Side, r leakRoute) {
	t.Helper()
	p, err := w.rm.CreatePathway(w.ctx, riskmanager.Pathway{RiskID: w.risk.ID, Side: side, Title: r.title, Scenario: r.scenario})
	seedNoErr(t, err)
	w.pathways[r.title] = p
	for i, name := range r.controls {
		b, err := w.rm.PlaceBarrier(w.ctx, riskmanager.Barrier{PathwayID: p.ID, ControlID: w.control(t, name).ID, Position: i + 1})
		seedNoErr(t, err)
		w.barriers[r.title+"|"+name] = b
	}
}

func leakBuild(t *testing.T) *leakWorld {
	t.Helper()
	rm, ctx := newManager(t)
	w := &leakWorld{rm: rm, ctx: ctx, pathways: map[string]riskmanager.Pathway{}, controls: map[string]riskmanager.Control{}, barriers: map[string]riskmanager.Barrier{}}
	risk, err := rm.CreateRisk(ctx, riskmanager.Risk{
		Title:    "Loss of control over sensitive enterprise data through AI",
		TopEvent: "Sensitive information is disclosed, made accessible, or transmitted to an unauthorized party or system through an AI application, model, agent, or connected service.",
		Hazard:   "Sensitive enterprise data",
		Owner:    "chief information security officer",
	})
	seedNoErr(t, err)
	w.risk, err = rm.SetRiskStatus(ctx, risk.ID, riskmanager.StatusOpen)
	seedNoErr(t, err)
	for kind, scale := range map[riskmanager.AssessmentKind][2]int{
		riskmanager.AssessmentInherent: {4, 5}, riskmanager.AssessmentResidual: {3, 4}, riskmanager.AssessmentTarget: {2, 3},
	} {
		seedAssessment(t, rm, ctx, w.risk.ID, kind, scale[0], scale[1], "2026-02-01T00:00:00Z")
	}
	for _, r := range leakThreats {
		w.route(t, riskmanager.SideThreat, r)
	}
	for _, r := range leakIllustrativeConsequences {
		w.route(t, riskmanager.SideConsequence, r)
	}

	w.child, err = rm.CreateRisk(ctx, riskmanager.Risk{
		Title:           "An enterprise AI assistant discloses restricted information to an unauthorized user",
		TopEvent:        "An enterprise AI assistant discloses restricted information to an unauthorized user.",
		Owner:           "head of knowledge management",
		ParentPathwayID: w.pathways["RAG / retrieval leakage"].ID,
	})
	seedNoErr(t, err)
	for _, title := range leakSubThreats {
		seedThreat(t, rm, ctx, w.child.ID, title)
	}

	dlp := w.barriers["Prompt input leakage|input DLP"]
	seedEvidence(t, rm, ctx, dlp.ID, riskmanager.OutcomePass, "2026-03-01T09:00:00Z")
	_, err = rm.UpdateBarrier(ctx, riskmanager.BarrierChange{
		ID: dlp.ID, Design: seedPtr(riskmanager.JudgementAdequate), Coverage: seedPtr(riskmanager.JudgementAdequate), Maturity: seedPtr(riskmanager.MaturityTested),
	})
	seedNoErr(t, err)
	seedEvidence(t, rm, ctx, w.barriers["RAG / retrieval leakage|permission-aware retrieval"].ID, riskmanager.OutcomeFail, "2026-03-05T09:00:00Z")
	return w
}

func leakNames(measures []riskmanager.Measure) []string {
	out := make([]string, len(measures))
	for i, m := range measures {
		out[i] = m.Control.Name
	}
	sort.Strings(out)
	return out
}

func TestWhatAreTheThreatsOfTheDataLeakBowTie(t *testing.T) {
	w := leakBuild(t)
	got, err := w.rm.Threats(w.ctx, riskmanager.PathwayFilter{RiskID: w.risk.ID})
	seedNoErr(t, err)
	if len(got) != len(leakThreats) {
		t.Fatalf("%d threats, want the chat's %d: %+v", len(got), len(leakThreats), got)
	}
	for i, want := range leakThreats {
		if got[i].Title != want.title || got[i].Scenario != want.scenario || got[i].BarrierCount != len(want.controls) {
			t.Fatalf("threat %d = %q (%d barriers), want %q (%d)", i, got[i].Title, got[i].BarrierCount, want.title, len(want.controls))
		}
	}
	if rag := got[1]; rag.Drilldowns != 1 {
		t.Fatalf("the RAG threat drills into %d risk(s), want the one sub-bow-tie", rag.Drilldowns)
	}
	agents, err := w.rm.Threats(w.ctx, riskmanager.PathwayFilter{RiskID: w.risk.ID, Query: "AGENT"})
	seedNoErr(t, err)
	if len(agents) != 2 || agents[0].Title != "Prompt injection" || agents[1].Title != "Agent/tool misuse" {
		t.Fatalf("threats involving agents, asked in plain words: %+v", agents)
	}
}

func TestWhatAreTheConsequencesOfTheDataLeakBowTieIllustrative(t *testing.T) {
	w := leakBuild(t)
	got, err := w.rm.Consequences(w.ctx, riskmanager.PathwayFilter{RiskID: w.risk.ID})
	seedNoErr(t, err)
	if len(got) != len(leakIllustrativeConsequences) {
		t.Fatalf("%d consequences: %+v", len(got), got)
	}
	for i, want := range leakIllustrativeConsequences {
		if got[i].Title != want.title || got[i].Side != riskmanager.SideConsequence || got[i].BarrierCount != len(want.controls) {
			t.Fatalf("consequence %d = %+v, want %q with %d barriers", i, got[i], want.title, len(want.controls))
		}
	}
}

func TestWhatAreThePreventiveMeasuresAgainstTheRAGThreat(t *testing.T) {
	w := leakBuild(t)
	rag := w.pathways["RAG / retrieval leakage"]
	got, err := w.rm.PreventiveMeasures(w.ctx, riskmanager.MeasureFilter{PathwayID: rag.ID})
	seedNoErr(t, err)
	if len(got) != 2 || got[0].Control.Name != "document-level authorisation" || got[1].Control.Name != "permission-aware retrieval" {
		t.Fatalf("measures against the RAG threat: %v", leakNames(got))
	}
	if got[0].Standing != riskmanager.StandingUnproven || got[1].Standing != riskmanager.StandingFailing || got[1].Latest == nil || got[1].Latest.Outcome != riskmanager.OutcomeFail {
		t.Fatalf("standing of the two measures = %s and %s, latest of the second %+v", got[0].Standing, got[1].Standing, got[1].Latest)
	}
}

func TestWhatAreThePreventiveMeasuresAgainstTheWholeDataLeakRiskAndHowTheyStand(t *testing.T) {
	w := leakBuild(t)
	got, err := w.rm.PreventiveMeasures(w.ctx, riskmanager.MeasureFilter{RiskID: w.risk.ID})
	seedNoErr(t, err)
	if len(got) != 15 {
		t.Fatalf("%d preventive measures, want 15: %v", len(got), leakNames(got))
	}
	byStanding := map[riskmanager.Standing]int{}
	for _, m := range got {
		if m.Pathway.Side != riskmanager.SideThreat {
			t.Fatalf("%q is not on a threat", m.Control.Name)
		}
		byStanding[m.Standing]++
	}
	if byStanding[riskmanager.StandingProven] != 1 || byStanding[riskmanager.StandingFailing] != 1 || byStanding[riskmanager.StandingUnproven] != 13 {
		t.Fatalf("standing across the risk = %v, want 1 proven, 1 failing, 13 unproven", byStanding)
	}
	dlp, err := w.rm.PreventiveMeasures(w.ctx, riskmanager.MeasureFilter{RiskID: w.risk.ID, Query: "dlp"})
	seedNoErr(t, err)
	if names := leakNames(dlp); !reflect.DeepEqual(names, []string{"input DLP", "output DLP"}) {
		t.Fatalf("measures mentioning DLP: %v", names)
	}
}

func TestWhatAreTheMitigatingMeasuresAgainstTheConsequences(t *testing.T) {
	w := leakBuild(t)
	got, err := w.rm.MitigatingMeasures(w.ctx, riskmanager.MeasureFilter{RiskID: w.risk.ID})
	seedNoErr(t, err)
	want := []string{"contain exposure", "contain exposure", "notify affected parties", "notify affected parties", "recover", "revoke access"}
	if names := leakNames(got); !reflect.DeepEqual(names, want) {
		t.Fatalf("mitigating measures: %v, want %v", names, want)
	}
	harm, err := w.rm.MitigatingMeasures(w.ctx, riskmanager.MeasureFilter{PathwayID: w.pathways["Customer harm and notification duty"].ID})
	seedNoErr(t, err)
	if names := leakNames(harm); !reflect.DeepEqual(names, []string{"contain exposure", "notify affected parties"}) {
		t.Fatalf("against customer harm: %v", names)
	}
	_, err = w.rm.MitigatingMeasures(w.ctx, riskmanager.MeasureFilter{PathwayID: w.pathways["RAG / retrieval leakage"].ID})
	seedWantErr(t, err, riskmanager.ErrInvalid)
}

func TestWhatReliesOnRestrictedNetworkEgressAndOnScopedCredentials(t *testing.T) {
	w := leakBuild(t)
	for _, name := range []string{"restricted network egress", "scoped credentials"} {
		got, err := w.rm.Reliance(w.ctx, w.controls[name].ID)
		seedNoErr(t, err)
		titles := []string{}
		for _, m := range got {
			titles = append(titles, m.Pathway.Title)
		}
		sort.Strings(titles)
		if !reflect.DeepEqual(titles, []string{"Agent/tool misuse", "Prompt injection"}) {
			t.Fatalf("%s is relied on by %v, want the injection and agent threats", name, titles)
		}
	}
	got, err := w.rm.Reliance(w.ctx, w.controls["contain exposure"].ID)
	seedNoErr(t, err)
	if len(got) != 2 || got[0].Pathway.Side != riskmanager.SideConsequence || got[1].Pathway.Side != riskmanager.SideConsequence {
		t.Fatalf("contain exposure: %+v", got)
	}
}

func TestTheSubBowTieDrillsFromTheRAGThreatAndHasThreatsOfItsOwn(t *testing.T) {
	w := leakBuild(t)
	bt, err := w.rm.Bowtie(w.ctx, w.risk.ID)
	seedNoErr(t, err)
	var rag models.Arm
	for _, arm := range bt.Threats {
		if arm.Pathway.ID == w.pathways["RAG / retrieval leakage"].ID {
			rag = arm
		}
	}
	if len(rag.Drilldowns) != 1 || rag.Drilldowns[0].ID != w.child.ID {
		t.Fatalf("drill-downs on the RAG arm: %+v", rag.Drilldowns)
	}
	if bt.Inherent == nil || bt.Inherent.Band != models.BandCritical || bt.Residual.Band != models.BandHigh || bt.Target.Band != models.BandMedium {
		t.Fatalf("ratings: %+v %+v %+v", bt.Inherent, bt.Residual, bt.Target)
	}
	own, err := w.rm.Threats(w.ctx, riskmanager.PathwayFilter{RiskID: w.child.ID})
	seedNoErr(t, err)
	if len(own) != len(leakSubThreats) {
		t.Fatalf("the sub-bow-tie has %d threats, want %d", len(own), len(leakSubThreats))
	}
	kids, err := w.rm.ListRisks(w.ctx, riskmanager.RiskFilter{ParentPathwayID: w.pathways["RAG / retrieval leakage"].ID})
	seedNoErr(t, err)
	if len(kids) != 1 || kids[0].ID != w.child.ID {
		t.Fatalf("risks drilled from the RAG threat: %+v", kids)
	}
}
