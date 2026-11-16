package riskmanager_test

import (
	"context"
	"fmt"
	"math"
	"testing"

	rm "github.com/aosanya/mwanachama-backend-riskmanager"
)

func fileRisks(t *testing.T, m rm.RiskManager, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		title := fmt.Sprintf("Risk %04d", i)
		if _, err := m.CreateRisk(context.Background(), rm.Risk{Title: title, TopEvent: title}); err != nil {
			t.Fatalf("CreateRisk: %v", err)
		}
	}
}

func TestTheRiskListNeverAnswersMoreThanMaxPageWhateverTheLimit(t *testing.T) {
	m := newManager(t)
	fileRisks(t, m, rm.MaxPage+3)
	for _, limit := range []int{rm.MaxPage + 1, 100000, math.MaxInt32 + 1, math.MaxInt64} {
		got, err := m.ListRisks(context.Background(), rm.RiskFilter{Limit: limit})
		if err != nil {
			t.Fatalf("limit %d: %v", limit, err)
		}
		if len(got) != rm.MaxPage {
			t.Fatalf("limit %d answered %d risks, want %d", limit, len(got), rm.MaxPage)
		}
	}
}

func TestTheRiskListStillHonoursASmallerLimitAndTheDefault(t *testing.T) {
	m := newManager(t)
	fileRisks(t, m, rm.DefaultPage+5)
	small, _ := m.ListRisks(context.Background(), rm.RiskFilter{Limit: 7})
	if len(small) != 7 {
		t.Fatalf("limit 7 answered %d", len(small))
	}
	def, _ := m.ListRisks(context.Background(), rm.RiskFilter{})
	if len(def) != rm.DefaultPage {
		t.Fatalf("no limit answered %d, want %d", len(def), rm.DefaultPage)
	}
}

func TestTheRiskListPagesThroughEveryRiskByOffset(t *testing.T) {
	m := newManager(t)
	fileRisks(t, m, 7)
	seen := map[string]bool{}
	for offset := 0; offset < 8; offset += 3 {
		page, err := m.ListRisks(context.Background(), rm.RiskFilter{Limit: 3, Offset: offset})
		if err != nil {
			t.Fatalf("offset %d: %v", offset, err)
		}
		for _, r := range page {
			if seen[r.ID] {
				t.Fatalf("risk %s listed on two pages", r.ID)
			}
			seen[r.ID] = true
		}
	}
	if len(seen) != 7 {
		t.Fatalf("pages held %d distinct risks, want 7", len(seen))
	}
}

func TestTheRiskListPastTheEndIsEmptyAndANegativeOffsetStartsAtTheFirst(t *testing.T) {
	m := newManager(t)
	fileRisks(t, m, 4)
	for _, offset := range []int{4, 100000, math.MaxInt64} {
		got, err := m.ListRisks(context.Background(), rm.RiskFilter{Offset: offset})
		if err != nil || len(got) != 0 {
			t.Fatalf("offset %d = %d risks, %v; want none", offset, len(got), err)
		}
	}
	got, err := m.ListRisks(context.Background(), rm.RiskFilter{Offset: -3})
	if err != nil || len(got) != 4 {
		t.Fatalf("offset -3 = %d risks, %v; want 4", len(got), err)
	}
}

func TestTheGapListNeverAnswersMoreThanMaxPageWhateverTheLimit(t *testing.T) {
	m := newManager(t)
	fileRisks(t, m, rm.MaxPage+3)
	all, err := m.Gaps(context.Background(), rm.GapFilter{Limit: math.MaxInt64})
	if err != nil {
		t.Fatalf("Gaps: %v", err)
	}
	if len(all) != rm.MaxPage {
		t.Fatalf("answered %d gaps, want %d", len(all), rm.MaxPage)
	}
}

func TestABowtieStillHoldsEveryActionPastMaxPage(t *testing.T) {
	ctx := context.Background()
	m := newManager(t)
	risk, err := m.CreateRisk(ctx, rm.Risk{Title: "Busy risk", TopEvent: "Busy"})
	if err != nil {
		t.Fatalf("CreateRisk: %v", err)
	}
	for i := 0; i < rm.MaxPage+2; i++ {
		if _, err := m.CreateAction(ctx, rm.Action{RiskID: risk.ID, Title: fmt.Sprintf("Action %04d", i)}); err != nil {
			t.Fatalf("CreateAction: %v", err)
		}
	}
	b, err := m.Bowtie(ctx, risk.ID)
	if err != nil {
		t.Fatalf("Bowtie: %v", err)
	}
	if len(b.Actions) != rm.MaxPage+2 {
		t.Fatalf("bowtie holds %d actions, want %d", len(b.Actions), rm.MaxPage+2)
	}
}
