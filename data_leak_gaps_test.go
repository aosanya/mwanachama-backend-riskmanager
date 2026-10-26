package riskmanager_test

import (
	"testing"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-riskmanager/models"
)

func TestDLPOnPromptsButNothingProvenOnAgentToolCallsShowsAsGaps(t *testing.T) {
	w := leakBuild(t)
	gaps, err := w.rm.Gaps(w.ctx, riskmanager.GapFilter{RiskID: w.risk.ID})
	seedNoErr(t, err)

	open := seedSubjects(gaps, models.GapPathwayUnguarded)
	for _, title := range []string{"Provider and integration leakage", "Logging and observability leakage", "Cross-tenant / session leakage"} {
		if !open[w.pathways[title].ID] {
			t.Fatalf("%q is not reported unguarded: %v", title, open)
		}
	}
	if len(open) != 3 {
		t.Fatalf("unguarded pathways: %v", open)
	}

	untested := seedSubjects(gaps, models.GapBarrierUntested)
	for _, name := range leakThreats[3].controls {
		if !untested[w.barriers["Agent/tool misuse|"+name].ID] {
			t.Fatalf("the agent barrier %q is not reported untested", name)
		}
	}
	dlp := w.barriers["Prompt input leakage|input DLP"].ID
	if untested[dlp] {
		t.Fatal("input DLP, which was tested, is reported untested")
	}
	failing := seedSubjects(gaps, models.GapBarrierFailing)
	if len(failing) != 1 || !failing[w.barriers["RAG / retrieval leakage|permission-aware retrieval"].ID] {
		t.Fatalf("failing barriers: %v", failing)
	}
	for _, g := range gaps {
		if g.SubjectID == dlp {
			t.Fatalf("a proven barrier is reported: %+v", g)
		}
	}
	kinds := seedKinds(gaps)
	if kinds[models.GapRiskUnowned]+kinds[models.GapRiskUnrated]+kinds[models.GapRiskOneSided]+kinds[models.GapControlUnowned] != 0 {
		t.Fatalf("the risk is owned, rated, two-sided and every control is operated, yet: %v", kinds)
	}
}

func TestTheUnguardedThreatsAndTheFailingTestLeadTheAnswerAndTheUnassessedBarriersFollow(t *testing.T) {
	w := leakBuild(t)
	gaps, err := w.rm.Gaps(w.ctx, riskmanager.GapFilter{RiskID: w.risk.ID})
	seedNoErr(t, err)

	leading := 0
	for leading < len(gaps) && gaps[leading].Severity == models.SeverityHigh {
		leading++
	}
	first := seedKinds(gaps[:leading])
	if leading != 4 || first[models.GapPathwayUnguarded] != 3 || first[models.GapBarrierFailing] != 1 {
		t.Fatalf("the answer opens with %d high gaps %v, want the three unguarded threats and the failing retrieval test", leading, first)
	}
	for _, g := range gaps[leading:] {
		if g.Severity != models.SeverityLow {
			t.Fatalf("beyond the high gaps this register should hold only low ones, found %+v", g)
		}
	}

	judged := len(w.barriers) - 1
	kinds := seedKinds(gaps)
	if kinds[models.GapBarrierUnassessed] != judged || kinds[models.GapBarrierUntested] != judged {
		t.Fatalf("%d barriers are unjudged and untested, got %v", judged, kinds)
	}
	if kinds[models.GapBarrierDesign]+kinds[models.GapBarrierCoverage] != 0 {
		t.Fatalf("nothing was judged partial or inadequate, yet: %v", kinds)
	}
	if want := 4 + 2*judged; len(gaps) != want {
		t.Fatalf("%d gaps, want %d: 4 high and an unassessed and an untested gap for each of %d barriers", len(gaps), want, judged)
	}
}

func TestTheHighGapsOfTheWholeRegisterIncludeTheSubBowTiesThreatsNothingStandsOnYet(t *testing.T) {
	w := leakBuild(t)
	high, err := w.rm.Gaps(w.ctx, riskmanager.GapFilter{Severity: models.SeverityHigh})
	seedNoErr(t, err)
	kinds := seedKinds(high)
	if len(high) != 4+len(leakSubThreats) || kinds[models.GapPathwayUnguarded] != 3+len(leakSubThreats) || kinds[models.GapBarrierFailing] != 1 {
		t.Fatalf("the register's high gaps: %d %v, want the four of the parent plus the sub-bow-tie's %d bare threats", len(high), kinds, len(leakSubThreats))
	}
	onChild := 0
	for _, g := range high {
		if g.RiskID == w.child.ID {
			onChild++
		}
	}
	if onChild != len(leakSubThreats) {
		t.Fatalf("%d of the high gaps belong to the sub-bow-tie, want %d", onChild, len(leakSubThreats))
	}
}
