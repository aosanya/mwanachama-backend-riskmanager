package riskmanager_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/identity"
	"github.com/aosanya/mwanachama-backend-shared/spec"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
)

type directory map[string]string

func (d directory) StableIdentity(context.Context, string, string) (string, error) {
	return "", identity.ErrNotFound
}

func (d directory) Resolve(_ context.Context, raw string) (string, string, error) {
	if _, err := identity.Parse(raw); err != nil {
		return "", "", err
	}
	object, ok := d[raw]
	if !ok {
		return "", "", fmt.Errorf("%s: %w", raw, identity.ErrNotFound)
	}
	return object, raw, nil
}

const (
	ann   = "actor/hub/ann"
	bea   = "actor/hub/bea"
	ghost = "actor/hub/ghost"
	team  = "actor/hub/ops"
)

func managerWithOwners(t *testing.T, owners identity.Owner, objects ...string) (riskmanager.RiskManager, context.Context) {
	t.Helper()
	db := newDB(t)
	s, err := riskmanager.LoadSpec(filepath.Join(".", "spec", "examples", "ai_safety.riskmanager.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatal(err)
	}
	rm, err := riskmanager.NewRiskManager(db, s, riskmanager.WithOwners(owners, objects...))
	if err != nil {
		t.Fatal(err)
	}
	return rm, context.Background()
}

func people() directory {
	return directory{ann: "actor", bea: "actor", team: "group"}
}

func TestARiskControlOrActionOwnerThatTheActorRegisterDoesNotKnowIsRefused(t *testing.T) {
	rm, ctx := managerWithOwners(t, people())

	_, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: "r", TopEvent: "e", Owner: ghost})
	seedWantErr(t, err, riskmanager.ErrInvalidReference)
	_, err = rm.CreateControl(ctx, riskmanager.Control{Name: "c", Owner: ghost})
	seedWantErr(t, err, riskmanager.ErrInvalidReference)

	r, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: "r", TopEvent: "e", Owner: ann})
	seedNoErr(t, err)
	c, err := rm.CreateControl(ctx, riskmanager.Control{Name: "c", Owner: ann})
	seedNoErr(t, err)
	_, err = rm.CreateAction(ctx, riskmanager.Action{RiskID: r.ID, Title: "w", Owner: ghost})
	seedWantErr(t, err, riskmanager.ErrInvalidReference)
	a, err := rm.CreateAction(ctx, riskmanager.Action{RiskID: r.ID, Title: "w", Owner: bea})
	seedNoErr(t, err)

	r.Owner = ghost
	_, err = rm.UpdateRisk(ctx, r)
	seedWantErr(t, err, riskmanager.ErrInvalidReference)
	c.Owner = ghost
	_, err = rm.UpdateControl(ctx, c)
	seedWantErr(t, err, riskmanager.ErrInvalidReference)
	a.Owner = ghost
	_, err = rm.UpdateAction(ctx, a)
	seedWantErr(t, err, riskmanager.ErrInvalidReference)

	r.Owner = bea
	moved, err := rm.UpdateRisk(ctx, r)
	seedNoErr(t, err)
	if moved.Owner != bea {
		t.Fatalf("owner %q", moved.Owner)
	}
}

func TestAnOwnerThatIsNotARawIsRefusedAndNoOwnerIsStillAGapNotARefusal(t *testing.T) {
	rm, ctx := managerWithOwners(t, people())
	_, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: "r", TopEvent: "e", Owner: "ann"})
	seedWantErr(t, err, riskmanager.ErrInvalidReference)
	if _, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: "r", TopEvent: "e"}); err != nil {
		t.Fatalf("a draft with no owner: %v", err)
	}
}

func TestOnlyTheDeclaredObjectsCanOwn(t *testing.T) {
	rm, ctx := managerWithOwners(t, people(), "actor")
	if _, err := rm.CreateControl(ctx, riskmanager.Control{Name: "by actor", Owner: ann}); err != nil {
		t.Fatalf("an actor: %v", err)
	}
	_, err := rm.CreateControl(ctx, riskmanager.Control{Name: "by group", Owner: team})
	seedWantErr(t, err, riskmanager.ErrInvalidReference)
}

func TestWithoutAnOwnerDirectoryAnyOwnerStringIsKept(t *testing.T) {
	rm, ctx := newManager(t)
	if _, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: "r", TopEvent: "e", Owner: "whoever"}); err != nil {
		t.Fatal(err)
	}
}

func TestEverythingOnePersonOwnsIsAQueryOnRisksControlsAndActions(t *testing.T) {
	rm, ctx := managerWithOwners(t, people())
	r, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: "mine", TopEvent: "e", Owner: ann})
	seedNoErr(t, err)
	_, err = rm.CreateRisk(ctx, riskmanager.Risk{Title: "hers", TopEvent: "e", Owner: bea})
	seedNoErr(t, err)
	_, err = rm.CreateControl(ctx, riskmanager.Control{Name: "mine", Owner: ann})
	seedNoErr(t, err)
	_, err = rm.CreateControl(ctx, riskmanager.Control{Name: "hers", Owner: bea})
	seedNoErr(t, err)
	_, err = rm.CreateAction(ctx, riskmanager.Action{RiskID: r.ID, Title: "mine", Owner: ann})
	seedNoErr(t, err)
	_, err = rm.CreateAction(ctx, riskmanager.Action{RiskID: r.ID, Title: "hers", Owner: bea})
	seedNoErr(t, err)

	risks, err := rm.ListRisks(ctx, riskmanager.RiskFilter{Owner: ann})
	seedNoErr(t, err)
	controls, err := rm.ListControls(ctx, riskmanager.ControlFilter{Owner: ann})
	seedNoErr(t, err)
	actions, err := rm.ListActions(ctx, riskmanager.ActionFilter{Owner: ann})
	seedNoErr(t, err)
	if len(risks) != 1 || len(controls) != 1 || len(actions) != 1 || actions[0].Title != "mine" {
		t.Fatalf("what ann owns: %d risks, %d controls, %d actions", len(risks), len(controls), len(actions))
	}
}

func TestAnUnchangedOwnerIsNotCheckedAgainSoARecordKeptBeforeTheDirectoryCanStillBeEdited(t *testing.T) {
	db := newDB(t)
	s, err := riskmanager.LoadSpec(examplePath("ai_safety"))
	seedNoErr(t, err)
	seedNoErr(t, spec.Migrate(db, s))
	ctx := context.Background()
	plain, err := riskmanager.NewRiskManager(db, s)
	seedNoErr(t, err)
	r, err := plain.CreateRisk(ctx, riskmanager.Risk{Title: "r", TopEvent: "e", Owner: "legacy name"})
	seedNoErr(t, err)

	checked, err := riskmanager.NewRiskManager(db, s, riskmanager.WithOwners(people()))
	seedNoErr(t, err)
	r.Title = "renamed"
	if _, err := checked.UpdateRisk(ctx, r); err != nil {
		t.Fatalf("editing the title with the owner unchanged: %v", err)
	}
	r.Owner = "another legacy name"
	_, err = checked.UpdateRisk(ctx, r)
	seedWantErr(t, err, riskmanager.ErrInvalidReference)
}
