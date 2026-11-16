package riskmanager_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	rm "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-shared/identity"
)

type fakeOwners map[string]bool

func (f fakeOwners) StableIdentity(ctx context.Context, object, id string) (string, error) {
	return id, nil
}

func (f fakeOwners) Resolve(ctx context.Context, raw string) (string, string, error) {
	if !f[raw] {
		return "", "", identity.ErrNotFound
	}
	return "actor", raw, nil
}

func newManager(t *testing.T, opts ...rm.Option) rm.RiskManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := rm.SpecForMount("test", "risks")
	if err != nil {
		t.Fatalf("SpecForMount: %v", err)
	}
	if err := rm.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	m, err := rm.NewRiskManager(db, s, opts...)
	if err != nil {
		t.Fatalf("NewRiskManager: %v", err)
	}
	return m
}

func TestARiskReadsBackAsDraft(t *testing.T) {
	ctx := context.Background()
	m := newManager(t)
	r, err := m.CreateRisk(ctx, rm.Risk{Title: "Flooded depot", TopEvent: "Depot floods"})
	if err != nil {
		t.Fatalf("CreateRisk: %v", err)
	}
	got, err := m.GetRisk(ctx, r.ID)
	if err != nil || got.Status != "draft" || got.Title != "Flooded depot" {
		t.Fatalf("GetRisk = %+v, %v", got, err)
	}
}

func TestAnUnknownRiskIsNotFound(t *testing.T) {
	if _, err := newManager(t).GetRisk(context.Background(), "nope"); !errors.Is(err, rm.ErrRiskNotFound) {
		t.Fatalf("err = %v, want ErrRiskNotFound", err)
	}
}

func TestAnUnknownOwnerIsAnInvalidReference(t *testing.T) {
	ctx := context.Background()
	m := newManager(t, rm.WithOwners(fakeOwners{"a-1": true}, "actor"))
	if _, err := m.CreateRisk(ctx, rm.Risk{Title: "T", TopEvent: "E", Owner: "ghost"}); !errors.Is(err, rm.ErrInvalidReference) {
		t.Fatalf("err = %v, want ErrInvalidReference", err)
	}
	if _, err := m.CreateRisk(ctx, rm.Risk{Title: "T", TopEvent: "E", Owner: "a-1"}); err != nil {
		t.Fatalf("known owner: %v", err)
	}
}

func TestOperationsAreDeclared(t *testing.T) {
	if len(rm.Operations()) == 0 {
		t.Fatal("no operations")
	}
}

func TestARiskIsAddedWithCausesConsequencesSafeguardsAndOwner(t *testing.T) {
	ctx := context.Background()
	m := newManager(t, rm.WithOwners(fakeOwners{"a-1": true}, "actor"))
	risk, err := m.CreateRisk(ctx, rm.Risk{Title: "Water tower freezes", TopEvent: "Water tower freezes in winter", Owner: "a-1"})
	if err != nil {
		t.Fatalf("CreateRisk: %v", err)
	}
	cause, err := m.CreatePathway(ctx, rm.Pathway{RiskID: risk.ID, Side: rm.SideThreat, Title: "Long frost"})
	if err != nil {
		t.Fatalf("cause: %v", err)
	}
	consequence, err := m.CreatePathway(ctx, rm.Pathway{RiskID: risk.ID, Side: rm.SideConsequence, Title: "No water for 400 homes"})
	if err != nil {
		t.Fatalf("consequence: %v", err)
	}
	for _, name := range []string{"Backup heating cable", "Mobile water tanks"} {
		control, err := m.CreateControl(ctx, rm.Control{Name: name})
		if err != nil {
			t.Fatalf("CreateControl: %v", err)
		}
		pathway := cause
		if name == "Mobile water tanks" {
			pathway = consequence
		}
		if _, err := m.PlaceBarrier(ctx, rm.Barrier{PathwayID: pathway.ID, ControlID: control.ID}); err != nil {
			t.Fatalf("PlaceBarrier: %v", err)
		}
	}
	bowtie, err := m.Bowtie(ctx, risk.ID)
	if err != nil {
		t.Fatalf("Bowtie: %v", err)
	}
	if bowtie.Risk.Owner != "a-1" || len(bowtie.Threats) != 1 || len(bowtie.Consequences) != 1 ||
		len(bowtie.Threats[0].Barriers) != 1 || len(bowtie.Consequences[0].Barriers) != 1 {
		t.Fatalf("bowtie = %+v", bowtie)
	}
	listed, err := m.ListRisks(ctx, rm.RiskFilter{Owner: "a-1"})
	if err != nil || len(listed) != 1 || listed[0].ID != risk.ID {
		t.Fatalf("register by owner = %v, %v", listed, err)
	}
	if _, err := m.CreateRisk(ctx, rm.Risk{Title: "Other", TopEvent: "Other", Owner: "ghost"}); !errors.Is(err, rm.ErrInvalidReference) {
		t.Fatalf("unknown owner err = %v", err)
	}
	if again, _ := m.ListRisks(ctx, rm.RiskFilter{}); len(again) != 1 {
		t.Fatalf("a refused owner left %d risks", len(again))
	}
}
