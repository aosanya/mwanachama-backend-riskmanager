package riskmanager_test

import (
	"testing"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-riskmanager/models"
)

func TestAFullyAssessedProvenAndOwnedBowTieHasNoGaps(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedCleanRisk(t, rm, ctx, "clean")
	if got := seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r.ID}); len(got) != 0 {
		t.Fatalf("a clean bow-tie reports %d gap(s): %+v", len(got), got)
	}
}

func TestARiskNobodyOwnsIsAGap(t *testing.T) {
	rm, ctx := newManager(t)
	r, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: "orphan", TopEvent: "e"})
	seedNoErr(t, err)
	got := seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r.ID, Kind: models.GapRiskUnowned})
	if len(got) != 1 || got[0].Subject != "risk" || got[0].SubjectID != r.ID || got[0].RiskID != r.ID || got[0].Message == "" {
		t.Fatalf("gaps: %+v", got)
	}
}

func TestABowTieWithOneSideMissingReportsEachMissingSide(t *testing.T) {
	rm, ctx := newManager(t)
	bare := seedRisk(t, rm, ctx, "bare")
	if n := seedKinds(seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: bare.ID}))[models.GapRiskOneSided]; n != 2 {
		t.Fatalf("no threats and no consequences reported %d one-sided gap(s), want 2", n)
	}
	left := seedRisk(t, rm, ctx, "left only")
	seedThreat(t, rm, ctx, left.ID, "t")
	if n := seedKinds(seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: left.ID}))[models.GapRiskOneSided]; n != 1 {
		t.Fatalf("no consequences reported %d one-sided gap(s), want 1", n)
	}
}

func TestEachMissingRatingKindIsItsOwnGap(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	if n := seedKinds(seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r.ID}))[models.GapRiskUnrated]; n != 2 {
		t.Fatalf("nothing rated reported %d, want inherent and residual both missing", n)
	}
	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentInherent, 3, 3, "2026-01-01T00:00:00Z")
	if n := seedKinds(seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r.ID}))[models.GapRiskUnrated]; n != 1 {
		t.Fatalf("only inherent rated reported %d, want 1", n)
	}
	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentResidual, 2, 2, "2026-01-01T00:00:00Z")
	if n := seedKinds(seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r.ID}))[models.GapRiskUnrated]; n != 0 {
		t.Fatalf("both rated still reports %d", n)
	}
}

func TestAPathwayWithNothingStandingOnItIsUnguardedOnEitherSide(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	threat := seedThreat(t, rm, ctx, r.ID, "t")
	consequence := seedConsequence(t, rm, ctx, r.ID, "c")
	guarded := seedThreat(t, rm, ctx, r.ID, "guarded")
	seedBarrier(t, rm, ctx, guarded.ID, seedControl(t, rm, ctx, "guard", "o").ID)
	subjects := seedSubjects(seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r.ID}), models.GapPathwayUnguarded)
	if len(subjects) != 2 || !subjects[threat.ID] || !subjects[consequence.ID] || subjects[guarded.ID] {
		t.Fatalf("unguarded pathways: %v", subjects)
	}
}

func TestABarrierThatHasNotReachedTestedIsAnUntestedGapUntilItDoes(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	b := seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r.ID, "t").ID, seedControl(t, rm, ctx, "c", "o").ID)
	for _, m := range []riskmanager.Maturity{riskmanager.MaturityProposed, riskmanager.MaturityDesigned, riskmanager.MaturityDeployed} {
		_, err := rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: b.ID, Maturity: &m})
		seedNoErr(t, err)
		if !seedSubjects(seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r.ID}), models.GapBarrierUntested)[b.ID] {
			t.Fatalf("a barrier at %s is not reported untested", m)
		}
	}
	seedEvidence(t, rm, ctx, b.ID, riskmanager.OutcomePass, "2026-01-01T00:00:00Z")
	for _, m := range []riskmanager.Maturity{riskmanager.MaturityTested, riskmanager.MaturityMonitored} {
		_, err := rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: b.ID, Maturity: &m})
		seedNoErr(t, err)
		if seedSubjects(seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r.ID}), models.GapBarrierUntested)[b.ID] {
			t.Fatalf("a barrier at %s is still reported untested", m)
		}
	}
}

