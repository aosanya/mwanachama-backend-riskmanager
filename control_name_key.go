package riskmanager

import (
	"fmt"
	"log/slog"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/plaintext"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

const controlNameKeyColumn = "name_key"

func controlNameKey(name string) string {
	return plaintext.FoldDrawnName(name)
}

type storedControlName struct {
	ID      string
	Name    string
	NameKey string
}

type controlKeyChange struct {
	id      string
	prior   string
	desired string
}

func backfillControlNameKeys(db *gorm.DB, s *spec.Spec) error {
	controls, ok := s.ByRole(roleControl)
	if !ok {
		return nil
	}
	table := s.TableFor(controls)
	var rows []storedControlName
	err := db.Table(table).
		Select("id, name, coalesce(" + controlNameKeyColumn + ", '') as " + controlNameKeyColumn).
		Order("created_at, id").
		Find(&rows).Error
	if err != nil {
		return fmt.Errorf("riskmanager: read the control names in %s: %w", table, err)
	}
	changes, unkeyed := controlKeyChanges(rows)
	if err := applyControlKeyChanges(db, table, changes); err != nil {
		return err
	}
	if unkeyed > 0 {
		slog.Warn("riskmanager: controls share a folded name, so the later ones keep an empty name key until they are renamed",
			"table", table, "controls", unkeyed)
	}
	return nil
}

func controlKeyChanges(rows []storedControlName) (changes []controlKeyChange, unkeyed int) {
	held := make(map[string]bool, len(rows))
	for _, r := range rows {
		if key := controlNameKey(r.Name); key != "" && r.NameKey == key {
			held[key] = true
		}
	}
	for _, r := range rows {
		key := controlNameKey(r.Name)
		desired := key
		if r.NameKey != key {
			if held[key] {
				desired = ""
			}
			held[key] = true
		}
		if desired == "" && key != "" {
			unkeyed++
		}
		if desired != r.NameKey {
			changes = append(changes, controlKeyChange{id: r.ID, prior: r.NameKey, desired: desired})
		}
	}
	return changes, unkeyed
}

func applyControlKeyChanges(db *gorm.DB, table string, changes []controlKeyChange) error {
	if len(changes) == 0 {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, c := range changes {
			if c.prior != "" {
				if err := setControlNameKey(tx, table, c.id, ""); err != nil {
					return err
				}
			}
		}
		for _, c := range changes {
			if c.desired != "" {
				if err := setControlNameKey(tx, table, c.id, c.desired); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func setControlNameKey(tx *gorm.DB, table, id, key string) error {
	if err := tx.Table(table).Where("id = ?", id).Update(controlNameKeyColumn, key).Error; err != nil {
		return fmt.Errorf("riskmanager: set the name key of control %s: %w", id, err)
	}
	return nil
}
