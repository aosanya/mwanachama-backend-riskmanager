package riskmanager

import (
	"context"
	"fmt"
)

func (m *riskManager) CreatePathway(ctx context.Context, p Pathway) (Pathway, error) {
	if err := m.checks(rolePathway, p); err != nil {
		return Pathway{}, err
	}
	if _, err := m.referredRisk(ctx, p.RiskID); err != nil {
		return Pathway{}, err
	}
	p.ID = newID()
	p.CreatedAt = m.now()
	p.UpdatedAt = p.CreatedAt
	if err := m.insert(ctx, rolePathway, p); err != nil {
		return Pathway{}, err
	}
	return p, nil
}

func (m *riskManager) GetPathway(ctx context.Context, id string) (Pathway, error) {
	var out Pathway
	err := m.take(m.q(ctx, rolePathway).Where("id = ?", id), rolePathway, &out, ErrPathwayNotFound)
	return out, err
}

func (m *riskManager) UpdatePathway(ctx context.Context, p Pathway) (Pathway, error) {
	held, err := m.GetPathway(ctx, p.ID)
	if err != nil {
		return Pathway{}, err
	}
	if _, err := m.liveRisk(ctx, held.RiskID); err != nil {
		return Pathway{}, err
	}

	held.Title = p.Title
	held.Scenario = p.Scenario
	held.Doc = p.Doc
	held.UpdatedAt = m.now()

	if err := m.checks(rolePathway, held); err != nil {
		return Pathway{}, err
	}
	if err := m.update(ctx, rolePathway, held.ID, held); err != nil {
		return Pathway{}, err
	}
	return held, nil
}

func (m *riskManager) DeletePathway(ctx context.Context, id string) error {
	held, err := m.GetPathway(ctx, id)
	if err != nil {
		return err
	}
	if _, err := m.liveRisk(ctx, held.RiskID); err != nil {
		return err
	}
	if guards, err := m.count(ctx, roleBarrier, "pathway_id = ?", id); err != nil {
		return err
	} else if guards > 0 {
		return fmt.Errorf("%w: %d barrier(s) stand on %q, so it cannot be deleted", ErrConflict, guards, held.Title)
	}
	if drilled, err := m.count(ctx, roleRisk, "parent_pathway_id = ?", id); err != nil {
		return err
	} else if drilled > 0 {
		return fmt.Errorf("%w: %d risk(s) drill into %q, so it cannot be deleted", ErrConflict, drilled, held.Title)
	}
	return m.remove(ctx, rolePathway, id)
}
