package riskmanager_test

import (
	"sort"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/dispatch"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
)

var markedReferences = map[string]string{
	"riskID":    "riskmanager.risk",
	"pathwayID": "riskmanager.pathway",
	"controlID": "riskmanager.control",
	"barrierID": "riskmanager.barrier",
	"actionID":  "riskmanager.action",
}

func TestEveryArgumentNamingAThingTheRegisterHoldsIsMarkedWithWhatItRefersTo(t *testing.T) {
	s, err := dispatch.Parse(riskmanager.Operations())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	marked := map[string]int{}
	for _, name := range sortedOperationNames(s) {
		for _, a := range s.Operations[name].Args {
			want, isReference := markedReferences[a.As]
			isReference = isReference && !a.Ignored && !a.Whole
			if isReference && a.Refers != want {
				t.Errorf("%s: argument %s refers %q, want %q", name, a.As, a.Refers, want)
			}
			if !isReference && a.Refers != "" {
				t.Errorf("%s: argument %s refers %q but is not one of the things the register holds", name, a.As, a.Refers)
			}
			if isReference {
				marked[a.As]++
			}
		}
	}
	for as := range markedReferences {
		if marked[as] == 0 {
			t.Errorf("no operation declares %s, so nothing proves it is marked", as)
		}
	}
}

func TestReferencesListsTheMarkedArgumentsAndNoCallerOrBodyField(t *testing.T) {
	s, err := dispatch.Parse(riskmanager.Operations())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	references := s.References()
	if len(references) != 14 {
		t.Fatalf("References lists %d arguments, want the 14 addresses the file declares", len(references))
	}
	seen := map[string]bool{}
	for _, r := range references {
		seen[r.As] = true
		if markedReferences[r.As] != r.Refers {
			t.Errorf("%s.%s is listed as referring %q", r.Operation, r.As, r.Refers)
		}
	}
	for as := range markedReferences {
		if !seen[as] {
			t.Errorf("References never lists %s", as)
		}
	}
	for _, notAReference := range []string{"recorded_by", "assessed_by", "status"} {
		if seen[notAReference] {
			t.Errorf("%s is the session or a field of the body, not a reference", notAReference)
		}
	}
}

func sortedOperationNames(s *dispatch.Spec) []string {
	names := make([]string, 0, len(s.Operations))
	for name := range s.Operations {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
