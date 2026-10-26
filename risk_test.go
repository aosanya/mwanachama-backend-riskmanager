package riskmanager_test

import (
	"strings"
	"testing"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
)

func TestANewRiskIsAlwaysADraftEvenWhenTheCallerSaysOpen(t *testing.T) {
	rm, ctx := newManager(t)
	r, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: "x", TopEvent: "y", Owner: "o", Status: riskmanager.StatusOpen})
	seedNoErr(t, err)
	if r.Status != riskmanager.StatusDraft {
		t.Fatalf("created as %s, want draft", r.Status)
	}
	held, err := rm.GetRisk(ctx, r.ID)
	seedNoErr(t, err)
	if held.Status != riskmanager.StatusDraft {
		t.Fatalf("stored as %s, want draft", held.Status)
	}
}

func TestACallerCannotChooseTheIDOfANewRisk(t *testing.T) {
	rm, ctx := newManager(t)
	r, err := rm.CreateRisk(ctx, riskmanager.Risk{ID: "mine", Title: "x", TopEvent: "y"})
	seedNoErr(t, err)
	if r.ID == "mine" || r.ID == "" {
		t.Fatalf("the caller's id survived: %q", r.ID)
	}
}

func TestARiskWithoutATitleOrATopEventIsRefused(t *testing.T) {
	rm, ctx := newManager(t)
	for name, r := range map[string]riskmanager.Risk{
		"no title":        {TopEvent: "y"},
		"blank title":     {Title: "   ", TopEvent: "y"},
		"no top event":    {Title: "x"},
		"blank top event": {Title: "x", TopEvent: "\t"},
		"nothing at all":  {},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := rm.CreateRisk(ctx, r)
			seedWantErr(t, err, riskmanager.ErrInvalid)
		})
	}
}

func TestADraftCannotOpenWithoutAnOwner(t *testing.T) {
	rm, ctx := newManager(t)
	r, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: "x", TopEvent: "y"})
	seedNoErr(t, err)
	_, err = rm.SetRiskStatus(ctx, r.ID, riskmanager.StatusOpen)
	seedWantErr(t, err, riskmanager.ErrInvalid)
	held, _ := rm.GetRisk(ctx, r.ID)
	if held.Status != riskmanager.StatusDraft {
		t.Fatalf("a refused transition still moved the risk to %s", held.Status)
	}
}

func TestAnOwnedDraftOpens(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedOpenRisk(t, rm, ctx, "x")
	if r.Status != riskmanager.StatusOpen {
		t.Fatalf("status = %s", r.Status)
	}
}

func TestAStatusThatIsNotOneOfTheDeclaredSetIsRefused(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	_, err := rm.SetRiskStatus(ctx, r.ID, "bogus")
	seedWantErr(t, err, riskmanager.ErrInvalid)
	_, err = rm.SetRiskStatus(ctx, r.ID, "")
	seedWantErr(t, err, riskmanager.ErrInvalid)
}

func TestARiskCannotBeAcceptedUntilItsResidualExposureIsRated(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedOpenRisk(t, rm, ctx, "x")
	_, err := rm.SetRiskStatus(ctx, r.ID, riskmanager.StatusAccepted)
	seedWantErr(t, err, riskmanager.ErrInvalid)

	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentInherent, 3, 3, "2026-01-01T00:00:00Z")
	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentTarget, 2, 2, "2026-01-01T00:00:00Z")
	_, err = rm.SetRiskStatus(ctx, r.ID, riskmanager.StatusAccepted)
	seedWantErr(t, err, riskmanager.ErrInvalid)

	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentResidual, 2, 3, "2026-01-02T00:00:00Z")
	accepted, err := rm.SetRiskStatus(ctx, r.ID, riskmanager.StatusAccepted)
	seedNoErr(t, err)
	if accepted.Status != riskmanager.StatusAccepted {
		t.Fatalf("status = %s", accepted.Status)
	}
}

func TestADraftCannotBeAcceptedDirectly(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentResidual, 2, 3, "2026-01-02T00:00:00Z")
	_, err := rm.SetRiskStatus(ctx, r.ID, riskmanager.StatusAccepted)
	seedWantErr(t, err, riskmanager.ErrInvalid)
}

func TestAnAcceptedRiskCanBeReopened(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedOpenRisk(t, rm, ctx, "x")
	seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentResidual, 2, 3, "2026-01-02T00:00:00Z")
	_, err := rm.SetRiskStatus(ctx, r.ID, riskmanager.StatusAccepted)
	seedNoErr(t, err)
	reopened, err := rm.SetRiskStatus(ctx, r.ID, riskmanager.StatusOpen)
	seedNoErr(t, err)
	if reopened.Status != riskmanager.StatusOpen {
		t.Fatalf("status = %s", reopened.Status)
	}
}

