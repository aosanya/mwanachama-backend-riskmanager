package riskmanager_test

import (
	"strings"
	"testing"

	"gorm.io/gorm"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
	"github.com/aosanya/mwanachama-backend-shared/spec"
)

var mountRoles = []string{
	riskmanager.RoleRisk, riskmanager.RolePathway, riskmanager.RoleControl,
	riskmanager.RoleBarrier, riskmanager.RoleEvidence, riskmanager.RoleAssessment,
	riskmanager.RoleAction,
}

func mountManager(t *testing.T, db *gorm.DB, instance, mount string) (riskmanager.RiskManager, *spec.Spec) {
	t.Helper()
	s, err := riskmanager.SpecForMount(instance, mount)
	if err != nil {
		t.Fatalf("SpecForMount(%q, %q): %v", instance, mount, err)
	}
	if err := riskmanager.Provision(db, s); err != nil {
		t.Fatalf("Provision(%q, %q): %v", instance, mount, err)
	}
	rm, err := riskmanager.NewRiskManager(db, s)
	if err != nil {
		t.Fatalf("NewRiskManager(%q, %q): %v", instance, mount, err)
	}
	return rm, s
}

func mountRisk(t *testing.T, rm riskmanager.RiskManager, title string) riskmanager.Risk {
	t.Helper()
	r, err := rm.CreateRisk(t.Context(), riskmanager.Risk{Title: title, TopEvent: title + " happens", Owner: "someone"})
	if err != nil {
		t.Fatalf("CreateRisk %q: %v", title, err)
	}
	return r
}

func TestSpecForBuildsTheWakalaDomainForAnyInstanceName(t *testing.T) {
	for _, instance := range []string{"app", "town", "x1", "board7"} {
		t.Run(instance, func(t *testing.T) {
			s, err := riskmanager.SpecFor(instance)
			if err != nil {
				t.Fatalf("SpecFor(%q): %v", instance, err)
			}
			if s.Instance != instance || s.Module != "riskmanager" || s.Domain != "wakala" {
				t.Fatalf("the spec reads instance=%q module=%q domain=%q", s.Instance, s.Module, s.Domain)
			}
			if s.MountName() != spec.DefaultMount {
				t.Fatalf("SpecFor names the mount %q, want the default %q", s.MountName(), spec.DefaultMount)
			}
			for _, role := range mountRoles {
				if _, ok := s.ByRole(role); !ok {
					t.Errorf("the wakala domain fills no object for the role %q", role)
				}
			}
		})
	}
}

func TestSpecForRefusesAnInstanceThatIsNotANameSegment(t *testing.T) {
	for _, instance := range []string{"bad_name", "", "with space", "semi;colon", "Upper_Case"} {
		t.Run(instance, func(t *testing.T) {
			_, err := riskmanager.SpecFor(instance)
			if err == nil {
				t.Fatalf("SpecFor(%q) was accepted", instance)
			}
			if !strings.Contains(err.Error(), "is not a usable name segment") {
				t.Fatalf("SpecFor(%q) was refused for another reason: %v", instance, err)
			}
		})
	}
}

func TestSpecForMountNamesTheMountAndRefusesABadOne(t *testing.T) {
	s, err := riskmanager.SpecForMount("app", "register")
	if err != nil {
		t.Fatalf("SpecForMount: %v", err)
	}
	if s.MountName() != "register" {
		t.Fatalf("the mount is %q, want register", s.MountName())
	}
	if _, err := riskmanager.SpecForMount("app", "bad_mount"); err == nil || !strings.Contains(err.Error(), "is not a usable name segment") {
		t.Fatalf("a mount with an underscore was not refused by the engine: %v", err)
	}
}

func TestTheWakalaDomainNamesEveryRoleForItselfAndFillsEachOnce(t *testing.T) {
	s, err := riskmanager.SpecFor("app")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if len(s.Objects) != len(mountRoles) {
		t.Fatalf("the wakala domain declares %d objects for %d roles", len(s.Objects), len(mountRoles))
	}
	for _, o := range s.Objects {
		if o.Role == "" {
			t.Errorf("the wakala domain declares %q with no role, which would be an object of its own", o.Name)
		}
		if o.Name != o.Role {
			t.Errorf("the wakala domain calls the role %q %q, so it adds vocabulary the platform has to learn", o.Role, o.Name)
		}
		for _, text := range []string{o.Name, o.Table} {
			if w, bad := carriesDomainWord(text); bad {
				t.Errorf("the wakala domain's %q carries the domain word %q", text, w)
			}
		}
	}
}

