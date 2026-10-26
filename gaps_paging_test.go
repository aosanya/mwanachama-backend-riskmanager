package riskmanager_test

import (
	"encoding/json"
	"slices"
	"testing"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-riskmanager/models"
)

func TestTheKindFilterAppliesBeforePagingSoAPageCountsOnlyThatKind(t *testing.T) {
	rm, ctx := newManager(t)
	seedMixedRegister(t, rm, ctx)
	many := seedRisk(t, rm, ctx, "many open routes")
	for _, title := range []string{"r1", "r2", "r3", "r4", "r5"} {
		seedThreat(t, rm, ctx, many.ID, title)
	}

	full := seedGapsOf(t, rm, ctx, riskmanager.GapFilter{Kind: models.GapPathwayUnguarded})
	if len(full) != 7 {
		t.Fatalf("unguarded pathways across the register: %d, want 7", len(full))
	}
	page := seedGapsOf(t, rm, ctx, riskmanager.GapFilter{Kind: models.GapPathwayUnguarded, Limit: 2, Offset: 2})
	if !slices.Equal(seedGapKeys(page), seedGapKeys(full[2:4])) {
		t.Fatalf("page of unguarded pathways = %v, want items 3 and 4 of %v", seedGapKeys(page), seedGapKeys(full))
	}
}

func TestPagingWalksTheWholeListWithoutRepeatsOrGaps(t *testing.T) {
	rm, ctx := newManager(t)
	seedMixedRegister(t, rm, ctx)
	full := seedGapsOf(t, rm, ctx, riskmanager.GapFilter{})
	if len(full) != 13 {
		t.Fatalf("the fixture holds %d gaps, want 13", len(full))
	}

	var walked []riskmanager.Gap
	for offset := 0; ; offset += 4 {
		page := seedGapsOf(t, rm, ctx, riskmanager.GapFilter{Limit: 4, Offset: offset})
		if len(page) == 0 {
			break
		}
		if len(page) > 4 {
			t.Fatalf("a page of limit 4 held %d", len(page))
		}
		walked = append(walked, page...)
	}
	if !slices.Equal(seedGapKeys(walked), seedGapKeys(full)) {
		t.Fatalf("walking in pages of 4 gave %v, want %v", seedGapKeys(walked), seedGapKeys(full))
	}
}

func TestAnOffsetAtOrPastTheEndIsAnEmptyListAndNeverNull(t *testing.T) {
	rm, ctx := newManager(t)
	seedMixedRegister(t, rm, ctx)
	full := seedGapsOf(t, rm, ctx, riskmanager.GapFilter{})
	for _, offset := range []int{len(full), len(full) + 1, 100000} {
		got := seedGapsOf(t, rm, ctx, riskmanager.GapFilter{Offset: offset})
		raw, err := json.Marshal(got)
		seedNoErr(t, err)
		if string(raw) != "[]" {
			t.Fatalf("offset %d answered %s, want []", offset, raw)
		}
	}
}

func TestANegativeOffsetAndANonPositiveLimitFallBackToTheStartAndTheDefaultPage(t *testing.T) {
	rm, ctx := newManager(t)
	seedMixedRegister(t, rm, ctx)
	full := seedGapKeys(seedGapsOf(t, rm, ctx, riskmanager.GapFilter{}))
	for _, f := range []riskmanager.GapFilter{{Offset: -5}, {Limit: -1}, {Limit: 0}, {Limit: 5000}} {
		if got := seedGapKeys(seedGapsOf(t, rm, ctx, f)); !slices.Equal(got, full) {
			t.Fatalf("%+v answered %v, want the whole list %v", f, got, full)
		}
	}
}

func TestAnUnknownKindOrSeverityIsRefusedRatherThanAnsweredWithNothing(t *testing.T) {
	rm, ctx := newManager(t)
	seedMixedRegister(t, rm, ctx)
	for _, f := range []riskmanager.GapFilter{{Kind: "nonsense"}, {Severity: "urgent"}, {Kind: models.GapRiskUnowned, Severity: "critical"}} {
		_, err := rm.Gaps(ctx, f)
		seedWantErr(t, err, riskmanager.ErrInvalid)
	}
	for _, f := range []riskmanager.GapFilter{{Kind: models.GapBarrierUnassessed}, {Severity: models.SeverityLow}, {Kind: models.GapRiskUnowned, Severity: models.SeverityMedium}} {
		if _, err := rm.Gaps(ctx, f); err != nil {
			t.Fatalf("%+v was refused: %v", f, err)
		}
	}
}
