package riskmanager_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-riskmanager/models"
)

const everything = 100000

const maxQueriesForGaps = 8

func seedFullRisk(t *testing.T, rm riskmanager.RiskManager, ctx context.Context, n int) riskmanager.Risk {
	t.Helper()
	r := seedOpenRisk(t, rm, ctx, fmt.Sprintf("risk %d", n))
	guarded := seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r.ID, "guarded threat").ID, seedControl(t, rm, ctx, fmt.Sprintf("guard %d", n), "the team").ID)
	seedEvidence(t, rm, ctx, guarded.ID, riskmanager.OutcomePass, "2026-01-01T00:00:00Z")
	seedEvidence(t, rm, ctx, guarded.ID, riskmanager.Outcome([]string{"pass", "fail", "partial"}[n%3]), "2026-02-01T00:00:00Z")
	seedThreat(t, rm, ctx, r.ID, "open threat")
	seedBarrier(t, rm, ctx, seedConsequence(t, rm, ctx, r.ID, "consequence").ID, seedControl(t, rm, ctx, fmt.Sprintf("cleanup %d", n), "").ID)
	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentInherent, 3, 3, "2026-01-01T00:00:00Z")
	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentInherent, 4, 4, "2026-02-01T00:00:00Z")
	seedAction(t, rm, ctx, r.ID, guarded.ID, "overdue", "2026-01-01T00:00:00Z")
	seedAction(t, rm, ctx, r.ID, "", "undated", "")
	return r
}

func TestEveryRiskIsReviewedHoweverManyThereAre(t *testing.T) {
	rm, ctx := newManager(t)
	const risks = 130
	for i := range risks {
		seedRisk(t, rm, ctx, fmt.Sprintf("risk %d", i))
	}
	got := seedGapsOf(t, rm, ctx, riskmanager.GapFilter{Limit: everything})
	if want := risks * 4; len(got) != want {
		t.Fatalf("%d risks with no threats, consequences or ratings gave %d gaps, want %d", risks, len(got), want)
	}
}

func TestABowTieHoldingMoreThanAHundredBarriersIsReadWhole(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "wide")
	threat := seedThreat(t, rm, ctx, r.ID, "t")
	const barriers = 120
	for i := range barriers {
		seedBarrier(t, rm, ctx, threat.ID, seedControl(t, rm, ctx, fmt.Sprintf("guard %d", i), "o").ID)
	}
	got := seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r.ID, Kind: models.GapBarrierUntested, Limit: everything})
	if len(got) != barriers {
		t.Fatalf("%d untested barriers were reported as %d", barriers, len(got))
	}
}

func TestGapsReadTheRegisterInAFixedNumberOfQueriesWhateverItsSize(t *testing.T) {
	small, smallCtx, smallMeter := newMeteredManager(t)
	for i := range 3 {
		seedFullRisk(t, small, smallCtx, i)
	}
	smallMeter.reset()
	seedGapsOf(t, small, smallCtx, riskmanager.GapFilter{Limit: everything})
	smallQueries := smallMeter.queries.Load()

	large, largeCtx, largeMeter := newMeteredManager(t)
	for i := range 150 {
		seedFullRisk(t, large, largeCtx, i)
	}
	largeMeter.reset()
	seedGapsOf(t, large, largeCtx, riskmanager.GapFilter{Limit: everything})
	largeQueries := largeMeter.queries.Load()

	if largeQueries != smallQueries || largeQueries > maxQueriesForGaps {
		t.Fatalf("3 risks took %d queries and 150 risks took %d, want the same and at most %d", smallQueries, largeQueries, maxQueriesForGaps)
	}
}

func TestOneRisksGapsAreAlsoReadInAFixedNumberOfQueries(t *testing.T) {
	rm, ctx, meter := newMeteredManager(t)
	var last riskmanager.Risk
	for i := range 20 {
		last = seedFullRisk(t, rm, ctx, i)
	}
	meter.reset()
	seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: last.ID, Limit: everything})
	if n := meter.queries.Load(); n > maxQueriesForGaps {
		t.Fatalf("one risk's gaps took %d queries, want at most %d", n, maxQueriesForGaps)
	}
}

func TestGapsReadAcrossTheRegisterAreTheGapsEachBowTieReportsAlone(t *testing.T) {
	rm, ctx := newManager(t)
	var ids []string
	for i := range 12 {
		ids = append(ids, seedFullRisk(t, rm, ctx, i).ID)
	}
	ids = append(ids, seedRisk(t, rm, ctx, "bare").ID)
	retired := seedRisk(t, rm, ctx, "retired")
	seedRetire(t, rm, ctx, retired.ID)

	want := []riskmanager.Gap{}
	for _, id := range ids {
		b, err := rm.Bowtie(ctx, id)
		seedNoErr(t, err)
		want = append(want, b.Gaps(time.Now())...)
	}
	models.RankGaps(want)

	got := seedGapsOf(t, rm, ctx, riskmanager.GapFilter{Limit: everything})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("across the register %d gaps, assembled one bow-tie at a time %d\nfirst difference: %s", len(got), len(want), firstDifference(got, want))
	}
}

func firstDifference(got, want []riskmanager.Gap) string {
	for i := 0; i < len(got) && i < len(want); i++ {
		if got[i] != want[i] {
			return fmt.Sprintf("at %d got %+v want %+v", i, got[i], want[i])
		}
	}
	return "one list is a prefix of the other"
}
