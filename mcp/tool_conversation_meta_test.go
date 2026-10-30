package mcp_test

import (
	"context"
	"sync"
	"testing"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-riskmanager/routes"
)

type conversationRecorder struct {
	riskmanager.RiskManager
	mu   sync.Mutex
	seen []string
}

func (r *conversationRecorder) GetRisk(ctx context.Context, id string) (riskmanager.Risk, error) {
	r.mu.Lock()
	r.seen = append(r.seen, dispatch.ConversationOf(ctx))
	r.mu.Unlock()
	return r.RiskManager.GetRisk(ctx, id)
}

func conversationsSeenBy(t *testing.T, meta gosdkmcp.Meta) []string {
	t.Helper()
	inner := newManager(t)
	ctx := context.Background()
	risk, err := inner.CreateRisk(ctx, riskmanager.Risk{Title: "r", TopEvent: "e"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	recorder := &conversationRecorder{RiskManager: inner}
	cs, callCtx := connect(t, func(context.Context, string) (riskmanager.RiskManager, error) { return recorder, nil }, routes.Mount{})
	res, err := cs.CallTool(callCtx, &gosdkmcp.CallToolParams{
		Meta:      meta,
		Name:      "riskmanager_risk_read",
		Arguments: map[string]any{"instance_id": "any", "riskID": risk.ID},
	})
	if err != nil || res.IsError {
		t.Fatalf("CallTool: %v %+v", err, res)
	}
	return recorder.seen
}

func TestAToolCallCarryingMetaConversationIdReachesTheManagerOnItsContext(t *testing.T) {
	seen := conversationsSeenBy(t, gosdkmcp.Meta{dispatch.ConversationMetaKey: "chat-42"})
	if len(seen) != 1 || seen[0] != "chat-42" {
		t.Fatalf("the manager saw conversations %q, want exactly [chat-42]", seen)
	}
}

func TestAToolCallCarryingNoMetaReachesTheManagerWithNoConversation(t *testing.T) {
	seen := conversationsSeenBy(t, nil)
	if len(seen) != 1 || seen[0] != "" {
		t.Fatalf("the manager saw conversations %q, want exactly an empty one", seen)
	}
}
