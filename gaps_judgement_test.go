package riskmanager_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-riskmanager/models"
)

func TestAJudgementOfPartialOrInadequateIsADesignAndACoverageGapAndNotAnUnassessedOne(t *testing.T) {
	for _, j := range []riskmanager.Judgement{riskmanager.JudgementPartial, riskmanager.JudgementInadequate} {
		t.Run(string(j), func(t *testing.T) {
			rm, ctx := newManager(t)
			r := seedRisk(t, rm, ctx, "x")
			b := seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r.ID, "t").ID, seedControl(t, rm, ctx, "c", "o").ID)
			_, err := rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: b.ID, Design: &j, Coverage: &j})
			seedNoErr(t, err)
			gaps := seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r.ID})
			if !seedSubjects(gaps, models.GapBarrierDesign)[b.ID] || !seedSubjects(gaps, models.GapBarrierCoverage)[b.ID] {
				t.Fatalf("a %s barrier is not reported on design and coverage: %+v", j, gaps)
			}
			if seedSubjects(gaps, models.GapBarrierUnassessed)[b.ID] {
				t.Fatalf("a %s barrier has been assessed and is still reported unassessed", j)
			}
		})
	}
}

func TestABarrierWithPartialDesignAndAdequateCoverageRaisesDesignButNotUnassessed(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	b := seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r.ID, "t").ID, seedControl(t, rm, ctx, "c", "o").ID)
	_, err := rm.UpdateBarrier(ctx, riskmanager.BarrierChange{
		ID: b.ID, Design: seedPtr(riskmanager.JudgementPartial), Coverage: seedPtr(riskmanager.JudgementAdequate),
	})
	seedNoErr(t, err)
	kinds := seedKindsOfSubject(seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r.ID}), b.ID)
	if kinds[models.GapBarrierDesign] != 1 || kinds[models.GapBarrierUnassessed] != 0 || kinds[models.GapBarrierCoverage] != 0 {
		t.Fatalf("kinds raised on the barrier: %v", kinds)
	}
}

func TestAFreshBarrierRaisesExactlyOneUnassessedGapAndOneUntestedGapAndNamesWhatIsMissing(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	b := seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r.ID, "t").ID, seedControl(t, rm, ctx, "c", "o").ID)
	assess := func() (map[riskmanager.GapKind]int, string) {
		gaps := seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r.ID, Kind: models.GapBarrierUnassessed})
		var message string
		if len(gaps) == 1 {
			message = gaps[0].Message
		}
		return seedKindsOfSubject(seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r.ID}), b.ID), message
	}

	kinds, message := assess()
	if len(kinds) != 2 || kinds[models.GapBarrierUnassessed] != 1 || kinds[models.GapBarrierUntested] != 1 {
		t.Fatalf("a fresh barrier raised %v, want exactly one unassessed and one untested", kinds)
	}
	if !strings.Contains(message, "design and coverage") {
		t.Fatalf("the unassessed gap does not name both judgements: %q", message)
	}

	_, err := rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: b.ID, Design: seedPtr(riskmanager.JudgementAdequate)})
	seedNoErr(t, err)
	kinds, message = assess()
	if kinds[models.GapBarrierUnassessed] != 1 || !strings.Contains(message, "its coverage assessed") || strings.Contains(message, "design") {
		t.Fatalf("with design done the gap should name only coverage: %v %q", kinds, message)
	}

	_, err = rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: b.ID, Coverage: seedPtr(riskmanager.JudgementAdequate)})
	seedNoErr(t, err)
	kinds, _ = assess()
	if kinds[models.GapBarrierUnassessed] != 0 || kinds[models.GapBarrierDesign] != 0 || kinds[models.GapBarrierCoverage] != 0 {
		t.Fatalf("an adequate barrier still raises judgement gaps: %v", kinds)
	}
	if kinds[models.GapBarrierUntested] != 1 {
		t.Fatalf("assessing a barrier does not test it, yet untested = %d", kinds[models.GapBarrierUntested])
	}
}

func TestOnlyAnAdequateJudgementClearsTheDesignCoverageAndUnassessedGaps(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	b := seedBarrier(t, rm, ctx, seedThreat(t, rm, ctx, r.ID, "t").ID, seedControl(t, rm, ctx, "c", "o").ID)
	adequate := riskmanager.JudgementAdequate
	_, err := rm.UpdateBarrier(ctx, riskmanager.BarrierChange{ID: b.ID, Design: &adequate, Coverage: &adequate})
	seedNoErr(t, err)
	kinds := seedKindsOfSubject(seedGapsOf(t, rm, ctx, riskmanager.GapFilter{RiskID: r.ID}), b.ID)
	if kinds[models.GapBarrierDesign]+kinds[models.GapBarrierCoverage]+kinds[models.GapBarrierUnassessed] != 0 {
		t.Fatalf("adequate design and coverage are still reported: %v", kinds)
	}
}

func TestEveryGapKindHasTheSeverityTheModuleDeclaresForIt(t *testing.T) {
	declared := map[models.GapKind]models.Severity{
		models.GapPathwayUnguarded: models.SeverityHigh, models.GapBarrierFailing: models.SeverityHigh, models.GapActionOverdue: models.SeverityHigh,
		models.GapRiskUnowned: models.SeverityMedium, models.GapRiskOneSided: models.SeverityMedium, models.GapRiskUnrated: models.SeverityMedium,
		models.GapControlUnowned: models.SeverityMedium, models.GapActionUndated: models.SeverityMedium,
		models.GapBarrierDesign: models.SeverityMedium, models.GapBarrierCoverage: models.SeverityMedium,
		models.GapBarrierUnassessed: models.SeverityLow, models.GapBarrierUntested: models.SeverityLow,
	}
	for _, kind := range seedGapKindConstants(t) {
		want, ok := declared[kind]
		if !ok {
			t.Errorf("models/gap.go declares the gap kind %q and this test's table does not: give it a severity in both places", kind)
			continue
		}
		if got := kind.Severity(); got != want || !got.Known() {
			t.Errorf("%s has severity %q, want %q", kind, got, want)
		}
	}
	if found := len(seedGapKindConstants(t)); found != len(declared) {
		t.Errorf("models/gap.go declares %d gap kinds and the table holds %d", found, len(declared))
	}
}

func seedGapKindConstants(t *testing.T) []models.GapKind {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "models/gap.go", nil, 0)
	if err != nil {
		t.Fatalf("parse models/gap.go: %v", err)
	}
	var kinds []models.GapKind
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			vs := spec.(*ast.ValueSpec)
			if id, ok := vs.Type.(*ast.Ident); !ok || id.Name != "GapKind" {
				continue
			}
			for _, v := range vs.Values {
				lit, ok := v.(*ast.BasicLit)
				if !ok {
					t.Fatalf("a GapKind constant is not a string literal: %v", v)
				}
				value, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("unquote %s: %v", lit.Value, err)
				}
				kinds = append(kinds, models.GapKind(value))
			}
		}
	}
	if len(kinds) == 0 {
		t.Fatal("found no GapKind constants in models/gap.go")
	}
	return kinds
}

func TestAKindNobodyDeclaredHasNoSeverityAndIsNotKnown(t *testing.T) {
	if got := models.GapKind("made_up").Severity(); got != "" {
		t.Fatalf("an undeclared kind has severity %q", got)
	}
	if models.Severity("").Known() || models.Severity("urgent").Known() {
		t.Fatal("an undeclared severity is known")
	}
}
