package riskmanager

import (
	"context"
	"fmt"
)

func (m *riskManager) Threats(ctx context.Context, filter PathwayFilter) ([]PathwaySummary, error) {
	return m.summarisePathways(ctx, SideThreat, filter)
}

func (m *riskManager) Consequences(ctx context.Context, filter PathwayFilter) ([]PathwaySummary, error) {
	return m.summarisePathways(ctx, SideConsequence, filter)
}

func (m *riskManager) summarisePathways(ctx context.Context, side Side, filter PathwayFilter) ([]PathwaySummary, error) {
	if filter.RiskID != "" {
		if _, err := m.GetRisk(ctx, filter.RiskID); err != nil {
			return nil, err
		}
	}
	q := m.q(ctx, rolePathway).Where("side = ?", string(side))
	if filter.RiskID != "" {
		q = q.Where("risk_id = ?", filter.RiskID)
	}
	q = containing(q, filter.Query, "title", "scenario")
	pathways, err := listOf[Pathway](m, q.Order("created_at").Limit(capped(filter.Limit)), rolePathway)
	if err != nil {
		return nil, err
	}
	out := make([]PathwaySummary, 0, len(pathways))
	if len(pathways) == 0 {
		return out, nil
	}

	pathwayIDs := make([]string, len(pathways))
	riskIDs := make([]string, len(pathways))
	for i, p := range pathways {
		pathwayIDs[i] = p.ID
		riskIDs[i] = p.RiskID
	}
	guards, err := m.asmCountBy(ctx, roleBarrier, "pathway_id", pathwayIDs)
	if err != nil {
		return nil, err
	}
	drills, err := m.asmCountBy(ctx, roleRisk, "parent_pathway_id", pathwayIDs)
	if err != nil {
		return nil, err
	}
	risks, err := m.asmRisksByID(ctx, asmUnique(riskIDs))
	if err != nil {
		return nil, err
	}
	for _, p := range pathways {
		out = append(out, PathwaySummary{
			Pathway:      p,
			RiskTitle:    risks[p.RiskID].Title,
			BarrierCount: guards[p.ID],
			Drilldowns:   drills[p.ID],
		})
	}
	return out, nil
}

func (m *riskManager) PreventiveMeasures(ctx context.Context, filter MeasureFilter) ([]Measure, error) {
	return m.measuresOn(ctx, SideThreat, filter)
}

func (m *riskManager) MitigatingMeasures(ctx context.Context, filter MeasureFilter) ([]Measure, error) {
	return m.measuresOn(ctx, SideConsequence, filter)
}

func (m *riskManager) measuresOn(ctx context.Context, side Side, filter MeasureFilter) ([]Measure, error) {
	pathways, err := m.pathwaysAsked(ctx, side, filter)
	if err != nil {
		return nil, err
	}
	if len(pathways) == 0 {
		return []Measure{}, nil
	}
	byID := make(map[string]Pathway, len(pathways))
	ids := make([]string, len(pathways))
	for i, p := range pathways {
		byID[p.ID] = p
		ids[i] = p.ID
	}
	barriers, err := m.asmBarriersOn(ctx, ids)
	if err != nil {
		return nil, err
	}
	placed, err := m.asmPlace(ctx, barriers)
	if err != nil {
		return nil, err
	}
	measures, err := m.asmMeasures(ctx, placed, byID)
	if err != nil {
		return nil, err
	}
	return narrowed(measures, filter.Query, capped(filter.Limit)), nil
}

func (m *riskManager) pathwaysAsked(ctx context.Context, side Side, filter MeasureFilter) ([]Pathway, error) {
	if filter.PathwayID != "" {
		p, err := m.GetPathway(ctx, filter.PathwayID)
		if err != nil {
			return nil, err
		}
		if p.Side != side {
			return nil, fmt.Errorf("%w: the pathway %q is a %s, not a %s", ErrInvalid, p.Title, p.Side, side)
		}
		if filter.RiskID != "" && p.RiskID != filter.RiskID {
			return nil, nil
		}
		return []Pathway{p}, nil
	}
	if filter.RiskID != "" {
		if _, err := m.GetRisk(ctx, filter.RiskID); err != nil {
			return nil, err
		}
	}
	q := m.q(ctx, rolePathway).Where("side = ?", string(side))
	if filter.RiskID != "" {
		q = q.Where("risk_id = ?", filter.RiskID)
	}
	return listOf[Pathway](m, q.Order("created_at"), rolePathway)
}

func narrowed(measures []Measure, query string, limit int) []Measure {
	out := make([]Measure, 0, len(measures))
	for _, ms := range measures {
		if !mentions(query, ms.Pathway.Title, ms.Pathway.Scenario, ms.Control.Name, ms.Control.Description) {
			continue
		}
		out = append(out, ms)
		if len(out) == limit {
			break
		}
	}
	return out
}

func (m *riskManager) Reliance(ctx context.Context, controlID string) ([]Measure, error) {
	if _, err := m.GetControl(ctx, controlID); err != nil {
		return nil, err
	}
	barriers, err := m.asmBarriersOfControl(ctx, controlID)
	if err != nil {
		return nil, err
	}
	if len(barriers) == 0 {
		return []Measure{}, nil
	}
	pathwayIDs := make([]string, len(barriers))
	for i, b := range barriers {
		pathwayIDs[i] = b.PathwayID
	}
	pathways, err := m.asmPathwaysByID(ctx, asmUnique(pathwayIDs))
	if err != nil {
		return nil, err
	}
	placed, err := m.asmPlace(ctx, barriers)
	if err != nil {
		return nil, err
	}
	return m.asmMeasures(ctx, placed, pathways)
}
