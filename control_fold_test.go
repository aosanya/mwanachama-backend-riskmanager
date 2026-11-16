package riskmanager_test

import (
	"context"
	"errors"
	"testing"

	rm "github.com/aosanya/mwanachama-backend-riskmanager"
)

const (
	composedControl   = "Hálózati tűzfal"
	decomposedControl = "Hálózati tűzfal"
)

func TestControlNames_ADecomposedTwinOfAControlNameIsASecondControl(t *testing.T) {
	ctx := context.Background()
	m := newManager(t)
	if _, err := m.CreateControl(ctx, rm.Control{Name: composedControl}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.CreateControl(ctx, rm.Control{Name: decomposedControl}); !errors.Is(err, rm.ErrConflict) {
		t.Fatalf("create the decomposed twin = %v, want a conflict", err)
	}
	other, err := m.CreateControl(ctx, rm.Control{Name: "Tuzoltó"})
	if err != nil {
		t.Fatal(err)
	}
	other.Name = decomposedControl
	if _, err := m.UpdateControl(ctx, other); !errors.Is(err, rm.ErrConflict) {
		t.Fatalf("rename onto the decomposed twin = %v, want a conflict", err)
	}
}

func TestControlNames_AControlMayTakeItsOwnNameInAnotherForm(t *testing.T) {
	ctx := context.Background()
	m := newManager(t)
	made, err := m.CreateControl(ctx, rm.Control{Name: composedControl})
	if err != nil {
		t.Fatal(err)
	}
	made.Name = decomposedControl
	if _, err := m.UpdateControl(ctx, made); err != nil {
		t.Fatalf("a control may take its own name in another form: %v", err)
	}
}
