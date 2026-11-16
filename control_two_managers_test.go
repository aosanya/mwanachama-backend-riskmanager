package riskmanager_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"

	"gorm.io/gorm"

	rm "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-shared/plaintext"
)

type twoManagers struct {
	first, second     rm.RiskManager
	firstDB, secondDB *gorm.DB
	controlsTable     string
}

func twinSpellings(t *testing.T) (composed, decomposed string) {
	t.Helper()
	composed, decomposed = "H\u00e1l\u00f3zati t\u0171zfal", "Ha\u0301lo\u0301zati tu\u030bzfal"
	if composed == decomposed || !plaintext.SameName(composed, decomposed) {
		t.Fatalf("%q and %q are not a composed and decomposed twin", composed, decomposed)
	}
	return composed, decomposed
}

func twoManagersOnOneFile(t *testing.T) twoManagers {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "risk.db") + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)"
	firstDB, secondDB := openControlDB(t, dsn), openControlDB(t, dsn)
	s, err := rm.LoadSpec(examplePath("ai_safety"))
	if err != nil {
		t.Fatal(err)
	}
	if err := rm.Provision(firstDB, s); err != nil {
		t.Fatal(err)
	}
	first, err := rm.NewRiskManager(firstDB, s)
	if err != nil {
		t.Fatal(err)
	}
	second, err := rm.NewRiskManager(secondDB, s)
	if err != nil {
		t.Fatal(err)
	}
	controls, _ := s.ByRole("control")
	return twoManagers{first: first, second: second, firstDB: firstDB, secondDB: secondDB, controlsTable: s.TableFor(controls)}
}

func (p twoManagers) beforeFirstWrite(t *testing.T, write string, other func(ctx context.Context) error) {
	t.Helper()
	var fired atomic.Bool
	hook := func(tx *gorm.DB) {
		if tx.Statement.Table != p.controlsTable || !fired.CompareAndSwap(false, true) {
			return
		}
		if err := other(tx.Statement.Context); err != nil {
			t.Errorf("the other manager's write: %v", err)
		}
	}
	callbacks := p.firstDB.Callback()
	var err error
	switch write {
	case "update":
		err = callbacks.Update().Before("gorm:begin_transaction").Register("rm:other-manager-first", hook)
	default:
		err = callbacks.Create().Before("gorm:begin_transaction").Register("rm:other-manager-first", hook)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func storedControls(t *testing.T, m rm.RiskManager) int {
	t.Helper()
	all, err := m.ListControls(context.Background(), rm.ControlFilter{})
	if err != nil {
		t.Fatal(err)
	}
	return len(all)
}

func TestControlTwinSpellings_AreDifferentBytesAndOneName(t *testing.T) {
	composed, decomposed := twinSpellings(t)
	if len(composed) == len(decomposed) {
		t.Errorf("%q and %q have the same byte length, so one cannot be the decomposed form of the other", composed, decomposed)
	}
}

func TestControlCreate_TwinsRacedFromTwoManagersOnOneFileLeaveOneControl(t *testing.T) {
	composed, decomposed := twinSpellings(t)
	ctx := context.Background()
	p := twoManagersOnOneFile(t)
	managers := []rm.RiskManager{p.first, p.second}
	spellings := []string{composed, decomposed}
	for round := 0; round < 3*rounds; round++ {
		suffix := fmt.Sprintf(" %d", round)
		results := race(racers, func(i int) error {
			_, err := managers[i%2].CreateControl(ctx, rm.Control{Name: spellings[(i/2)%2] + suffix})
			return err
		})
		requireOneWinnerRestConflict(t, results, composed+suffix)
	}
	if got := storedControls(t, p.first); got != 3*rounds {
		t.Errorf("%d controls are stored after %d rounds, want one per round", got, 3*rounds)
	}
}

func TestControlCreate_ATwinStoredByAnotherManagerBetweenTheCheckAndTheInsertIsAConflict(t *testing.T) {
	composed, decomposed := twinSpellings(t)
	ctx := context.Background()
	p := twoManagersOnOneFile(t)
	p.beforeFirstWrite(t, "create", func(ctx context.Context) error {
		_, err := p.second.CreateControl(ctx, rm.Control{Name: decomposed})
		return err
	})
	if _, err := p.first.CreateControl(ctx, rm.Control{Name: composed}); !errors.Is(err, rm.ErrConflict) {
		t.Fatalf("create after the twin was stored elsewhere = %v, want a conflict", err)
	}
	if got := storedControls(t, p.first); got != 1 {
		t.Errorf("%d controls are stored, want 1", got)
	}
}

func TestControlUpdate_ATwinStoredByAnotherManagerBetweenTheCheckAndTheWriteIsAConflict(t *testing.T) {
	composed, decomposed := twinSpellings(t)
	ctx := context.Background()
	p := twoManagersOnOneFile(t)
	mine, err := p.first.CreateControl(ctx, rm.Control{Name: "Draft"})
	if err != nil {
		t.Fatal(err)
	}
	p.beforeFirstWrite(t, "update", func(ctx context.Context) error {
		_, err := p.second.CreateControl(ctx, rm.Control{Name: decomposed})
		return err
	})
	mine.Name = composed
	if _, err := p.first.UpdateControl(ctx, mine); !errors.Is(err, rm.ErrConflict) {
		t.Fatalf("rename after the twin was stored elsewhere = %v, want a conflict", err)
	}
	if got := storedControls(t, p.first); got != 2 {
		t.Errorf("%d controls are stored, want the renamed one and the twin's holder only: 2", got)
	}
}
