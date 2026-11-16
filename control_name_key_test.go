package riskmanager_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"gorm.io/gorm"

	rm "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

type storedKey struct {
	Name    string
	NameKey string
}

func storedKeys(t *testing.T, db *gorm.DB, table string) map[string]string {
	t.Helper()
	var rows []storedKey
	if err := db.Table(table).Select("name, name_key").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	keys := make(map[string]string, len(rows))
	for _, r := range rows {
		keys[r.Name] = r.NameKey
	}
	return keys
}

func controlsBeforeTheKeyExisted(t *testing.T) (*gorm.DB, *spec.Spec, string) {
	t.Helper()
	db := openControlDB(t, "file:"+filepath.Join(t.TempDir(), "legacy.db"))
	s, err := rm.LoadSpec(examplePath("ai_safety"))
	if err != nil {
		t.Fatal(err)
	}
	controls, _ := s.ByRole("control")
	table := s.TableFor(controls)
	create := "create table " + table + " (id text primary key, name text not null unique, description text default '', owner text default '', doc json, created_at text, updated_at text)"
	if err := db.Exec(create).Error; err != nil {
		t.Fatal(err)
	}
	composed, decomposed := twinSpellings(t)
	stored := []struct{ id, name, created string }{
		{"c1", "Backup", "2026-01-01T00:00:01.000000Z"},
		{"c2", composed, "2026-01-01T00:00:02.000000Z"},
		{"c3", decomposed, "2026-01-01T00:00:03.000000Z"},
		{"c4", "Firewall  Rules.", "2026-01-01T00:00:04.000000Z"},
	}
	for _, c := range stored {
		insert := "insert into " + table + " (id, name, created_at, updated_at) values (?, ?, ?, ?)"
		if err := db.Exec(insert, c.id, c.name, c.created, c.created).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db, s, table
}

func TestProvision_FillsTheNameKeyOfControlsStoredBeforeItExisted(t *testing.T) {
	db, s, table := controlsBeforeTheKeyExisted(t)
	composed, decomposed := twinSpellings(t)
	if err := rm.Provision(db, s); err != nil {
		t.Fatal(err)
	}
	keys := storedKeys(t, db, table)
	want := map[string]string{
		"Backup":           "backup",
		composed:           "hálózati tűzfal",
		decomposed:         "",
		"Firewall  Rules.": "firewall rules",
	}
	for name, key := range want {
		if keys[name] != key {
			t.Errorf("the name key of %q = %q, want %q", name, keys[name], key)
		}
	}
	if err := rm.Provision(db, s); err != nil {
		t.Fatalf("a second Provision: %v", err)
	}
	for name, key := range storedKeys(t, db, table) {
		if key != want[name] {
			t.Errorf("after a second Provision the name key of %q = %q, want %q", name, key, want[name])
		}
	}
}

func TestProvision_ALegacyDatabaseRefusesANewTwinOfAKeyedControl(t *testing.T) {
	db, s, _ := controlsBeforeTheKeyExisted(t)
	if err := rm.Provision(db, s); err != nil {
		t.Fatal(err)
	}
	m, err := rm.NewRiskManager(db, s)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.CreateControl(context.Background(), rm.Control{Name: "BACKUP"}); !errors.Is(err, rm.ErrConflict) {
		t.Fatalf("create a case twin of a stored control = %v, want a conflict", err)
	}
}

func TestControlNames_TheRegisterComparesCaseSpaceRunsAndTrailingDots(t *testing.T) {
	m := newManager(t)
	ctx := context.Background()
	if _, err := m.CreateControl(ctx, rm.Control{Name: "Network firewall"}); err != nil {
		t.Fatal(err)
	}
	for _, twin := range []string{"network FIREWALL", "Network   firewall", " Network firewall. "} {
		if _, err := m.CreateControl(ctx, rm.Control{Name: twin}); !errors.Is(err, rm.ErrConflict) {
			t.Errorf("create %q = %v, want a conflict", twin, err)
		}
	}
}
