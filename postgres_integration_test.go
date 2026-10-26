//go:build integration

package riskmanager_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
)

func randomInstance(t *testing.T) string {
	t.Helper()
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("random instance: %v", err)
	}
	return "pg" + hex.EncodeToString(b)
}

func newPostgres(t *testing.T) (riskmanager.RiskManager, context.Context) {
	t.Helper()
	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		t.Skip("POSTGRES_URL is not set")
	}
	db, err := gorm.Open(gormpostgres.Open(url), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}

	raw, err := os.ReadFile(examplePath("ai_safety"))
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	doc["instance"] = randomInstance(t)
	rewritten, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("rewrite spec: %v", err)
	}

	s, err := riskmanager.ParseSpec(rewritten)
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if err := riskmanager.Provision(db, s); err != nil {
		t.Fatalf("provision: %v", err)
	}
	t.Cleanup(func() {
		for _, o := range s.Objects {
			db.Exec("drop table if exists " + s.TableFor(o) + " cascade")
		}
		db.Exec("drop table if exists " + s.NameRegistryTable() + " cascade")
	})

	rm, err := riskmanager.NewRiskManager(db, s)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	return rm, context.Background()
}

func TestPG_ADocumentSurvivesJSONBRoundTrip(t *testing.T) {
	rm, ctx := newPostgres(t)

	r, err := rm.CreateRisk(ctx, riskmanager.Risk{
		Title: "Data leaves through an assistant", TopEvent: "Sensitive data is disclosed",
		Doc: map[string]any{
			"category": "confidentiality",
			"score":    float64(4),
			"nested":   map[string]any{"regulation": "data protection"},
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	read, err := rm.GetRisk(ctx, r.ID)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if read.Doc["category"] != "confidentiality" || read.Doc["score"] != float64(4) {
		t.Fatalf("jsonb lost a document value: %+v", read.Doc)
	}
	nested, ok := read.Doc["nested"].(map[string]any)
	if !ok || nested["regulation"] != "data protection" {
		t.Fatalf("jsonb lost the nested document: %+v", read.Doc["nested"])
	}
}

func TestPG_TheUniqueIndexRefusesASecondPlacement(t *testing.T) {
	rm, ctx := newPostgres(t)

	r, err := rm.CreateRisk(ctx, riskmanager.Risk{Title: "Drafting", TopEvent: "Control is lost"})
	if err != nil {
		t.Fatalf("risk: %v", err)
	}
	p, err := rm.CreatePathway(ctx, riskmanager.Pathway{RiskID: r.ID, Side: riskmanager.SideThreat, Title: "A route"})
	if err != nil {
		t.Fatalf("pathway: %v", err)
	}
	c, err := rm.CreateControl(ctx, riskmanager.Control{Name: "A safeguard"})
	if err != nil {
		t.Fatalf("control: %v", err)
	}
	if _, err := rm.PlaceBarrier(ctx, riskmanager.Barrier{PathwayID: p.ID, ControlID: c.ID}); err != nil {
		t.Fatalf("first placement: %v", err)
	}
	if _, err := rm.PlaceBarrier(ctx, riskmanager.Barrier{PathwayID: p.ID, ControlID: c.ID}); err == nil {
		t.Fatalf("one control was placed twice on one route")
	}
}
