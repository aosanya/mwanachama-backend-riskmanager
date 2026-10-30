package riskmanager_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
)

func standOnNewRisks(t *testing.T, rm riskmanager.RiskManager, ctx context.Context, c riskmanager.Control, outcomes ...riskmanager.Outcome) {
	t.Helper()
	for i, outcome := range outcomes {
		r := seedRisk(t, rm, ctx, fmt.Sprintf("%s risk %d", c.Name, i))
		b := seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r.ID, "route").ID, c.ID)
		if outcome != "" {
			seedEvidence(t, rm, ctx, b.ID, outcome, "2026-01-01T00:00:00Z")
		}
	}
}

func failingNames(t *testing.T, rm riskmanager.RiskManager, ctx context.Context, f riskmanager.ControlFilter) []string {
	t.Helper()
	f.Failing = true
	got, err := rm.ListControls(ctx, f)
	seedNoErr(t, err)
	names := make([]string, len(got))
	for i, c := range got {
		names[i] = c.Name
	}
	return names
}

func TestControlsFailingOnTheMostBarriersComeFirst(t *testing.T) {
	rm, ctx := newManager(t)
	lightest := seedControl(t, rm, ctx, "lightest", "ops")
	weakest := seedControl(t, rm, ctx, "weakest", "sec")
	middle := seedControl(t, rm, ctx, "middle", "sec")
	standOnNewRisks(t, rm, ctx, lightest, riskmanager.OutcomeFail)
	standOnNewRisks(t, rm, ctx, weakest, riskmanager.OutcomeFail, riskmanager.OutcomePartial, riskmanager.OutcomeFail)
	standOnNewRisks(t, rm, ctx, middle, riskmanager.OutcomePartial, riskmanager.OutcomeFail, riskmanager.OutcomePass)

	if got := failingNames(t, rm, ctx, riskmanager.ControlFilter{}); !slices.Equal(got, []string{"weakest", "middle", "lightest"}) {
		t.Fatalf("failing controls: %v", got)
	}
	listed, err := rm.ListControls(ctx, riskmanager.ControlFilter{Failing: true})
	seedNoErr(t, err)
	if listed[0].FailingBarriers != 3 || listed[1].FailingBarriers != 2 || listed[2].FailingBarriers != 1 {
		t.Fatalf("failing barrier counts: %d %d %d", listed[0].FailingBarriers, listed[1].FailingBarriers, listed[2].FailingBarriers)
	}
}

func TestOnlyTheLatestEvidenceDecidesWhetherABarrierIsFailing(t *testing.T) {
	rm, ctx := newManager(t)
	c := seedControl(t, rm, ctx, "mended", "o")
	r := seedRisk(t, rm, ctx, "r")
	b := seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r.ID, "t").ID, c.ID)
	seedEvidence(t, rm, ctx, b.ID, riskmanager.OutcomeFail, "2026-01-01T00:00:00Z")
	if got := failingNames(t, rm, ctx, riskmanager.ControlFilter{}); !slices.Equal(got, []string{"mended"}) {
		t.Fatalf("after a failure: %v", got)
	}
	seedEvidence(t, rm, ctx, b.ID, riskmanager.OutcomePass, "2026-02-01T00:00:00Z")
	if got := failingNames(t, rm, ctx, riskmanager.ControlFilter{}); len(got) != 0 {
		t.Fatalf("after a later pass: %v", got)
	}
}

func TestAControlWithNoEvidenceOrOnlyOnARetiredRiskIsNotFailing(t *testing.T) {
	rm, ctx := newManager(t)
	standOnNewRisks(t, rm, ctx, seedControl(t, rm, ctx, "untried", "o"), "")
	gone := seedControl(t, rm, ctx, "gone", "o")
	r := seedRisk(t, rm, ctx, "retiring")
	b := seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r.ID, "t").ID, gone.ID)
	seedEvidence(t, rm, ctx, b.ID, riskmanager.OutcomeFail, "2026-01-01T00:00:00Z")
	seedRetire(t, rm, ctx, r.ID)
	if got := failingNames(t, rm, ctx, riskmanager.ControlFilter{}); len(got) != 0 {
		t.Fatalf("failing controls: %v", got)
	}
}

func TestTheFailingFilterNarrowsByOwnerAndLimitAndTiesGoByName(t *testing.T) {
	rm, ctx := newManager(t)
	standOnNewRisks(t, rm, ctx, seedControl(t, rm, ctx, "b", "sec"), riskmanager.OutcomeFail)
	standOnNewRisks(t, rm, ctx, seedControl(t, rm, ctx, "a", "sec"), riskmanager.OutcomeFail)
	standOnNewRisks(t, rm, ctx, seedControl(t, rm, ctx, "c", "ops"), riskmanager.OutcomeFail)
	if got := failingNames(t, rm, ctx, riskmanager.ControlFilter{}); !slices.Equal(got, []string{"a", "b", "c"}) {
		t.Fatalf("ties: %v", got)
	}
	if got := failingNames(t, rm, ctx, riskmanager.ControlFilter{Owner: "sec"}); !slices.Equal(got, []string{"a", "b"}) {
		t.Fatalf("owner: %v", got)
	}
	if got := failingNames(t, rm, ctx, riskmanager.ControlFilter{Limit: 2}); !slices.Equal(got, []string{"a", "b"}) {
		t.Fatalf("limit: %v", got)
	}
}

func TestWithoutTheFailingFilterEveryControlIsListedByNameAsBefore(t *testing.T) {
	rm, ctx := newManager(t)
	standOnNewRisks(t, rm, ctx, seedControl(t, rm, ctx, "b", "o"), riskmanager.OutcomeFail)
	seedControl(t, rm, ctx, "a", "o")
	got, err := rm.ListControls(ctx, riskmanager.ControlFilter{})
	seedNoErr(t, err)
	if len(got) != 2 || got[0].Name != "a" || got[1].FailingBarriers != 0 {
		t.Fatalf("controls: %+v", got)
	}
}
