package riskmanager

import (
	"context"
	"fmt"
)

func (m *riskManager) RecordEvidence(ctx context.Context, e Evidence) (Evidence, error) {
	if e.RecordedAt == "" {
		e.RecordedAt = m.now()
	}
	if err := m.checks(roleEvidence, e); err != nil {
		return Evidence{}, err
	}
	if !isRFC3339(e.RecordedAt) {
		return Evidence{}, fmt.Errorf("%w: evidence.recorded_at %q is not an RFC 3339 instant", ErrInvalid, e.RecordedAt)
	}
	barrier, err := m.referredBarrier(ctx, e.BarrierID)
	if err != nil {
		return Evidence{}, err
	}
	if _, err := m.referredRisk(ctx, barrier.RiskID); err != nil {
		return Evidence{}, err
	}

	e.ID = newID()
	e.CreatedAt = m.now()
	if err := m.insert(ctx, roleEvidence, e); err != nil {
		return Evidence{}, err
	}
	return e, nil
}

func (m *riskManager) ListEvidence(ctx context.Context, filter EvidenceFilter) ([]Evidence, error) {
	q := m.q(ctx, roleEvidence)
	if filter.BarrierID != "" {
		q = q.Where("barrier_id = ?", filter.BarrierID)
	}
	if isRFC3339(filter.From) {
		q = q.Where("recorded_at >= ?", filter.From)
	}
	if isRFC3339(filter.To) {
		q = q.Where("recorded_at <= ?", filter.To)
	}
	return listOf[Evidence](m, q.Order("recorded_at").Order("created_at").Limit(capped(filter.Limit)), roleEvidence)
}
