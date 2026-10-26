package riskmanager_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func examplePath(domain string) string {
	return filepath.Join(".", "spec", "examples", domain+".riskmanager.json")
}

func newDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func newManagerFor(t *testing.T, domain string) (riskmanager.RiskManager, context.Context) {
	t.Helper()
	db := newDB(t)
	s, err := riskmanager.LoadSpec(examplePath(domain))
	if err != nil {
		t.Fatalf("load spec %s: %v", domain, err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate %s: %v", domain, err)
	}
	rm, err := riskmanager.NewRiskManager(db, s)
	if err != nil {
		t.Fatalf("new manager %s: %v", domain, err)
	}
	return rm, context.Background()
}

func newManager(t *testing.T) (riskmanager.RiskManager, context.Context) {
	t.Helper()
	return newManagerFor(t, "ai_safety")
}
