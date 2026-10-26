package riskmanager_test

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-riskmanager/models"
)

type seedRegister struct {
	low, high, mixed riskmanager.Risk
}

func seedMixedRegister(t *testing.T, rm riskmanager.RiskManager, ctx context.Context) seedRegister {
	t.Helper()
	low := seedCleanRisk(t, rm, ctx, "low only")
	seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, low.ID, "extra route").ID, seedControl(t, rm, ctx, "extra guard", "the team").ID)

	high := seedRisk(t, rm, ctx, "bare")
	seedThreat(t, rm, ctx, high.ID, "open route")

	mixed := seedRisk(t, rm, ctx, "mixed")
	weak := seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, mixed.ID, "weak route").ID, seedControl(t, rm, ctx, "weak guard", "the team").ID)
	seedEvidence(t, rm, ctx, weak.ID, riskmanager.OutcomeFail, "2026-01-01T00:00:00Z")
	seedThreat(t, rm, ctx, mixed.ID, "also open")
	return seedRegister{low: low, high: high, mixed: mixed}
}

func seedSeverities(gaps []riskmanager.Gap) []riskmanager.Severity {
	out := make([]riskmanager.Severity, len(gaps))
	for i, g := range gaps {
		out[i] = g.Severity
	}
	return out
}

func seedRepeated(s riskmanager.Severity, n int) []riskmanager.Severity {
	out := make([]riskmanager.Severity, n)
	for i := range out {
		out[i] = s
	}
	return out
}

func seedGapKeys(gaps []riskmanager.Gap) []string {
	out := make([]string, len(gaps))
	for i, g := range gaps {
		out[i] = string(g.Kind) + "|" + g.SubjectID + "|" + g.Message
	}
	return out
}

func seedKindsInOrder(gaps []riskmanager.Gap) []riskmanager.GapKind {
	out := make([]riskmanager.GapKind, len(gaps))
	for i, g := range gaps {
		out[i] = g.Kind
	}
	return out
}

func seedOfSeverity(gaps []riskmanager.Gap, s riskmanager.Severity) []riskmanager.Gap {
	var out []riskmanager.Gap
	for _, g := range gaps {
		if g.Severity == s {
			out = append(out, g)
		}
	}
	return out
}

func seedKindsOfSubject(gaps []riskmanager.Gap, subjectID string) map[riskmanager.GapKind]int {
	out := map[riskmanager.GapKind]int{}
	for _, g := range gaps {
		if g.SubjectID == subjectID {
			out[g.Kind]++
		}
	}
	return out
}

func TestGapsOfOneRiskComeHighThenMediumThenLowKeepingTheOrderTheyWereFoundIn(t *testing.T) {
	rm, ctx := newManager(t)
	reg := seedMixedRegister(t, rm, ctx)
	gaps := seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: reg.mixed.ID})

	want := slices.Concat(seedRepeated(models.SeverityHigh, 2), seedRepeated(models.SeverityMedium, 3), seedRepeated(models.SeverityLow, 2))
	if got := seedSeverities(gaps); !slices.Equal(got, want) {
		t.Fatalf("severities in order = %v, want %v", got, want)
	}
	if got, want := seedKindsInOrder(seedOfSeverity(gaps, models.SeverityHigh)), []riskmanager.GapKind{models.GapBarrierFailing, models.GapPathwayUnguarded}; !slices.Equal(got, want) {
		t.Fatalf("high gaps = %v, want %v", got, want)
	}
	if got, want := seedKindsInOrder(seedOfSeverity(gaps, models.SeverityMedium)), []riskmanager.GapKind{models.GapRiskOneSided, models.GapRiskUnrated, models.GapRiskUnrated}; !slices.Equal(got, want) {
		t.Fatalf("medium gaps = %v, want %v", got, want)
	}
	if got, want := seedKindsInOrder(seedOfSeverity(gaps, models.SeverityLow)), []riskmanager.GapKind{models.GapBarrierUnassessed, models.GapBarrierUntested}; !slices.Equal(got, want) {
		t.Fatalf("low gaps = %v, want %v", got, want)
	}
}

