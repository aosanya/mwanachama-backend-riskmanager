package riskmanager

import (
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func Provision(db *gorm.DB, s *spec.Spec) error {
	if err := spec.Migrate(db, s); err != nil {
		return err
	}
	return backfillControlNameKeys(db, s)
}
