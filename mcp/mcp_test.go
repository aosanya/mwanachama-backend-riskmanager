package mcp_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
	riskmcp "github.com/aosanya/mwanachama-backend-riskmanager/mcp"
	"github.com/aosanya/mwanachama-backend-riskmanager/routes"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

func newManager(t *testing.T) riskmanager.RiskManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	s, err := riskmanager.LoadSpec(filepath.Join("..", "spec", "examples", "ai_safety.riskmanager.json"))
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if err := spec.Migrate(db, s); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	rm, err := riskmanager.NewRiskManager(db, s)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	return rm
}

func connect(t *testing.T, resolve riskmcp.ManagerResolver, m routes.Mount) (*gosdkmcp.ClientSession, context.Context) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	server := gosdkmcp.NewServer(&gosdkmcp.Implementation{Name: "test", Version: "0.0.0"}, nil)
	if err := riskmcp.RegisterTools(server, resolve, m); err != nil {
		t.Fatalf("register: %v", err)
	}
	st, ct := gosdkmcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	client := gosdkmcp.NewClient(&gosdkmcp.Implementation{Name: "client", Version: "0.0.0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs, ctx
}

func byInstance(managers map[string]riskmanager.RiskManager) riskmcp.ManagerResolver {
	return func(_ context.Context, id string) (riskmanager.RiskManager, error) {
		rm, ok := managers[id]
		if !ok {
			return nil, errors.New("not found")
		}
		return rm, nil
	}
}

func callTool(t *testing.T, ctx context.Context, cs *gosdkmcp.ClientSession, name string, args map[string]any) *gosdkmcp.CallToolResult {
	t.Helper()
	res, err := cs.CallTool(ctx, &gosdkmcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return res
}

func structured(t *testing.T, res *gosdkmcp.CallToolResult, into any) {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool error: %v", res.Content[0].(*gosdkmcp.TextContent).Text)
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := json.Unmarshal(raw, into); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
}

func listed(t *testing.T, ctx context.Context, cs *gosdkmcp.ClientSession) map[string]*gosdkmcp.Tool {
	t.Helper()
	list, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	out := map[string]*gosdkmcp.Tool{}
	for _, tl := range list.Tools {
		out[tl.Name] = tl
	}
	return out
}

func TestEveryDeclaredOperationIsAToolThatNamesItsInstance(t *testing.T) {
	cs, ctx := connect(t, byInstance(nil), routes.Mount{})
	tools := listed(t, ctx, cs)
	if len(tools) != 32 {
		t.Fatalf("listed %d tools, want the 32 the operations file declares", len(tools))
	}
	for name, tl := range tools {
		raw, _ := json.Marshal(tl.InputSchema)
		if !strings.Contains(string(raw), `"instance_id"`) {
			t.Errorf("%s does not take an instance_id: %s", name, raw)
		}
	}
	for _, want := range []string{
		"riskmanager_risk_create", "riskmanager_risk_bowtie", "riskmanager_threat_list",
		"riskmanager_consequence_list", "riskmanager_preventive_measure_list",
		"riskmanager_mitigating_measure_list", "riskmanager_control_reliance", "riskmanager_gap_list",
	} {
		if _, ok := tools[want]; !ok {
			t.Errorf("tools/list is missing %s", want)
		}
	}
}

func TestTheCallerIsNeverOfferedToAnAgent(t *testing.T) {
	cs, ctx := connect(t, byInstance(nil), routes.Mount{})
	tools := listed(t, ctx, cs)
	for name, field := range map[string]string{
		"riskmanager_evidence_record":   "recorded_by",
		"riskmanager_assessment_record": "assessed_by",
	} {
		tl, ok := tools[name]
		if !ok {
			t.Fatalf("tools/list is missing %s", name)
		}
		raw, _ := json.Marshal(tl.InputSchema)
		if strings.Contains(string(raw), `"`+field+`"`) {
			t.Errorf("%s offers an agent the caller's own field %s: %s", name, field, raw)
		}
	}
}

func TestTheQuestionsAnAgentAsksAboutABowTie(t *testing.T) {
	rm := newManager(t)
	ctx0 := context.Background()
	risk, err := rm.CreateRisk(ctx0, riskmanager.Risk{Title: "Data leaves through an assistant", TopEvent: "Sensitive data is disclosed to an unauthorised party"})
	if err != nil {
		t.Fatalf("seed risk: %v", err)
	}
	threat, err := rm.CreatePathway(ctx0, riskmanager.Pathway{RiskID: risk.ID, Side: riskmanager.SideThreat, Title: "Prompt pasted into an unapproved service"})
	if err != nil {
		t.Fatalf("seed threat: %v", err)
	}
	consequence, err := rm.CreatePathway(ctx0, riskmanager.Pathway{RiskID: risk.ID, Side: riskmanager.SideConsequence, Title: "Customers must be told"})
	if err != nil {
		t.Fatalf("seed consequence: %v", err)
	}
	gateway, err := rm.CreateControl(ctx0, riskmanager.Control{Name: "Approved gateway", Owner: "platform-team"})
	if err != nil {
		t.Fatalf("seed control: %v", err)
	}
	notice, err := rm.CreateControl(ctx0, riskmanager.Control{Name: "Notification procedure", Owner: "legal"})
	if err != nil {
		t.Fatalf("seed control: %v", err)
	}
	if _, err := rm.PlaceBarrier(ctx0, riskmanager.Barrier{PathwayID: threat.ID, ControlID: gateway.ID}); err != nil {
		t.Fatalf("seed barrier: %v", err)
	}
	if _, err := rm.PlaceBarrier(ctx0, riskmanager.Barrier{PathwayID: consequence.ID, ControlID: notice.ID}); err != nil {
		t.Fatalf("seed barrier: %v", err)
	}

	cs, ctx := connect(t, byInstance(map[string]riskmanager.RiskManager{"hub": rm}), routes.Mount{})

	var threats struct{ Items []riskmanager.PathwaySummary }
	structured(t, callTool(t, ctx, cs, "riskmanager_threat_list", map[string]any{
		"instance_id": "hub", "risk_id": risk.ID,
	}), &threats)
	if len(threats.Items) != 1 || threats.Items[0].Title != "Prompt pasted into an unapproved service" || threats.Items[0].BarrierCount != 1 {
		t.Fatalf("what are the threats answered %+v", threats.Items)
	}

	var consequences struct{ Items []riskmanager.PathwaySummary }
	structured(t, callTool(t, ctx, cs, "riskmanager_consequence_list", map[string]any{
		"instance_id": "hub", "risk_id": risk.ID,
	}), &consequences)
	if len(consequences.Items) != 1 || consequences.Items[0].Title != "Customers must be told" {
		t.Fatalf("what are the consequences answered %+v", consequences.Items)
	}

	var preventive struct{ Items []riskmanager.Measure }
	structured(t, callTool(t, ctx, cs, "riskmanager_preventive_measure_list", map[string]any{
		"instance_id": "hub", "risk_id": risk.ID,
	}), &preventive)
	if len(preventive.Items) != 1 || preventive.Items[0].Control.Name != "Approved gateway" || preventive.Items[0].Standing != riskmanager.StandingUnproven {
		t.Fatalf("what are the preventive measures answered %+v", preventive.Items)
	}

	var mitigating struct{ Items []riskmanager.Measure }
	structured(t, callTool(t, ctx, cs, "riskmanager_mitigating_measure_list", map[string]any{
		"instance_id": "hub", "risk_id": risk.ID,
	}), &mitigating)
	if len(mitigating.Items) != 1 || mitigating.Items[0].Control.Name != "Notification procedure" {
		t.Fatalf("what are the mitigating measures answered %+v", mitigating.Items)
	}

	var searched struct{ Items []riskmanager.Measure }
	structured(t, callTool(t, ctx, cs, "riskmanager_preventive_measure_list", map[string]any{
		"instance_id": "hub", "query": "gateway",
	}), &searched)
	if len(searched.Items) != 1 {
		t.Fatalf("a search by words answered %+v", searched.Items)
	}
}

func TestEachInstanceReachesOnlyItsOwnRegister(t *testing.T) {
	one, two := newManager(t), newManager(t)
	cs, ctx := connect(t, byInstance(map[string]riskmanager.RiskManager{"one": one, "two": two}), routes.Mount{})

	r, err := one.CreateRisk(context.Background(), riskmanager.Risk{Title: "Mine", TopEvent: "It happens"})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if res := callTool(t, ctx, cs, "riskmanager_risk_read", map[string]any{"instance_id": "two", "riskID": r.ID}); !res.IsError {
		t.Fatalf("instance two read instance one's risk")
	}
	if res := callTool(t, ctx, cs, "riskmanager_risk_read", map[string]any{"instance_id": "one", "riskID": r.ID}); res.IsError {
		t.Fatalf("instance one could not read its own risk")
	}
	if res := callTool(t, ctx, cs, "riskmanager_risk_read", map[string]any{"instance_id": "nobody", "riskID": r.ID}); !res.IsError {
		t.Fatalf("an unknown instance resolved")
	}
	if res := callTool(t, ctx, cs, "riskmanager_risk_read", map[string]any{"riskID": r.ID}); !res.IsError {
		t.Fatalf("a call naming no instance was answered")
	}
}

func TestARefusedActionNeverReachesTheManager(t *testing.T) {
	rm := newManager(t)
	deny := func(context.Context, string) error { return errors.New("refused") }
	cs, ctx := connect(t, byInstance(map[string]riskmanager.RiskManager{"hub": rm}), routes.Mount{Authorize: deny})
	if res := callTool(t, ctx, cs, "riskmanager_risk_create", map[string]any{
		"instance_id": "hub", "title": "Refused", "top_event": "never stored",
	}); !res.IsError {
		t.Fatalf("a refused create was answered")
	}
	held, err := rm.ListRisks(context.Background(), riskmanager.RiskFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(held) != 0 {
		t.Fatalf("a refused create stored %d risk(s)", len(held))
	}
}