func TestRetiredIsTheEndOfTheRoadFromEveryStatus(t *testing.T) {
	for _, from := range []riskmanager.Status{riskmanager.StatusDraft, riskmanager.StatusOpen} {
		t.Run(string(from), func(t *testing.T) {
			rm, ctx := newManager(t)
			r := seedRisk(t, rm, ctx, "x")
			if from == riskmanager.StatusOpen {
				r = seedOpenRisk(t, rm, ctx, "x2")
			}
			seedRetire(t, rm, ctx, r.ID)
			for _, next := range []riskmanager.Status{
				riskmanager.StatusDraft, riskmanager.StatusOpen, riskmanager.StatusAccepted, riskmanager.StatusRetired,
			} {
				_, err := rm.SetRiskStatus(ctx, r.ID, next)
				seedWantErr(t, err, riskmanager.ErrInvalid)
			}
		})
	}
}

func TestOpenCannotGoBackToDraft(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedOpenRisk(t, rm, ctx, "x")
	_, err := rm.SetRiskStatus(ctx, r.ID, riskmanager.StatusDraft)
	seedWantErr(t, err, riskmanager.ErrInvalid)
}

func TestAnOpenRiskCannotBeUpdatedToAnEmptyOwner(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedOpenRisk(t, rm, ctx, "x")
	r.Owner = ""
	_, err := rm.UpdateRisk(ctx, r)
	seedWantErr(t, err, riskmanager.ErrInvalid)
	held, _ := rm.GetRisk(ctx, r.ID)
	if held.Owner != "owner-1" {
		t.Fatalf("a refused update still cleared the owner: %q", held.Owner)
	}
}

func TestADraftMayBeUpdatedToHaveNoOwner(t *testing.T) {
	rm, ctx := newManager(t)
	r := seedRisk(t, rm, ctx, "x")
	r.Owner = ""
	updated, err := rm.UpdateRisk(ctx, r)
	seedNoErr(t, err)
	if updated.Owner != "" {
		t.Fatalf("owner = %q", updated.Owner)
	}
}

func TestUpdatingARiskNeverChangesItsStatusItsParentOrItsCreationTime(t *testing.T) {
	rm, ctx := newManager(t)
	parent := seedOpenRisk(t, rm, ctx, "parent")
	pathway := seedThreat(t, rm, ctx, parent.ID, "t")
	other := seedThreat(t, rm, ctx, parent.ID, "u")
	child, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: "child", TopEvent: "c", Owner: "o", ParentPathwayID: pathway.ID})
	seedNoErr(t, err)

	updated, err := rm.UpdateRisk(ctx, riskmanager.Risk{
		ID: child.ID, Title: "renamed", TopEvent: "c2", Owner: "o",
		Status: riskmanager.StatusRetired, ParentPathwayID: other.ID, CreatedAt: "1999-01-01T00:00:00Z",
		Doc: map[string]any{"category": "data"},
	})
	seedNoErr(t, err)
	held, err := rm.GetRisk(ctx, child.ID)
	seedNoErr(t, err)
	for _, got := range []riskmanager.Risk{updated, held} {
		if got.Title != "renamed" || got.Doc["category"] != "data" {
			t.Fatalf("the update did not land: %+v", got)
		}
		if got.Status != riskmanager.StatusDraft {
			t.Fatalf("the body retired the risk: %s", got.Status)
		}
		if got.ParentPathwayID != pathway.ID {
			t.Fatalf("the drill-down was rewired to %q", got.ParentPathwayID)
		}
		if got.CreatedAt != child.CreatedAt {
			t.Fatalf("created_at moved from %s to %s", child.CreatedAt, got.CreatedAt)
		}
	}
}

func TestAChildRiskDrillsIntoAPathway(t *testing.T) {
	rm, ctx := newManager(t)
	parent := seedOpenRisk(t, rm, ctx, "parent")
	pathway := seedThreat(t, rm, ctx, parent.ID, "t")
	child, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: "child", TopEvent: "c", ParentPathwayID: pathway.ID})
	seedNoErr(t, err)
	if child.ParentPathwayID != pathway.ID {
		t.Fatalf("parent = %q", child.ParentPathwayID)
	}
	kids, err := rm.ListRisks(ctx, riskmanager.RiskFilter{ParentPathwayID: pathway.ID})
	seedNoErr(t, err)
	if len(kids) != 1 || kids[0].ID != child.ID {
		t.Fatalf("drill-downs of the pathway: %+v", kids)
	}
}