func TestABarrierWhoseLatestEvidenceIsNotAPassIsFailingUntilAPassFollowsIt(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	b := seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r.ID, "t").ID, seedControl(t, rm, ctx, "c", "o").ID)
	failing := func() bool {
		return seedSubjects(seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r.ID}), models.GapBarrierFailing)[b.ID]
	}
	if failing() {
		t.Fatal("a barrier with nothing on record is reported failing")
	}
	seedEvidence(t, rm, ctx, b.ID, riskmanager.OutcomePass, "2026-01-01T00:00:00Z")
	if failing() {
		t.Fatal("a passing barrier is reported failing")
	}
	seedEvidence(t, rm, ctx, b.ID, riskmanager.OutcomePartial, "2026-02-01T00:00:00Z")
	if !failing() {
		t.Fatal("a partial result is not reported")
	}
	seedEvidence(t, rm, ctx, b.ID, riskmanager.OutcomeFail, "2026-03-01T00:00:00Z")
	if !failing() {
		t.Fatal("a failure is not reported")
	}
	seedEvidence(t, rm, ctx, b.ID, riskmanager.OutcomePass, "2026-04-01T00:00:00Z")
	if failing() {
		t.Fatal("a later pass did not clear the failure")
	}
}

func TestAControlNobodyOperatesIsReportedOncePerRiskHoweverManyPathwaysItStandsOn(t *testing.T) {
	rm, ctx := newManager(t)
	r1 := seedRisk(t, rm, ctx, "one")
	r2 := seedRisk(t, rm, ctx, "two")
	orphan := seedControl(t, rm, ctx, "orphan", "")
	seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r1.ID, "a").ID, orphan.ID)
	seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r1.ID, "b").ID, orphan.ID)
	seedBarrier(t, rm, ctx, seedConsequence(t, rm, ctx, r1.ID, "c").ID, orphan.ID)
	seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r2.ID, "d").ID, orphan.ID)

	if n := seedKinds(seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r1.ID}))[models.GapControlUnowned]; n != 1 {
		t.Fatalf("one risk reports the control %d times, want once", n)
	}
	every := seedGapsOf(t, rm, ctx, riskmanager.GapFilter{Kind: models.GapControlUnowned})
	if len(every) != 2 || every[0].SubjectID != orphan.ID || every[1].SubjectID != orphan.ID || every[0].RiskID == every[1].RiskID {
		t.Fatalf("across the register the control should be reported once for each risk: %+v", every)
	}
}

func TestAnActionWithNoDueDateOrOnePastItIsAGapAndAFinishedOneIsNot(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	undated := seedAction(t, rm, ctx, r.ID, "", "undated", "")
	overdue := seedAction(t, rm, ctx, r.ID, "", "overdue", "2020-01-01T00:00:00Z")
	moving := seedAction(t, rm, ctx, r.ID, "", "moving but late", "2020-01-01T00:00:00Z")
	done := seedAction(t, rm, ctx, r.ID, "", "done late", "2020-01-01T00:00:00Z")
	dropped := seedAction(t, rm, ctx, r.ID, "", "dropped late", "2020-01-01T00:00:00Z")
	future := seedAction(t, rm, ctx, r.ID, "", "not yet due", "2999-01-01T00:00:00Z")
	for id, next := range map[string]riskmanager.ActionStatus{
		moving.ID: riskmanager.ActionInProgress, done.ID: riskmanager.ActionDone, dropped.ID: riskmanager.ActionDropped,
	} {
		_, err := rm.SetActionStatus(ctx, id, next)
		seedNoErr(t, err)
	}

	gaps := seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r.ID})
	late := seedSubjects(gaps, models.GapActionOverdue)
	if len(late) != 2 || !late[overdue.ID] || !late[moving.ID] {
		t.Fatalf("overdue actions: %v", late)
	}
	missingDate := seedSubjects(gaps, models.GapActionUndated)
	if len(missingDate) != 1 || !missingDate[undated.ID] {
		t.Fatalf("undated actions: %v", missingDate)
	}
	if late[future.ID] || missingDate[future.ID] || late[done.ID] || late[dropped.ID] {
		t.Fatalf("an action that is not late is reported: %v %v", late, missingDate)
	}
}

func TestARetiredRiskReportsNoGapsHoweverBareItWas(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedOpenRisk(t, rm, ctx, "x")
	seedThreat(t, rm, ctx, r.ID, "unguarded")
	seedAction(t, rm, ctx, r.ID, "", "undated", "")
	if len(seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r.ID})) == 0 {
		t.Fatal("the bare risk should report gaps before it is retired")
	}
	seedRetire(t, rm, ctx, r.ID)
	if got := seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r.ID}); len(got) != 0 {
		t.Fatalf("a retired risk reports %d gap(s): %+v", len(got), got)
	}
	for _, g := range seedGapsOf(t, rm, ctx, riskmanager.GapFilter{}) {
		if g.RiskID == r.ID {
			t.Fatalf("the register-wide listing still includes the retired risk: %+v", g)
		}
	}
}

