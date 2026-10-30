package riskmanager

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-riskmanager/models"
)

func (m *riskManager) CreateAction(ctx context.Context, a Action) (Action, error) {
	a.Status = models.ActionOpen
	a.ClosedAt = ""
	if err := m.checks(roleAction, a); err != nil {
		return Action{}, err
	}
	if err := m.ownerKnown(ctx, "action's owner", a.Owner); err != nil {
		return Action{}, err
	}
	if err := dueDateIsAnInstant(a.DueAt); err != nil {
		return Action{}, err
	}
	if _, err := m.referredRisk(ctx, a.RiskID); err != nil {
		return Action{}, err
	}
	if err := m.barrierBelongsTo(ctx, a.BarrierID, a.RiskID); err != nil {
		return Action{}, err
	}

	a.ID = newID()
	a.CreatedAt = m.now()
	a.UpdatedAt = a.CreatedAt
	if err := m.insert(ctx, roleAction, a); err != nil {
		return Action{}, err
	}
	return a, nil
}

func dueDateIsAnInstant(dueAt string) error {
	if dueAt != "" && !isRFC3339(dueAt) {
		return fmt.Errorf("%w: action.due_at %q is not an RFC 3339 instant", ErrInvalid, dueAt)
	}
	return nil
}

func (m *riskManager) barrierBelongsTo(ctx context.Context, barrierID, riskID string) error {
	if barrierID == "" {
		return nil
	}
	b, err := m.referredBarrier(ctx, barrierID)
	if err != nil {
		return err
	}
	if b.RiskID != riskID {
		return fmt.Errorf("%w: the barrier %q does not belong to the risk %q", ErrInvalidReference, barrierID, riskID)
	}
	return nil
}

func (m *riskManager) action(ctx context.Context, id string) (Action, error) {
	var out Action
	err := m.take(m.q(ctx, roleAction).Where("id = ?", id), roleAction, &out, ErrActionNotFound)
	return out, err
}

func (m *riskManager) ListActions(ctx context.Context, filter ActionFilter) ([]Action, error) {
	q := m.q(ctx, roleAction)
	if filter.RiskID != "" {
		q = q.Where("risk_id = ?", filter.RiskID)
	}
	if filter.BarrierID != "" {
		q = q.Where("barrier_id = ?", filter.BarrierID)
	}
	if filter.Owner != "" {
		q = q.Where("owner = ?", filter.Owner)
	}
	if filter.Status != "" {
		q = q.Where("status = ?", string(filter.Status))
	}
	return listOf[Action](m, q.Order("created_at").Limit(capped(filter.Limit)), roleAction)
}

func (m *riskManager) UpdateAction(ctx context.Context, a Action) (Action, error) {
	held, err := m.action(ctx, a.ID)
	if err != nil {
		return Action{}, err
	}
	if _, err := m.liveRisk(ctx, held.RiskID); err != nil {
		return Action{}, err
	}
	if held.Status.Closed() {
		return Action{}, fmt.Errorf("%w: an action that is %s takes no further changes", ErrConflict, held.Status)
	}

	held.Title = a.Title
	held.Description = a.Description
	priorOwner := held.Owner
	held.Owner = a.Owner
	held.DueAt = a.DueAt
	held.UpdatedAt = m.now()

	if err := m.checks(roleAction, held); err != nil {
		return Action{}, err
	}
	if err := m.ownerChanged(ctx, "action's owner", priorOwner, held.Owner); err != nil {
		return Action{}, err
	}
	if err := dueDateIsAnInstant(held.DueAt); err != nil {
		return Action{}, err
	}
	if err := m.update(ctx, roleAction, held.ID, held); err != nil {
		return Action{}, err
	}
	return held, nil
}

func (m *riskManager) SetActionStatus(ctx context.Context, id string, next ActionStatus) (Action, error) {
	held, err := m.action(ctx, id)
	if err != nil {
		return Action{}, err
	}
	if _, err := m.liveRisk(ctx, held.RiskID); err != nil {
		return Action{}, err
	}
	if err := m.checksField(roleAction, "status", string(next)); err != nil {
		return Action{}, err
	}
	if !held.Status.CanTransitionTo(next) {
		return Action{}, fmt.Errorf("%w: an action that is %s cannot become %s", ErrInvalid, held.Status, next)
	}

	held.Status = next
	held.UpdatedAt = m.now()
	if next.Closed() {
		held.ClosedAt = held.UpdatedAt
	}
	if err := m.update(ctx, roleAction, held.ID, held); err != nil {
		return Action{}, err
	}
	return held, nil
}
