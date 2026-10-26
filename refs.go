package riskmanager

import (
	"context"
	"errors"
	"fmt"
)

func missing(err, notFound error, what, id string) error {
	if errors.Is(err, notFound) {
		return fmt.Errorf("%w: no %s %q", ErrInvalidReference, what, id)
	}
	return err
}

func (m *riskManager) liveRisk(ctx context.Context, id string) (Risk, error) {
	r, err := m.GetRisk(ctx, id)
	if err != nil {
		return Risk{}, err
	}
	if r.Status.Frozen() {
		return Risk{}, fmt.Errorf("%w: the risk %q is retired and no longer takes changes", ErrConflict, r.Title)
	}
	return r, nil
}

func (m *riskManager) referredRisk(ctx context.Context, id string) (Risk, error) {
	r, err := m.liveRisk(ctx, id)
	return r, missing(err, ErrRiskNotFound, "risk", id)
}

func (m *riskManager) referredPathway(ctx context.Context, id string) (Pathway, error) {
	p, err := m.GetPathway(ctx, id)
	return p, missing(err, ErrPathwayNotFound, "pathway", id)
}

func (m *riskManager) referredControl(ctx context.Context, id string) (Control, error) {
	c, err := m.GetControl(ctx, id)
	return c, missing(err, ErrControlNotFound, "control", id)
}

func (m *riskManager) barrier(ctx context.Context, id string) (Barrier, error) {
	var out Barrier
	err := m.take(m.q(ctx, roleBarrier).Where("id = ?", id), roleBarrier, &out, ErrBarrierNotFound)
	return out, err
}

func (m *riskManager) referredBarrier(ctx context.Context, id string) (Barrier, error) {
	b, err := m.barrier(ctx, id)
	return b, missing(err, ErrBarrierNotFound, "barrier", id)
}
