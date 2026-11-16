package riskmanager_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	rm "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

const (
	racers = 8
	rounds = 12
)

type controlStore struct {
	name string
	open func(t *testing.T) *gorm.DB
}

var controlStores = []controlStore{
	{"sqlite file in WAL mode", func(t *testing.T) *gorm.DB {
		dsn := "file:" + filepath.Join(t.TempDir(), "risk.db") + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)"
		return openControlDB(t, dsn)
	}},
	{"sqlite shared memory", func(t *testing.T) *gorm.DB {
		dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_pragma=busy_timeout(10000)"
		return openControlDB(t, dsn)
	}},
}

func openControlDB(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func controlManagerOn(t *testing.T, db *gorm.DB) rm.RiskManager {
	t.Helper()
	s, err := rm.LoadSpec(examplePath("ai_safety"))
	if err != nil {
		t.Fatal(err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatal(err)
	}
	m, err := rm.NewRiskManager(db, s)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func race(n int, call func(i int) error) []error {
	results := make([]error, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i] = call(i)
		}(i)
	}
	close(start)
	wg.Wait()
	return results
}

func requireOneWinnerRestConflict(t *testing.T, results []error, name string) {
	t.Helper()
	winners := 0
	for _, err := range results {
		switch {
		case err == nil:
			winners++
		case errors.Is(err, rm.ErrConflict) && strings.Contains(err.Error(), "already exists"):
		default:
			t.Errorf("loser of %q got %v, want a conflict naming the control", name, err)
		}
	}
	if winners != 1 {
		t.Errorf("%q had %d winners, want 1", name, winners)
	}
}

func controlsCalled(t *testing.T, m rm.RiskManager, ctx context.Context, name string) int {
	t.Helper()
	all, err := m.ListControls(ctx, rm.ControlFilter{})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, c := range all {
		if c.Name == name {
			n++
		}
	}
	return n
}

func TestControlCreate_SimultaneousCreatesOfOneNameLeaveOneControlAndAConflictForTheRest(t *testing.T) {
	for _, store := range controlStores {
		t.Run(store.name, func(t *testing.T) {
			ctx := context.Background()
			m := controlManagerOn(t, store.open(t))
			for round := 0; round < rounds; round++ {
				name := fmt.Sprintf("Probe %d", round)
				results := race(racers, func(int) error {
					_, err := m.CreateControl(ctx, rm.Control{Name: name})
					return err
				})
				requireOneWinnerRestConflict(t, results, name)
				if got := controlsCalled(t, m, ctx, name); got != 1 {
					t.Errorf("%q is stored %d times, want 1", name, got)
				}
			}
		})
	}
}

func TestControlCreate_SimultaneousCreatesOfComposedAndDecomposedTwinsLeaveOneControl(t *testing.T) {
	forms := []string{"Hálózati tűzfal", "Hálózáti tűzfal"}
	for _, store := range controlStores {
		t.Run(store.name, func(t *testing.T) {
			ctx := context.Background()
			m := controlManagerOn(t, store.open(t))
			for round := 0; round < rounds; round++ {
				suffix := fmt.Sprintf(" %d", round)
				results := race(racers, func(i int) error {
					_, err := m.CreateControl(ctx, rm.Control{Name: forms[i%2] + suffix})
					return err
				})
				requireOneWinnerRestConflict(t, results, forms[0]+suffix)
			}
		})
	}
}

func TestControlUpdate_SimultaneousRenamesToOneTakenNameLeaveOneControlWithIt(t *testing.T) {
	for _, store := range controlStores {
		t.Run(store.name, func(t *testing.T) {
			ctx := context.Background()
			m := controlManagerOn(t, store.open(t))
			for round := 0; round < rounds; round++ {
				ids := make([]string, racers)
				for i := range ids {
					made, err := m.CreateControl(ctx, rm.Control{Name: fmt.Sprintf("Draft %d-%d", round, i)})
					if err != nil {
						t.Fatal(err)
					}
					ids[i] = made.ID
				}
				target := fmt.Sprintf("Final %d", round)
				results := race(racers, func(i int) error {
					_, err := m.UpdateControl(ctx, rm.Control{ID: ids[i], Name: target})
					return err
				})
				requireOneWinnerRestConflict(t, results, target)
				if got := controlsCalled(t, m, ctx, target); got != 1 {
					t.Errorf("%q is held by %d controls, want 1", target, got)
				}
			}
		})
	}
}
