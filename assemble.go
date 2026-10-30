package riskmanager

import (
	"cmp"
	"context"
	"slices"

	"github.com/aosanya/mwanachama-backend-riskmanager/models"
)

const asmEverything = 100000

func asmUnique(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func (m *riskManager) asmBarriersOn(ctx context.Context, pathwayIDs []string) ([]Barrier, error) {
	if len(pathwayIDs) == 0 {
		return []Barrier{}, nil
	}
	q := m.q(ctx, roleBarrier).Where("pathway_id IN ?", pathwayIDs).Order(`"position"`).Order("created_at")
	return listOf[Barrier](m, q, roleBarrier)
}

func (m *riskManager) asmBarriersOfControl(ctx context.Context, controlID string) ([]Barrier, error) {
	q := m.q(ctx, roleBarrier).Where("control_id = ?", controlID).Order("created_at")
	return listOf[Barrier](m, q, roleBarrier)
}

func (m *riskManager) asmPlace(ctx context.Context, barriers []Barrier) ([]models.Placed, error) {
	out := make([]models.Placed, 0, len(barriers))
	if len(barriers) == 0 {
		return out, nil
	}
	controlIDs := make([]string, len(barriers))
	barrierIDs := make([]string, len(barriers))
	for i, b := range barriers {
		controlIDs[i] = b.ControlID
		barrierIDs[i] = b.ID
	}

	controls, err := listOf[Control](m, m.q(ctx, roleControl).Where("id IN ?", asmUnique(controlIDs)), roleControl)
	if err != nil {
		return nil, err
	}
	controlByID := make(map[string]Control, len(controls))
	for _, c := range controls {
		controlByID[c.ID] = c
	}

	evidence, err := listOf[Evidence](m, m.q(ctx, roleEvidence).Where("barrier_id IN ?", barrierIDs).Order("recorded_at").Order("created_at"), roleEvidence)
	if err != nil {
		return nil, err
	}
	evidenceByBarrier := make(map[string][]Evidence, len(barriers))
	for _, e := range evidence {
		evidenceByBarrier[e.BarrierID] = append(evidenceByBarrier[e.BarrierID], e)
	}

	for _, b := range barriers {
		out = append(out, placedOf(b, controlByID[b.ControlID], evidenceByBarrier[b.ID]))
	}
	return out, nil
}

func placedOf(b Barrier, c Control, held []Evidence) models.Placed {
	placed := models.Placed{Barrier: b, Control: c, EvidenceCount: len(held)}
	if len(held) > 0 {
		latest := held[len(held)-1]
		placed.Latest = &latest
	}
	return placed
}

func (m *riskManager) asmPathwaysByID(ctx context.Context, ids []string) (map[string]Pathway, error) {
	out := make(map[string]Pathway, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	found, err := listOf[Pathway](m, m.q(ctx, rolePathway).Where("id IN ?", ids), rolePathway)
	if err != nil {
		return nil, err
	}
	for _, p := range found {
		out[p.ID] = p
	}
	return out, nil
}

func (m *riskManager) asmRisksByID(ctx context.Context, ids []string) (map[string]Risk, error) {
	out := make(map[string]Risk, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	found, err := listOf[Risk](m, m.q(ctx, roleRisk).Where("id IN ?", ids), roleRisk)
	if err != nil {
		return nil, err
	}
	for _, r := range found {
		out[r.ID] = r
	}
	return out, nil
}

func (m *riskManager) asmCountBy(ctx context.Context, role, column string, ids []string) (map[string]int, error) {
	out := make(map[string]int, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var held []string
	if err := m.q(ctx, role).Where(column+" IN ?", ids).Pluck(column, &held).Error; err != nil {
		return nil, err
	}
	for _, id := range held {
		out[id]++
	}
	return out, nil
}

func (m *riskManager) asmMeasures(ctx context.Context, placed []models.Placed, pathways map[string]Pathway) ([]Measure, error) {
	riskIDs := make([]string, 0, len(pathways))
	for _, p := range pathways {
		riskIDs = append(riskIDs, p.RiskID)
	}
	risks, err := m.asmRisksByID(ctx, asmUnique(riskIDs))
	if err != nil {
		return nil, err
	}

	out := make([]Measure, 0, len(placed))
	for _, p := range placed {
		pathway := pathways[p.Barrier.PathwayID]
		out = append(out, Measure{
			RiskID:    pathway.RiskID,
			RiskTitle: risks[pathway.RiskID].Title,
			Pathway:   pathway,
			Placed:    p,
			Standing:  p.Verdict(m.clock()),
		})
	}
	slices.SortStableFunc(out, func(a, b Measure) int {
		return cmp.Or(
			cmp.Compare(risks[a.RiskID].CreatedAt, risks[b.RiskID].CreatedAt),
			cmp.Compare(a.RiskID, b.RiskID),
			cmp.Compare(a.Pathway.CreatedAt, b.Pathway.CreatedAt),
			cmp.Compare(a.Pathway.ID, b.Pathway.ID),
			cmp.Compare(a.Barrier.Position, b.Barrier.Position),
			cmp.Compare(a.Barrier.CreatedAt, b.Barrier.CreatedAt),
		)
	})
	return out, nil
}
