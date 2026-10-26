package riskmanager

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-riskmanager/models"
)

func (m *riskManager) PlaceBarrier(ctx context.Context, b Barrier) (Barrier, error) {
	if b.Design == "" {
		b.Design = models.JudgementUnassessed
	}
	if b.Coverage == "" {
		b.Coverage = models.JudgementUnassessed
	}
	if b.Maturity == "" {
		b.Maturity = models.MaturityProposed
	}
	if err := m.checks(roleBarrier, b); err != nil {
		return Barrier{}, err
	}
	if b.Maturity.NeedsEvidence() {
		return Barrier{}, fmt.Errorf("%w: a barrier cannot be placed as %s, because no evidence can exist yet", ErrInvalid, b.Maturity)
	}
	pathway, err := m.referredPathway(ctx, b.PathwayID)
	if err != nil {
		return Barrier{}, err
	}
	if _, err := m.referredRisk(ctx, pathway.RiskID); err != nil {
		return Barrier{}, err
	}
	if _, err := m.referredControl(ctx, b.ControlID); err != nil {
		return Barrier{}, err
	}
	if err := m.stands(ctx, b.PathwayID, b.ControlID); err != nil {
		return Barrier{}, err
	}

	b.RiskID = pathway.RiskID
	b.ID = newID()
	b.CreatedAt = m.now()
	b.UpdatedAt = b.CreatedAt
	if err := m.insert(ctx, roleBarrier, b); err != nil {
		return Barrier{}, err
	}
	return b, nil
}

func (m *riskManager) stands(ctx context.Context, pathwayID, controlID string) error {
	placed, err := m.count(ctx, roleBarrier, "pathway_id = ? AND control_id = ?", pathwayID, controlID)
	if err != nil {
		return err
	}
	if placed > 0 {
		return fmt.Errorf("%w: that control already stands on that pathway", ErrConflict)
	}
	return nil
}

func (m *riskManager) ListBarriers(ctx context.Context, filter BarrierFilter) ([]Barrier, error) {
	q := m.q(ctx, roleBarrier)
	if filter.RiskID != "" {
		q = q.Where("risk_id = ?", filter.RiskID)
	}
	if filter.PathwayID != "" {
		q = q.Where("pathway_id = ?", filter.PathwayID)
	}
	if filter.ControlID != "" {
		q = q.Where("control_id = ?", filter.ControlID)
	}
	return listOf[Barrier](m, q.Order("position").Order("created_at").Limit(capped(filter.Limit)), roleBarrier)
}

func (m *riskManager) UpdateBarrier(ctx context.Context, change BarrierChange) (Barrier, error) {
	held, err := m.barrier(ctx, change.ID)
	if err != nil {
		return Barrier{}, err
	}
	if _, err := m.liveRisk(ctx, held.RiskID); err != nil {
		return Barrier{}, err
	}

	applyBarrierChange(&held, change)
	held.UpdatedAt = m.now()

	if err := m.checks(roleBarrier, held); err != nil {
		return Barrier{}, err
	}
	if err := m.maturityHasEvidence(ctx, held); err != nil {
		return Barrier{}, err
	}
	if err := m.update(ctx, roleBarrier, held.ID, held); err != nil {
		return Barrier{}, err
	}
	return held, nil
}

func applyBarrierChange(held *Barrier, change BarrierChange) {
	if change.Position != nil {
		held.Position = *change.Position
	}
	if change.Design != nil {
		held.Design = *change.Design
	}
	if change.Coverage != nil {
		held.Coverage = *change.Coverage
	}
	if change.Maturity != nil {
		held.Maturity = *change.Maturity
	}
	if change.Notes != nil {
		held.Notes = *change.Notes
	}
}

func (m *riskManager) maturityHasEvidence(ctx context.Context, b Barrier) error {
	if !b.Maturity.NeedsEvidence() {
		return nil
	}
	proofs, err := m.count(ctx, roleEvidence, "barrier_id = ?", b.ID)
	if err != nil {
		return err
	}
	if proofs == 0 {
		return fmt.Errorf("%w: a barrier cannot be %s with no evidence on record", ErrInvalid, b.Maturity)
	}
	return nil
}

func (m *riskManager) RemoveBarrier(ctx context.Context, id string) error {
	held, err := m.barrier(ctx, id)
	if err != nil {
		return err
	}
	if _, err := m.liveRisk(ctx, held.RiskID); err != nil {
		return err
	}
	if proofs, err := m.count(ctx, roleEvidence, "barrier_id = ?", id); err != nil {
		return err
	} else if proofs > 0 {
		return fmt.Errorf("%w: %d piece(s) of evidence are recorded against this barrier, so it cannot be removed", ErrConflict, proofs)
	}
	if work, err := m.count(ctx, roleAction, "barrier_id = ?", id); err != nil {
		return err
	} else if work > 0 {
		return fmt.Errorf("%w: %d action(s) refer to this barrier, so it cannot be removed", ErrConflict, work)
	}
	return m.remove(ctx, roleBarrier, id)
}
