package riskmanager_test

import (
	"path/filepath"
	"testing"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-riskmanager/models"
	"github.com/aosanya/mwanachama-backend-riskmanager/routes"
	"github.com/aosanya/mwanachama-backend-shared/dispatch"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func shippedExamples(t *testing.T) []string {
	t.Helper()
	found, err := filepath.Glob(filepath.Join(".", "spec", "examples", "*.riskmanager.json"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(found) < 2 {
		t.Fatalf("the module ships %d domain specs, and domain-neutrality is only exercised by at least two", len(found))
	}
	return append([]string{"riskmanager.wakala.json"}, found...)
}

func TestEveryExampleFitsTheTypes(t *testing.T) {
	for _, path := range shippedExamples(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			s, err := riskmanager.LoadSpec(path)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			db := newDB(t)
			if err := spec.Migrate(db, s); err != nil {
				t.Fatalf("migrate: %v", err)
			}
			if _, err := riskmanager.NewRiskManager(db, s); err != nil {
				t.Fatalf("the spec and the Go types disagree: %v", err)
			}
		})
	}
}

func TestTwoDomainsCoexist(t *testing.T) {
	db := newDB(t)
	for _, path := range shippedExamples(t) {
		s, err := riskmanager.LoadSpec(path)
		if err != nil {
			t.Fatalf("load %s: %v", path, err)
		}
		if err := spec.Migrate(db, s); err != nil {
			t.Fatalf("migrate %s into a database another domain already occupies: %v", path, err)
		}
	}

	ai, err := riskmanager.LoadSpec(examplePath("ai_safety"))
	if err != nil {
		t.Fatalf("load ai_safety: %v", err)
	}
	food, err := riskmanager.LoadSpec(examplePath("food_safety"))
	if err != nil {
		t.Fatalf("load food_safety: %v", err)
	}

	aiRisk, _ := ai.ByRole(riskmanager.RoleRisk)
	foodRisk, _ := food.ByRole(riskmanager.RoleRisk)
	if ai.TableFor(aiRisk) == food.TableFor(foodRisk) {
		t.Fatalf("two domains landed their risk in one table: %s", ai.TableFor(aiRisk))
	}

	aiManager, err := riskmanager.NewRiskManager(db, ai)
	if err != nil {
		t.Fatalf("ai manager: %v", err)
	}
	foodManager, err := riskmanager.NewRiskManager(db, food)
	if err != nil {
		t.Fatalf("food manager: %v", err)
	}

	ctx := t.Context()
	if _, err := aiManager.CreateRisk(ctx, riskmanager.Risk{Title: "Data leaves through a prompt", TopEvent: "Sensitive data reaches an unapproved service"}); err != nil {
		t.Fatalf("ai write: %v", err)
	}
	if _, err := foodManager.CreateRisk(ctx, riskmanager.Risk{Title: "Unsafe batch reaches a buyer", TopEvent: "A contaminated batch is delivered"}); err != nil {
		t.Fatalf("food write: %v", err)
	}

	held, err := foodManager.ListRisks(ctx, riskmanager.RiskFilter{})
	if err != nil {
		t.Fatalf("food read: %v", err)
	}
	if len(held) != 1 || held[0].Title != "Unsafe batch reaches a buyer" {
		t.Fatalf("food sees %d risks, and one domain is reading another's table: %+v", len(held), held)
	}
}

func TestRequiredFieldsHaveNoDefault(t *testing.T) {
	b, err := riskmanager.Blueprint()
	if err != nil {
		t.Fatalf("blueprint: %v", err)
	}
	for _, o := range b.Objects {
		for _, f := range o.Fields {
			if f.Required && f.Default != "" {
				t.Errorf("%s.%s is required and defaulted to %q — the default is exactly what lets an omitted value pass unnoticed",
					o.Role, f.Name, f.Default)
			}
		}
	}

	for _, path := range shippedExamples(t) {
		s, err := riskmanager.LoadSpec(path)
		if err != nil {
			t.Fatalf("load %s: %v", path, err)
		}
		for _, o := range s.Objects {
			for _, f := range o.Fields {
				if f.Required && f.Default != "" {
					t.Errorf("%s: %s.%s is required and defaulted to %q", filepath.Base(path), o.Name, f.Name, f.Default)
				}
			}
		}
	}
}

func heldVocabularies() map[string][]string {
	return map[string][]string{
		"risk.status": {
			string(models.StatusDraft), string(models.StatusOpen),
			string(models.StatusAccepted), string(models.StatusRetired),
		},
		"pathway.side": {string(models.SideThreat), string(models.SideConsequence)},
		"barrier.design": {
			string(models.JudgementUnassessed), string(models.JudgementAdequate),
			string(models.JudgementPartial), string(models.JudgementInadequate),
		},
		"barrier.coverage": {
			string(models.JudgementUnassessed), string(models.JudgementAdequate),
			string(models.JudgementPartial), string(models.JudgementInadequate),
		},
		"barrier.maturity": {
			string(models.MaturityProposed), string(models.MaturityDesigned),
			string(models.MaturityDeployed), string(models.MaturityTested),
			string(models.MaturityMonitored),
		},
		"evidence.kind": {
			string(models.EvidenceTest), string(models.EvidenceLog), string(models.EvidenceAudit),
			string(models.EvidenceConfiguration), string(models.EvidenceIncident),
		},
		"evidence.outcome": {
			string(models.OutcomePass), string(models.OutcomePartial), string(models.OutcomeFail),
		},
		"assessment.kind": {
			string(models.AssessmentInherent), string(models.AssessmentResidual), string(models.AssessmentTarget),
		},
		"action.status": {
			string(models.ActionOpen), string(models.ActionInProgress),
			string(models.ActionDone), string(models.ActionDropped),
		},
	}
}

func TestVocabularyMatchesTheBlueprint(t *testing.T) {
	b, err := riskmanager.Blueprint()
	if err != nil {
		t.Fatalf("blueprint: %v", err)
	}

	held := heldVocabularies()
	declared := map[string][]string{}
	for _, o := range b.Objects {
		for _, f := range o.Fields {
			if len(f.Values) > 0 {
				declared[o.Role+"."+f.Name] = f.Values
			}
		}
	}

	for at, values := range declared {
		if _, ok := held[at]; !ok {
			t.Errorf("the blueprint declares values for %s and models/vocabulary.go holds none", at)
			continue
		}
		for _, v := range values {
			if !holdsValue(held[at], v) {
				t.Errorf("the blueprint declares %s = %q and Go holds no constant for it", at, v)
			}
		}
	}
	for at, values := range held {
		if _, ok := declared[at]; !ok {
			t.Errorf("models/vocabulary.go holds constants for %s and the blueprint declares no values there", at)
			continue
		}
		for _, v := range values {
			if !holdsValue(declared[at], v) {
				t.Errorf("Go holds %s = %q and the blueprint does not declare it", at, v)
			}
		}
	}
}

func TestEverySentinelIsMappedToAStatus(t *testing.T) {
	s, err := dispatch.Parse(riskmanager.Operations())
	if err != nil {
		t.Fatalf("parse operations: %v", err)
	}
	for _, problem := range dispatch.UnmappedSentinels(s, routes.Sentinels, ".") {
		t.Error(problem)
	}
}

func holdsValue(values []string, s string) bool {
	for _, v := range values {
		if v == s {
			return true
		}
	}
	return false
}