func TestTheRegisterWideListingCoversEveryRiskThatIsNotRetired(t *testing.T) {
	rm, ctx := newManager(t)
	draft := seedRisk(t, rm, ctx, "draft")
	open := seedOpenRisk(t, rm, ctx, "open")
	accepted := seedOpenRisk(t, rm, ctx, "accepted")
	seedAssessment(t, rm, ctx, accepted.ID, riskmanager.AssessmentResidual, 2, 2, "2026-01-01T00:00:00Z")
	_, err := rm.SetRiskStatus(ctx, accepted.ID, riskmanager.StatusAccepted)
	seedNoErr(t, err)
	retired := seedOpenRisk(t, rm, ctx, "retired")
	seedRetire(t, rm, ctx, retired.ID)

	seen := map[string]bool{}
	for _, g := range seedGapsOf(t, rm, ctx, riskmanager.GapFilter{}) {
		seen[g.RiskID] = true
	}
	if !seen[draft.ID] || !seen[open.ID] || !seen[accepted.ID] || seen[retired.ID] {
		t.Fatalf("risks with gaps: %v", seen)
	}
}

func TestGapsFilterByKindAndByRisk(t *testing.T) {
	rm, ctx := newManager(t)
	r1 := seedRisk(t, rm, ctx, "one")
	r2 := seedRisk(t, rm, ctx, "two")
	seedThreat(t, rm, ctx, r1.ID, "a")
	seedThreat(t, rm, ctx, r2.ID, "b")

	for _, g := range seedGapsOf(t, rm, ctx, riskmanager.GapFilter{Kind: models.GapPathwayUnguarded}) {
		if g.Kind != models.GapPathwayUnguarded {
			t.Fatalf("a kind filter returned %s", g.Kind)
		}
	}
	if n := len(seedGapsOf(t, rm, ctx, riskmanager.GapFilter{Kind: models.GapPathwayUnguarded})); n != 2 {
		t.Fatalf("unguarded pathways across the register: %d", n)
	}
	for _, g := range seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r1.ID}) {
		if g.RiskID != r1.ID {
			t.Fatalf("a risk filter returned a gap of %s", g.RiskID)
		}
	}
	if n := len(seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r1.ID, Kind: models.GapPathwayUnguarded})); n != 1 {
		t.Fatalf("one risk, one kind: %d", n)
	}
}

func TestGapsForAMissingRiskAreNotFound(t *testing.T) {
	rm, ctx := newManager(t)
	_, err := rm.Gaps(ctx, riskmanager.GapFilter{RiskID: "nowhere"})
	seedWantErr(t, err, riskmanager.ErrRiskNotFound)
}

func TestEveryGapNamesItsRiskItsSubjectAndSaysWhatIsWrong(t *testing.T) {
	rm, ctx := newManager(t)
	r, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: "everything wrong", TopEvent: "e"})
	seedNoErr(t, err)
	threat := seedThreat(t, rm, ctx, r.ID, "t")
	b := seedBarrier(t, rm, ctx, threat.ID, seedControl(t, rm, ctx, "c", "").ID)
	_, err = rm.UpdateBarrier(ctx, riskmanager.BarrierChange{
		ID: b.ID, Design: seedPtr(riskmanager.JudgementPartial), Coverage: seedPtr(riskmanager.JudgementInadequate),
	})
	seedNoErr(t, err)
	seedEvidence(t, rm, ctx, b.ID, riskmanager.OutcomeFail, "2026-01-01T00:00:00Z")
	seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r.ID, "fresh").ID, seedControl(t, rm, ctx, "fresh guard", "o").ID)
	seedThreat(t, rm, ctx, r.ID, "open route")
	seedAction(t, rm, ctx, r.ID, b.ID, "late", "2020-01-01T00:00:00Z")
	seedAction(t, rm, ctx, r.ID, "", "undated", "")

	subjectOf := map[riskmanager.GapKind]string{
		models.GapRiskUnowned: "risk", models.GapRiskOneSided: "risk", models.GapRiskUnrated: "risk", models.GapPathwayUnguarded: "pathway",
		models.GapBarrierDesign: "barrier", models.GapBarrierCoverage: "barrier", models.GapBarrierUnassessed: "barrier",
		models.GapBarrierUntested: "barrier", models.GapBarrierFailing: "barrier", models.GapControlUnowned: "control",
		models.GapActionUndated: "action", models.GapActionOverdue: "action",
	}
	seen := map[riskmanager.GapKind]bool{}
	for _, g := range seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r.ID}) {
		seen[g.Kind] = true
		if g.RiskID != r.ID || g.SubjectID == "" || g.Message == "" || g.Severity != g.Kind.Severity() {
			t.Fatalf("an incomplete gap: %+v", g)
		}
		if want, ok := subjectOf[g.Kind]; ok && g.Subject != want {
			t.Fatalf("%s names its subject as %q, want %q", g.Kind, g.Subject, want)
		}
	}
	for kind := range subjectOf {
		if !seen[kind] {
			t.Errorf("a bow-tie with everything wrong never reported %s", kind)
		}
	}
}
