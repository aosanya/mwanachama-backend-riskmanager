package riskmanager_test

import (
	"context"
	"testing"
	"time"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
)

type parkedDirectory struct {
	directory
	entered chan struct{}
	release chan struct{}
}

func (p parkedDirectory) Resolve(ctx context.Context, raw string) (string, string, error) {
	p.entered <- struct{}{}
	<-p.release
	return p.directory.Resolve(ctx, raw)
}

func TestControlUpdate_ABlockedOwnerLookupDoesNotStallTheCreationOfAnotherControl(t *testing.T) {
	parked := parkedDirectory{directory: people(), entered: make(chan struct{}, 1), release: make(chan struct{})}
	m, ctx := managerWithOwners(t, parked, "actor")
	held, err := m.CreateControl(ctx, riskmanager.Control{Name: "Backup"})
	if err != nil {
		t.Fatal(err)
	}
	updated := make(chan error, 1)
	go func() {
		held.Owner = ann
		_, err := m.UpdateControl(ctx, held)
		updated <- err
	}()
	<-parked.entered
	defer func() {
		close(parked.release)
		if err := <-updated; err != nil {
			t.Errorf("the update once the lookup answered: %v", err)
		}
	}()

	created := make(chan error, 1)
	go func() {
		short, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		defer cancel()
		_, err := m.CreateControl(short, riskmanager.Control{Name: "Firewall"})
		created <- err
	}()
	select {
	case err := <-created:
		if err != nil {
			t.Errorf("create of another control while an owner lookup is blocked: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("create of another control was still waiting two seconds after its own 200 ms deadline, behind a blocked owner lookup")
	}
}
