package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
)

func TestTheControlQueryBindsFailingAndAnswersTheWeakestFirst(t *testing.T) {
	mux, rm, ctx := newMux(t)
	for _, name := range []string{"sound", "shaky"} {
		c, err := rm.CreateControl(ctx, riskmanager.Control{Name: name, Owner: "o"})
		if err != nil {
			t.Fatal(err)
		}
		r := seedRisk(t, rm, ctx, name)
		b, err := rm.PlaceBarrier(ctx, riskmanager.Barrier{PathwayID: seedThreat(t, rm, ctx, r.ID, "t").ID, ControlID: c.ID})
		if err != nil {
			t.Fatal(err)
		}
		outcome := riskmanager.OutcomePass
		if name == "shaky" {
			outcome = riskmanager.OutcomeFail
		}
		if _, err := rm.RecordEvidence(ctx, riskmanager.Evidence{BarrierID: b.ID, Kind: riskmanager.EvidenceTest, Outcome: outcome, Summary: "s"}); err != nil {
			t.Fatal(err)
		}
	}
	w := do(t, mux, "GET", "/controls?failing=true", "")
	if w.Code != http.StatusOK {
		t.Fatalf("answered %d: %s", w.Code, w.Body)
	}
	var got []riskmanager.Control
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "shaky" || got[0].FailingBarriers != 1 {
		t.Fatalf("failing controls: %+v", got)
	}
}