func TestABowtieRanksItsOwnGapsWithoutTheManagersHelp(t *testing.T) {
	rm, ctx := newManager(t)
	reg := seedMixedRegister(t, rm, ctx)
	bt, err := rm.Bowtie(ctx, reg.mixed.ID)
	seedNoErr(t, err)

	want := slices.Concat(seedRepeated(models.SeverityHigh, 2), seedRepeated(models.SeverityMedium, 3), seedRepeated(models.SeverityLow, 2))
	if got := seedSeverities(bt.Gaps(time.Now())); !slices.Equal(got, want) {
		t.Fatalf("a bow-tie's own gaps in order = %v, want %v", got, want)
	}
}

func TestGapsAcrossTheRegisterAreRankedTogetherAndNotRiskByRisk(t *testing.T) {
	rm, ctx := newManager(t)
	reg := seedMixedRegister(t, rm, ctx)
	gaps := seedGapsOf(t, rm, ctx, riskmanager.GapFilter{})

	want := slices.Concat(seedRepeated(models.SeverityHigh, 3), seedRepeated(models.SeverityMedium, 6), seedRepeated(models.SeverityLow, 4))
	if got := seedSeverities(gaps); !slices.Equal(got, want) {
		t.Fatalf("severities in order = %v, want %v", got, want)
	}
	if gaps[0].RiskID != reg.high.ID {
		t.Fatalf("the first gap belongs to %s; the oldest risk has only low gaps, so the first should be the bare risk's", gaps[0].RiskID)
	}

	highs := seedOfSeverity(gaps, models.SeverityHigh)
	wantHighs := []struct {
		risk string
		kind riskmanager.GapKind
	}{
		{reg.high.ID, models.GapPathwayUnguarded}, {reg.mixed.ID, models.GapBarrierFailing}, {reg.mixed.ID, models.GapPathwayUnguarded},
	}
	for i, w := range wantHighs {
		if highs[i].RiskID != w.risk || highs[i].Kind != w.kind {
			t.Fatalf("high gap %d = %s of %s, want %s of %s", i, highs[i].Kind, highs[i].RiskID, w.kind, w.risk)
		}
	}

	lows := seedOfSeverity(gaps, models.SeverityLow)
	for i, risk := range []string{reg.low.ID, reg.low.ID, reg.mixed.ID, reg.mixed.ID} {
		if lows[i].RiskID != risk {
			t.Fatalf("low gap %d belongs to %s, want %s: equal severities should stay in the order the risks were created", i, lows[i].RiskID, risk)
		}
	}
}

func TestTheSeverityFilterKeepsOnlyThatSeverityAndTheThreeTogetherAreTheWholeList(t *testing.T) {
	rm, ctx := newManager(t)
	seedMixedRegister(t, rm, ctx)
	all := seedGapsOf(t, rm, ctx, riskmanager.GapFilter{})

	total := 0
	for s, want := range map[riskmanager.Severity]int{models.SeverityHigh: 3, models.SeverityMedium: 6, models.SeverityLow: 4} {
		got := seedGapsOf(t, rm, ctx, riskmanager.GapFilter{Severity: s})
		if len(got) != want {
			t.Fatalf("%s: %d gaps, want %d", s, len(got), want)
		}
		for _, g := range got {
			if g.Severity != s {
				t.Fatalf("a %s filter returned a %s gap: %+v", s, g.Severity, g)
			}
		}
		total += len(got)
	}
	if total != len(all) {
		t.Fatalf("the three severities hold %d gaps between them, the whole list holds %d", total, len(all))
	}
}

func TestAGapCarriesItsSeverityInTheAnswerItSerialises(t *testing.T) {
	rm, ctx := newManager(t)
	reg := seedMixedRegister(t, rm, ctx)
	gaps := seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: reg.high.ID, Kind: models.GapPathwayUnguarded})
	if len(gaps) != 1 {
		t.Fatalf("gaps: %+v", gaps)
	}
	raw, err := json.Marshal(gaps[0])
	seedNoErr(t, err)
	var decoded map[string]any
	seedNoErr(t, json.Unmarshal(raw, &decoded))
	if decoded["severity"] != "high" || decoded["kind"] != "pathway_unguarded" {
		t.Fatalf("serialised gap: %s", raw)
	}
	for _, g := range seedGapsOf(t, rm, ctx, riskmanager.GapFilter{}) {
		if g.Severity != g.Kind.Severity() {
			t.Fatalf("%s carries severity %s, want %s", g.Kind, g.Severity, g.Kind.Severity())
		}
	}
}
