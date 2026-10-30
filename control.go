package riskmanager

import (
	"context"
	"fmt"
)

func (m *riskManager) CreateControl(ctx context.Context, c Control) (Control, error) {
	if err := m.checks(roleControl, c); err != nil {
		return Control{}, err
	}
	if err := m.ownerKnown(ctx, "control's owner", c.Owner); err != nil {
		return Control{}, err
	}
	if err := m.nameIsFree(ctx, c.Name); err != nil {
		return Control{}, err
	}
	c.ID = newID()
	c.CreatedAt = m.now()
	c.UpdatedAt = c.CreatedAt
	if err := m.insert(ctx, roleControl, c); err != nil {
		return Control{}, err
	}
	return c, nil
}

func (m *riskManager) nameIsFree(ctx context.Context, name string) error {
	taken, err := m.count(ctx, roleControl, "name = ?", name)
	if err != nil {
		return err
	}
	if taken > 0 {
		return fmt.Errorf("%w: a control called %q already exists", ErrConflict, name)
	}
	return nil
}

func (m *riskManager) GetControl(ctx context.Context, id string) (Control, error) {
	var out Control
	err := m.take(m.q(ctx, roleControl).Where("id = ?", id), roleControl, &out, ErrControlNotFound)
	return out, err
}

func (m *riskManager) ListControls(ctx context.Context, filter ControlFilter) ([]Control, error) {
	if filter.Failing {
		return m.failingControls(ctx, filter)
	}
	q := m.q(ctx, roleControl)
	if filter.Owner != "" {
		q = q.Where("owner = ?", filter.Owner)
	}
	return listOf[Control](m, q.Order("name").Limit(capped(filter.Limit)), roleControl)
}

func (m *riskManager) UpdateControl(ctx context.Context, c Control) (Control, error) {
	held, err := m.GetControl(ctx, c.ID)
	if err != nil {
		return Control{}, err
	}
	if c.Name != held.Name {
		if err := m.nameIsFree(ctx, c.Name); err != nil {
			return Control{}, err
		}
	}

	held.Name = c.Name
	held.Description = c.Description
	priorOwner := held.Owner
	held.Owner = c.Owner
	held.Doc = c.Doc
	held.UpdatedAt = m.now()

	if err := m.checks(roleControl, held); err != nil {
		return Control{}, err
	}
	if err := m.ownerChanged(ctx, "control's owner", priorOwner, held.Owner); err != nil {
		return Control{}, err
	}
	if err := m.update(ctx, roleControl, held.ID, held); err != nil {
		return Control{}, err
	}
	return held, nil
}

type failingCount struct {
	ControlID string
	Failing   int
}

func (m *riskManager) failingControls(ctx context.Context, filter ControlFilter) ([]Control, error) {
	controls, barriers := m.st.Table(roleControl), m.st.Table(roleBarrier)
	risks, evidence := m.st.Table(roleRisk), m.st.Table(roleEvidence)
	q := m.db.WithContext(ctx).Table(evidence+" AS latest").
		Select("b.control_id AS control_id, COUNT(*) AS failing").
		Joins("JOIN "+barriers+" b ON b.id = latest.barrier_id").
		Joins("JOIN "+risks+" r ON r.id = b.risk_id").
		Joins("JOIN "+controls+" c ON c.id = b.control_id").
		Where("r.status <> ?", string(StatusRetired)).
		Where("latest.outcome <> ?", string(OutcomePass)).
		Where("NOT EXISTS (SELECT 1 FROM " + evidence + " later WHERE later.barrier_id = latest.barrier_id AND " +
			"(later.recorded_at > latest.recorded_at OR (later.recorded_at = latest.recorded_at AND later.created_at > latest.created_at)))")
	if filter.Owner != "" {
		q = q.Where("c.owner = ?", filter.Owner)
	}
	var counts []failingCount
	if err := q.Group("b.control_id, c.name").Order("failing DESC").Order("c.name").Limit(capped(filter.Limit)).Scan(&counts).Error; err != nil {
		return nil, err
	}
	out := make([]Control, 0, len(counts))
	if len(counts) == 0 {
		return out, nil
	}
	ids := make([]string, len(counts))
	for i, c := range counts {
		ids[i] = c.ControlID
	}
	held, err := listOf[Control](m, m.q(ctx, roleControl).Where("id IN ?", ids), roleControl)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]Control, len(held))
	for _, c := range held {
		byID[c.ID] = c
	}
	for _, c := range counts {
		control := byID[c.ControlID]
		control.FailingBarriers = c.Failing
		out = append(out, control)
	}
	return out, nil
}
