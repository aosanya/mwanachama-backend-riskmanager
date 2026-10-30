package riskmanager_test

import (
	"context"
	"sync/atomic"
	"testing"

	"gorm.io/gorm"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

type readMeter struct {
	queries atomic.Int64
	rows    atomic.Int64
}

func (r *readMeter) reset() { r.queries.Store(0); r.rows.Store(0) }

func newMeteredManager(t *testing.T) (riskmanager.RiskManager, context.Context, *readMeter) {
	t.Helper()
	db := newDB(t)
	meter := &readMeter{}
	if err := db.Callback().Query().Before("gorm:query").Register("meter:queries", func(tx *gorm.DB) {
		if !tx.DryRun {
			meter.queries.Add(1)
		}
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := db.Callback().Query().After("gorm:query").Register("meter:rows", func(tx *gorm.DB) {
		if !tx.DryRun {
			meter.rows.Add(tx.RowsAffected)
		}
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	s, err := riskmanager.LoadSpec(examplePath("ai_safety"))
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	rm, err := riskmanager.NewRiskManager(db, s)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	return rm, context.Background(), meter
}