func TestAChildRiskNamingAMissingPathwayIsAnInvalidReference(t *testing.T) {
	rm, ctx := newManager(t)
	_, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: "child", TopEvent: "c", ParentPathwayID: "nowhere"})
	seedWantErr(t, err, riskmanager.ErrInvalidReference)
}

func TestAChildRiskCannotDrillIntoAPathwayOfARetiredRisk(t *testing.T) {
	rm, ctx := newManager(t)
	parent := seedOpenRisk(t, rm, ctx, "parent")
	pathway := seedThreat(t, rm, ctx, parent.ID, "t")
	seedRetire(t, rm, ctx, parent.ID)
	_, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: "child", TopEvent: "c", ParentPathwayID: pathway.ID})
	seedWantErr(t, err, riskmanager.ErrConflict)
}

func TestListingRisksFiltersByStatusOwnerAndWords(t *testing.T) {
	rm, ctx := newManager(t)
	a := seedOpenRisk(t, rm, ctx, "Data leak through assistants")
	b, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: "Vendor failure", TopEvent: "A supplier stops", Hazard: "Concentrated DATA custody", Owner: "owner-9"})
	seedNoErr(t, err)

	ids := func(f riskmanager.RiskFilter) map[string]bool {
		got, err := rm.ListRisks(ctx, f)
		seedNoErr(t, err)
		out := map[string]bool{}
		for _, r := range got {
			out[r.ID] = true
		}
		return out
	}
	if got := ids(riskmanager.RiskFilter{}); len(got) != 2 {
		t.Fatalf("unfiltered: %v", got)
	}
	if got := ids(riskmanager.RiskFilter{Status: riskmanager.StatusOpen}); len(got) != 1 || !got[a.ID] {
		t.Fatalf("open only: %v", got)
	}
	if got := ids(riskmanager.RiskFilter{Owner: "owner-9"}); len(got) != 1 || !got[b.ID] {
		t.Fatalf("owner-9: %v", got)
	}
	if got := ids(riskmanager.RiskFilter{Query: "dAtA"}); len(got) != 2 {
		t.Fatalf("a word in a title and in a hazard, any case: %v", got)
	}
	if got := ids(riskmanager.RiskFilter{Query: "supplier"}); len(got) != 1 || !got[b.ID] {
		t.Fatalf("a word in a top event: %v", got)
	}
	if got := ids(riskmanager.RiskFilter{Query: "no such words"}); len(got) != 0 {
		t.Fatalf("nothing should match: %v", got)
	}
	if got := ids(riskmanager.RiskFilter{Limit: 1}); len(got) != 1 {
		t.Fatalf("limit 1: %v", got)
	}
}

func TestAWordSearchTreatsPercentAndUnderscoreLiterally(t *testing.T) {
	rm, ctx := newManager(t)
	pct := seedRisk(t, rm, ctx, "50% of records")
	under := seedRisk(t, rm, ctx, "audit_log exposure")
	seedRisk(t, rm, ctx, "plain risk")

	only := func(q string, want string) {
		t.Helper()
		got, err := rm.ListRisks(ctx, riskmanager.RiskFilter{Query: q})
		seedNoErr(t, err)
		if len(got) != 1 || got[0].ID != want {
			t.Fatalf("query %q matched %d risks: %+v", q, len(got), got)
		}
	}
	only("%", pct.ID)
	only("_", under.ID)
	only("50%", pct.ID)
	only("t_l", under.ID)
	if got, _ := rm.ListRisks(ctx, riskmanager.RiskFilter{Query: strings.Repeat("%", 3)}); len(got) != 0 {
		t.Fatalf("a run of wildcards matched %d risks", len(got))
	}
}

func TestAMissingRiskIsNotFound(t *testing.T) {
	rm, ctx := newManager(t)
	_, err := rm.GetRisk(ctx, "nobody")
	seedWantErr(t, err, riskmanager.ErrRiskNotFound)
	_, err = rm.UpdateRisk(ctx, riskmanager.Risk{ID: "nobody", Title: "x", TopEvent: "y"})
	seedWantErr(t, err, riskmanager.ErrRiskNotFound)
	_, err = rm.SetRiskStatus(ctx, "nobody", riskmanager.StatusOpen)
	seedWantErr(t, err, riskmanager.ErrRiskNotFound)
}
