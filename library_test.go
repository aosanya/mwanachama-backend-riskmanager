package riskmanager_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func aiSafetyLibrary(t *testing.T) riskmanager.Library {
	t.Helper()
	lib, err := riskmanager.LoadLibrary(filepath.Join(".", "library", "ai_safety.library.json"))
	seedNoErr(t, err)
	return lib
}

func TestTheAISafetyLibraryHoldsTheConversationsEightThreatsAndNoConsequences(t *testing.T) {
	lib := aiSafetyLibrary(t)
	if len(lib.Pathways) != 8 {
		t.Fatalf("%d pathways, want the conversation's eight", len(lib.Pathways))
	}
	for _, p := range lib.Pathways {
		if p.Side != riskmanager.SideThreat {
			t.Errorf("%q is a %s: the conversation's consequences were only drawn in an image", p.Title, p.Side)
		}
	}
	if lib.Note == "" {
		t.Fatal("the library does not say it ships no consequences")
	}
	if len(lib.Drilldowns) != 1 || lib.Drilldowns[0].Pathway != "RAG / retrieval leakage" || len(lib.Drilldowns[0].Pathways) != 6 {
		t.Fatalf("drilldowns: %+v", lib.Drilldowns)
	}
}

func importer(rm riskmanager.RiskManager, opened *int) func(context.Context) (riskmanager.RiskManager, error) {
	return func(context.Context) (riskmanager.RiskManager, error) {
		*opened++
		return rm, nil
	}
}

func loadedSpec(t *testing.T) *spec.Spec {
	t.Helper()
	s, err := riskmanager.LoadSpec(examplePath("ai_safety"))
	seedNoErr(t, err)
	return s
}

func TestImportingTheLibraryBuildsTheBowTieAndItsRetrievalSubBowTie(t *testing.T) {
	rm, ctx := newManager(t)
	opened := 0
	report, err := riskmanager.ImportLibrary(ctx, loadedSpec(t), aiSafetyLibrary(t), importer(rm, &opened))
	seedNoErr(t, err)
	if opened != 1 || report.Created == 0 || report.Existing != 0 {
		t.Fatalf("opened %d, report %+v", opened, report)
	}
	risks, err := rm.ListRisks(ctx, riskmanager.RiskFilter{})
	seedNoErr(t, err)
	if len(risks) != 2 {
		t.Fatalf("%d risks, want the parent and the retrieval sub-bow-tie", len(risks))
	}
	threats, err := rm.Threats(ctx, riskmanager.PathwayFilter{RiskID: risks[0].ID})
	seedNoErr(t, err)
	if len(threats) != 8 || threats[1].Drilldowns != 1 {
		t.Fatalf("%d threats, retrieval drilled %d times", len(threats), threats[1].Drilldowns)
	}
	sub, err := rm.Threats(ctx, riskmanager.PathwayFilter{RiskID: risks[1].ID})
	seedNoErr(t, err)
	if risks[1].ParentPathwayID != threats[1].ID || len(sub) != 6 {
		t.Fatalf("sub-bow-tie parent %q, %d threats", risks[1].ParentPathwayID, len(sub))
	}
	reliance, err := rm.PreventiveMeasures(ctx, riskmanager.MeasureFilter{RiskID: risks[0].ID, Query: "input DLP"})
	seedNoErr(t, err)
	if len(reliance) != 1 || reliance[0].Control.Doc["stage"] != "preventive" {
		t.Fatalf("input DLP: %+v", reliance)
	}
}

func TestImportingTheLibraryTwiceChangesNothingTheSecondTime(t *testing.T) {
	rm, ctx := newManager(t)
	opened := 0
	s, lib := loadedSpec(t), aiSafetyLibrary(t)
	_, err := riskmanager.ImportLibrary(ctx, s, lib, importer(rm, &opened))
	seedNoErr(t, err)
	before := registerSize(t, rm, ctx)
	second, err := riskmanager.ImportLibrary(ctx, s, lib, importer(rm, &opened))
	seedNoErr(t, err)
	if second.Created != 0 || second.Existing == 0 {
		t.Fatalf("second run %+v", second)
	}
	if after := registerSize(t, rm, ctx); after != before {
		t.Fatalf("the register held %v and now holds %v", before, after)
	}
}

type size struct{ risks, pathways, controls, barriers int }

func registerSize(t *testing.T, rm riskmanager.RiskManager, ctx context.Context) size {
	t.Helper()
	risks, err := rm.ListRisks(ctx, riskmanager.RiskFilter{Limit: everything})
	seedNoErr(t, err)
	controls, err := rm.ListControls(ctx, riskmanager.ControlFilter{Limit: everything})
	seedNoErr(t, err)
	barriers, err := rm.ListBarriers(ctx, riskmanager.BarrierFilter{Limit: everything})
	seedNoErr(t, err)
	threats, err := rm.Threats(ctx, riskmanager.PathwayFilter{Limit: everything})
	seedNoErr(t, err)
	return size{len(risks), len(threats), len(controls), len(barriers)}
}

func TestARowThatFailsCheckIsRefusedBeforeAConnectionIsOpened(t *testing.T) {
	rm, ctx := newManager(t)
	lib := aiSafetyLibrary(t)
	lib.Pathways[3].Controls[0].Name = ""
	opened := 0
	_, err := riskmanager.ImportLibrary(ctx, loadedSpec(t), lib, importer(rm, &opened))
	if !errors.Is(err, riskmanager.ErrInvalid) || opened != 0 {
		t.Fatalf("err %v, connections opened %d", err, opened)
	}
	lib = aiSafetyLibrary(t)
	lib.Drilldowns[0].Pathway = "no such pathway"
	_, err = riskmanager.ImportLibrary(ctx, loadedSpec(t), lib, importer(rm, &opened))
	if !errors.Is(err, riskmanager.ErrInvalid) || opened != 0 {
		t.Fatalf("err %v, connections opened %d", err, opened)
	}
	if got := registerSize(t, rm, ctx); got != (size{}) {
		t.Fatalf("a refused import wrote %v", got)
	}
}
