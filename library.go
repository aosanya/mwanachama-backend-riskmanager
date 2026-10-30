package riskmanager

import (
	"context"
	"encoding/json"
	"fmt"

	"os"

	"github.com/aosanya/mwanachama-backend-shared/spec"
)

type LibraryControl struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Stage       string `json:"stage"`
}

type LibraryPathway struct {
	Title    string           `json:"title"`
	Scenario string           `json:"scenario,omitempty"`
	Side     Side             `json:"side"`
	Controls []LibraryControl `json:"controls"`
}

type LibraryRisk struct {
	Title    string `json:"title"`
	TopEvent string `json:"top_event"`
	Hazard   string `json:"hazard,omitempty"`
}

type LibraryDrilldown struct {
	Pathway  string           `json:"pathway"`
	Risk     LibraryRisk      `json:"risk"`
	Pathways []LibraryPathway `json:"pathways"`
}

type Library struct {
	Name       string             `json:"library"`
	Note       string             `json:"note"`
	Risk       LibraryRisk        `json:"risk"`
	Pathways   []LibraryPathway   `json:"pathways"`
	Drilldowns []LibraryDrilldown `json:"drilldowns"`
}

type ImportReport struct {
	Created  int
	Existing int
}

func ParseLibrary(raw []byte) (Library, error) {
	var lib Library
	if err := json.Unmarshal(raw, &lib); err != nil {
		return Library{}, fmt.Errorf("%w: library: %v", ErrInvalid, err)
	}
	return lib, nil
}

func LoadLibrary(path string) (Library, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Library{}, fmt.Errorf("%w: library: %v", ErrInvalid, err)
	}
	return ParseLibrary(raw)
}

func (lib Library) Check(s *spec.Spec) error {
	if err := checkLibraryBowtie(s, lib.Risk, lib.Pathways); err != nil {
		return err
	}
	held := map[string]bool{}
	for _, p := range lib.Pathways {
		held[p.Title] = true
	}
	for _, d := range lib.Drilldowns {
		if !held[d.Pathway] {
			return fmt.Errorf("%w: the drill-down %q names the pathway %q, which the library does not hold", ErrInvalid, d.Risk.Title, d.Pathway)
		}
		if err := checkLibraryBowtie(s, d.Risk, d.Pathways); err != nil {
			return err
		}
	}
	return nil
}

func checkLibraryBowtie(s *spec.Spec, r LibraryRisk, pathways []LibraryPathway) error {
	if err := Check(s, RoleRisk, Risk{Title: r.Title, TopEvent: r.TopEvent, Hazard: r.Hazard, Status: StatusDraft}); err != nil {
		return err
	}
	for _, p := range pathways {
		if err := Check(s, RolePathway, Pathway{RiskID: "checked-on-import", Title: p.Title, Scenario: p.Scenario, Side: p.Side}); err != nil {
			return err
		}
		for _, c := range p.Controls {
			if err := Check(s, RoleControl, libraryControl(c)); err != nil {
				return err
			}
		}
	}
	return nil
}

func libraryControl(c LibraryControl) Control {
	return Control{Name: c.Name, Description: c.Description, Doc: map[string]any{"stage": c.Stage}}
}

func ImportLibrary(ctx context.Context, s *spec.Spec, lib Library, open func(context.Context) (RiskManager, error)) (ImportReport, error) {
	if err := lib.Check(s); err != nil {
		return ImportReport{}, err
	}
	rm, err := open(ctx)
	if err != nil {
		return ImportReport{}, err
	}
	imp := libraryImport{rm: rm}
	parent, err := imp.bowtie(ctx, lib.Risk, "", lib.Pathways)
	if err != nil {
		return imp.report, err
	}
	for _, d := range lib.Drilldowns {
		if _, err := imp.bowtie(ctx, d.Risk, parent[d.Pathway].ID, d.Pathways); err != nil {
			return imp.report, err
		}
	}
	return imp.report, nil
}

type libraryImport struct {
	rm     RiskManager
	report ImportReport
}

func (i *libraryImport) bowtie(ctx context.Context, r LibraryRisk, parentPathwayID string, pathways []LibraryPathway) (map[string]Pathway, error) {
	risk, err := i.risk(ctx, r, parentPathwayID)
	if err != nil {
		return nil, err
	}
	held, err := i.rm.Threats(ctx, PathwayFilter{RiskID: risk.ID, Limit: DefaultPage * 100})
	if err != nil {
		return nil, err
	}
	have := map[string]Pathway{}
	for _, h := range held {
		have[h.Title] = h.Pathway
	}
	consequences, err := i.rm.Consequences(ctx, PathwayFilter{RiskID: risk.ID, Limit: DefaultPage * 100})
	if err != nil {
		return nil, err
	}
	for _, h := range consequences {
		have[h.Title] = h.Pathway
	}
	out := map[string]Pathway{}
	for _, p := range pathways {
		pathway, ok := have[p.Title]
		if ok {
			i.report.Existing++
		} else {
			pathway, err = i.rm.CreatePathway(ctx, Pathway{RiskID: risk.ID, Side: p.Side, Title: p.Title, Scenario: p.Scenario})
			if err != nil {
				return nil, err
			}
			i.report.Created++
		}
		out[p.Title] = pathway
		if err := i.stand(ctx, pathway, p.Controls); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (i *libraryImport) risk(ctx context.Context, r LibraryRisk, parentPathwayID string) (Risk, error) {
	held, err := i.rm.ListRisks(ctx, RiskFilter{Query: r.Title, ParentPathwayID: parentPathwayID, Limit: DefaultPage * 100})
	if err != nil {
		return Risk{}, err
	}
	for _, h := range held {
		if h.Title == r.Title && h.ParentPathwayID == parentPathwayID {
			i.report.Existing++
			return h, nil
		}
	}
	created, err := i.rm.CreateRisk(ctx, Risk{Title: r.Title, TopEvent: r.TopEvent, Hazard: r.Hazard, ParentPathwayID: parentPathwayID})
	if err != nil {
		return Risk{}, err
	}
	i.report.Created++
	return created, nil
}

func (i *libraryImport) stand(ctx context.Context, pathway Pathway, controls []LibraryControl) error {
	if len(controls) == 0 {
		return nil
	}
	stands, err := i.rm.ListBarriers(ctx, BarrierFilter{PathwayID: pathway.ID, Limit: DefaultPage * 100})
	if err != nil {
		return err
	}
	standing := map[string]bool{}
	for _, b := range stands {
		standing[b.ControlID] = true
	}
	known, err := i.rm.ListControls(ctx, ControlFilter{Limit: DefaultPage * 100})
	if err != nil {
		return err
	}
	byName := map[string]Control{}
	for _, c := range known {
		byName[c.Name] = c
	}
	for position, lc := range controls {
		control, ok := byName[lc.Name]
		if ok {
			i.report.Existing++
		} else {
			control, err = i.rm.CreateControl(ctx, libraryControl(lc))
			if err != nil {
				return err
			}
			byName[lc.Name] = control
			i.report.Created++
		}
		if standing[control.ID] {
			i.report.Existing++
			continue
		}
		if _, err := i.rm.PlaceBarrier(ctx, Barrier{PathwayID: pathway.ID, ControlID: control.ID, Position: position + 1}); err != nil {
			return err
		}
		i.report.Created++
	}
	return nil
}