func TestTwoMountsAndTwoInstancesOfTheWakalaDomainKeepSeparateTables(t *testing.T) {
	db := newDB(t)
	first, firstSpec := mountManager(t, db, "app", "first")
	second, secondSpec := mountManager(t, db, "app", "second")
	other, otherSpec := mountManager(t, db, "town", "first")

	pairs := []struct {
		name string
		a, b *spec.Spec
	}{
		{"two mounts of one instance", firstSpec, secondSpec},
		{"two instances of one mount", firstSpec, otherSpec},
	}
	for _, p := range pairs {
		for _, role := range mountRoles {
			oa, _ := p.a.ByRole(role)
			ob, _ := p.b.ByRole(role)
			if p.a.TableFor(oa) == p.b.TableFor(ob) {
				t.Errorf("%s: the role %q landed in one table, %s", p.name, role, p.a.TableFor(oa))
			}
		}
	}
	oa, _ := firstSpec.ByRole(riskmanager.RoleRisk)
	ob, _ := secondSpec.ByRole(riskmanager.RoleRisk)
	if firstSpec.RawNameFor(oa) == secondSpec.RawNameFor(ob) {
		t.Errorf("two mounts share the raw name %q", firstSpec.RawNameFor(oa))
	}

	mountRisk(t, first, "First mount's risk")
	mountRisk(t, first, "First mount's second risk")
	mountRisk(t, second, "Second mount's risk")

	for _, c := range []struct {
		name string
		rm   riskmanager.RiskManager
		want int
	}{{"first mount", first, 2}, {"second mount", second, 1}, {"other instance", other, 0}} {
		held, err := c.rm.ListRisks(t.Context(), riskmanager.RiskFilter{})
		if err != nil {
			t.Fatalf("%s: ListRisks: %v", c.name, err)
		}
		if len(held) != c.want {
			t.Errorf("the %s sees %d risks, want %d, so one mount is reading another's table: %+v", c.name, len(held), c.want, held)
		}
	}
}

func TestAManagerWorksOverASpecForMountSpecAfterProvision(t *testing.T) {
	db := newDB(t)
	rm, _ := mountManager(t, db, "town", "register")
	ctx := t.Context()

	risk := mountRisk(t, rm, "The gate is left open")
	threat, err := rm.CreatePathway(ctx, riskmanager.Pathway{RiskID: risk.ID, Side: riskmanager.SideThreat, Title: "Nobody locks up"})
	if err != nil {
		t.Fatalf("CreatePathway: %v", err)
	}
	control, err := rm.CreateControl(ctx, riskmanager.Control{Name: "Closing checklist", Owner: "caretaker"})
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}
	barrier, err := rm.PlaceBarrier(ctx, riskmanager.Barrier{PathwayID: threat.ID, ControlID: control.ID})
	if err != nil {
		t.Fatalf("PlaceBarrier: %v", err)
	}
	if _, err := rm.RecordEvidence(ctx, riskmanager.Evidence{
		BarrierID: barrier.ID,
		Kind:      riskmanager.EvidenceAudit,
		Outcome:   riskmanager.OutcomePass,
		Summary:   "Walked the site at closing and found the gate locked.",
	}); err != nil {
		t.Fatalf("RecordEvidence: %v", err)
	}

	view, err := rm.Bowtie(ctx, risk.ID)
	if err != nil {
		t.Fatalf("Bowtie: %v", err)
	}
	if len(view.Threats) != 1 || len(view.Threats[0].Barriers) != 1 {
		t.Fatalf("the bow-tie reads %d threats, %+v", len(view.Threats), view.Threats)
	}
	placed := view.Threats[0].Barriers[0]
	if placed.Control.Name != "Closing checklist" || placed.EvidenceCount != 1 {
		t.Fatalf("the barrier reads %+v", placed)
	}
}
