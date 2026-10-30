package riskmanager_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/identity"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
)

var _ identity.Owner = riskmanager.RiskManager(nil)

func holdOneOfEach(t *testing.T, rm riskmanager.RiskManager, ctx context.Context) map[string]string {
	t.Helper()
	r := seedRisk(t, rm, ctx, "r")
	p := seedThreat(t, rm, ctx, r.ID, "t")
	c := seedControl(t, rm, ctx, "c", "o")
	b := seedBarrier(t, rm, ctx, p.ID, c.ID)
	e := seedEvidence(t, rm, ctx, b.ID, riskmanager.OutcomePass, "2026-01-01T00:00:00Z")
	a := seedAssessment(t, rm, ctx, r.ID, riskmanager.AssessmentInherent, 3, 3, "2026-01-01T00:00:00Z")
	w := seedAction(t, rm, ctx, r.ID, b.ID, "w", "")
	return map[string]string{
		riskmanager.RoleRisk: r.ID, riskmanager.RolePathway: p.ID, riskmanager.RoleControl: c.ID, riskmanager.RoleBarrier: b.ID,
		riskmanager.RoleEvidence: e.ID, riskmanager.RoleAssessment: a.ID, riskmanager.RoleAction: w.ID,
	}
}

func TestStableIdentityIsTheRawRiskmanagerInstanceMountRoleAndRowId(t *testing.T) {
	rm, ctx := newManager(t)
	for role, id := range holdOneOfEach(t, rm, ctx) {
		raw, err := rm.StableIdentity(ctx, role, id)
		if err != nil {
			t.Fatalf("StableIdentity(%s): %v", role, err)
		}
		parsed, err := identity.Parse(raw)
		if err != nil || parsed.Module != "riskmanager" || !strings.HasSuffix(parsed.Identity, "/"+role+"/"+id) {
			t.Fatalf("raw %q for %s parsed as %+v, %v", raw, role, parsed, err)
		}
		object, rowID, err := rm.Resolve(ctx, raw)
		if err != nil || object != role || rowID != id {
			t.Fatalf("Resolve(%q) = %q %q %v, want %q %q", raw, object, rowID, err, role, id)
		}
	}
}

func TestResolveRefusesWhatIsNotThisRegistersWithTheSharedErrors(t *testing.T) {
	rm, ctx := newManager(t)
	held := holdOneOfEach(t, rm, ctx)
	raw, err := rm.StableIdentity(ctx, riskmanager.RoleRisk, held[riskmanager.RoleRisk])
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := identity.Parse(raw)
	for name, tc := range map[string]struct {
		raw  string
		want error
	}{
		"not a raw":        {"nonsense", identity.ErrInvalidRaw},
		"another module":   {"agency/" + parsed.Instance + "/" + parsed.Identity, identity.ErrNotOwned},
		"another instance": {"riskmanager/other/" + parsed.Identity, identity.ErrNotOwned},
		"another mount":    {"riskmanager/" + parsed.Instance + "/elsewhere/risk/" + held[riskmanager.RoleRisk], identity.ErrNotOwned},
		"an unknown role":  {"riskmanager/" + parsed.Instance + "/" + strings.SplitN(parsed.Identity, "/", 2)[0] + "/widget/1", identity.ErrNotFound},
		"a missing row":    {"riskmanager/" + parsed.Instance + "/" + strings.SplitN(parsed.Identity, "/", 2)[0] + "/risk/no-such-row", identity.ErrNotFound},
	} {
		if _, _, err := rm.Resolve(ctx, tc.raw); !errors.Is(err, tc.want) {
			t.Errorf("%s: Resolve(%q) = %v, want %v", name, tc.raw, err, tc.want)
		}
	}
	if _, err := rm.StableIdentity(ctx, riskmanager.RoleRisk, "no-such-row"); !errors.Is(err, identity.ErrNotFound) {
		t.Errorf("StableIdentity of a missing row = %v", err)
	}
	if _, err := rm.StableIdentity(ctx, "widget", "1"); !errors.Is(err, riskmanager.ErrNoStableIdentity) {
		t.Errorf("StableIdentity of an unknown role = %v", err)
	}
}
