package riskmanager_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	rm "github.com/aosanya/mwanachama-backend-riskmanager"
)

const contested = "Network firewall"

func requireConflictNaming(t *testing.T, err error, name string) {
	t.Helper()
	if !errors.Is(err, rm.ErrConflict) || !strings.Contains(err.Error(), name) {
		t.Fatalf("got %v, want a conflict naming %q", err, name)
	}
}

func TestControlCreate_AnExactNameStoredByAnotherManagerBetweenTheCheckAndTheInsertIsAConflict(t *testing.T) {
	ctx := context.Background()
	p := twoManagersOnOneFile(t)
	p.beforeFirstWrite(t, "create", func(ctx context.Context) error {
		_, err := p.second.CreateControl(ctx, rm.Control{Name: contested})
		return err
	})
	_, err := p.first.CreateControl(ctx, rm.Control{Name: contested})
	requireConflictNaming(t, err, contested)
	if got := storedControls(t, p.first); got != 1 {
		t.Errorf("%d controls are stored, want 1", got)
	}
}

func TestControlUpdate_AnExactNameStoredByAnotherManagerBetweenTheCheckAndTheWriteIsAConflict(t *testing.T) {
	ctx := context.Background()
	p := twoManagersOnOneFile(t)
	mine, err := p.first.CreateControl(ctx, rm.Control{Name: "Draft"})
	if err != nil {
		t.Fatal(err)
	}
	p.beforeFirstWrite(t, "update", func(ctx context.Context) error {
		_, err := p.second.CreateControl(ctx, rm.Control{Name: contested})
		return err
	})
	mine.Name = contested
	_, err = p.first.UpdateControl(ctx, mine)
	requireConflictNaming(t, err, contested)
	if got := storedControls(t, p.first); got != 2 {
		t.Errorf("%d controls are stored, want 2", got)
	}
}

func TestControlCreate_OneExactNameRacedFromTwoManagersOnOneFileLeavesOneControl(t *testing.T) {
	ctx := context.Background()
	p := twoManagersOnOneFile(t)
	managers := []rm.RiskManager{p.first, p.second}
	for round := 0; round < 3*rounds; round++ {
		name := contested + " " + string(rune('A'+round%26)) + string(rune('a'+round/26))
		results := race(racers, func(i int) error {
			_, err := managers[i%2].CreateControl(ctx, rm.Control{Name: name})
			return err
		})
		requireOneWinnerRestConflict(t, results, name)
	}
	if got := storedControls(t, p.first); got != 3*rounds {
		t.Errorf("%d controls are stored after %d rounds, want one per round", got, 3*rounds)
	}
}

func TestControlCreate_TwinsRacedFromOneManagerLeaveOneControl(t *testing.T) {
	composed, decomposed := twinSpellings(t)
	spellings := []string{composed, decomposed}
	for _, store := range controlStores {
		t.Run(store.name, func(t *testing.T) {
			ctx := context.Background()
			m := controlManagerOn(t, store.open(t))
			for round := 0; round < rounds; round++ {
				suffix := " " + string(rune('A'+round))
				results := race(racers, func(i int) error {
					_, err := m.CreateControl(ctx, rm.Control{Name: spellings[i%2] + suffix})
					return err
				})
				requireOneWinnerRestConflict(t, results, composed+suffix)
			}
			if got := storedControls(t, m); got != rounds {
				t.Errorf("%d controls are stored after %d rounds, want one per round", got, rounds)
			}
		})
	}
}
