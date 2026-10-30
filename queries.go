package riskmanager

import (
	"context"
	"fmt"

	"gorm.io/gorm"
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
	inScope, err := m.measureScope(ctx, side, filter)
	if err != nil || !inScope {
		return []Measure{}, err
	}
	barriers, err := listOf[Barrier](m, m.rankedBarriersOn(ctx, side, filter), roleBarrier)
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

func (m *riskManager) rankedBarriersOn(ctx context.Context, side Side, filter MeasureFilter) *gorm.DB {
	barriers, pathways := m.st.Table(roleBarrier), m.st.Table(rolePathway)
	risks, controls := m.st.Table(roleRisk), m.st.Table(roleControl)
	q := m.db.WithContext(ctx).Table(barriers).Select(barriers+".*").
		Joins("JOIN "+pathways+" ON "+pathways+".id = "+barriers+".pathway_id").
		Joins("LEFT JOIN "+risks+" ON "+risks+".id = "+pathways+".risk_id").
		Joins("LEFT JOIN "+controls+" ON "+controls+".id = "+barriers+".control_id").
		Where(pathways+".side = ?", string(side))
	if filter.RiskID != "" {
		q = q.Where(pathways+".risk_id = ?", filter.RiskID)
	}
	if filter.PathwayID != "" {
		q = q.Where(pathways+".id = ?", filter.PathwayID)
	}
	q = containing(q, filter.Query, pathways+".title", pathways+".scenario", controls+".name", controls+".description")
	return q.Order(risks + ".created_at").Order(risks + ".id").
		Order(pathways + ".created_at").Order(pathways + ".id").
		Order(barriers + `."position"`).Order(barriers + ".created_at").
		Limit(capped(filter.Limit))
}

func (m *riskManager) measureScope(ctx context.Context, side Side, filter MeasureFilter) (bool, error) {
	if filter.PathwayID != "" {
		p, err := m.GetPathway(ctx, filter.PathwayID)
		if err != nil {
			return false, err
		}
		if p.Side != side {
			return false, fmt.Errorf("%w: the pathway %q is a %s, not a %s", ErrInvalid, p.Title, p.Side, side)
		}
		return filter.RiskID == "" || p.RiskID == filter.RiskID, nil
	}
	if filter.RiskID != "" {
		if _, err := m.GetRisk(ctx, filter.RiskID); err != nil {
			return false, err
		}
	}
	return true, nil
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
