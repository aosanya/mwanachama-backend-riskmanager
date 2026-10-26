package riskmanager

import (
	"context"
	"fmt"
)

func (m *riskManager) CreateControl(ctx context.Context, c Control) (Control, error) {
	if err := m.checks(roleControl, c); err != nil {
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
	held.Owner = c.Owner
	held.Doc = c.Doc
	held.UpdatedAt = m.now()

	if err := m.checks(roleControl, held); err != nil {
		return Control{}, err
	}
	if err := m.update(ctx, roleControl, held.ID, held); err != nil {
		return Control{}, err
	}
	return held, nil
}
