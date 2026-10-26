package riskmanager

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-riskmanager/models"
)

func (m *riskManager) CreateRisk(ctx context.Context, r Risk) (Risk, error) {
	r.Status = models.StatusDraft
	if err := m.checks(roleRisk, r); err != nil {
		return Risk{}, err
	}
	if err := m.parentIsLive(ctx, r.ParentPathwayID); err != nil {
		return Risk{}, err
	}
	r.ID = newID()
	r.CreatedAt = m.now()
	r.UpdatedAt = r.CreatedAt
	if err := m.insert(ctx, roleRisk, r); err != nil {
		return Risk{}, err
	}
	return r, nil
}

func (m *riskManager) parentIsLive(ctx context.Context, pathwayID string) error {
	if pathwayID == "" {
		return nil
	}
	p, err := m.referredPathway(ctx, pathwayID)
	if err != nil {
		return err
	}
	_, err = m.referredRisk(ctx, p.RiskID)
	return err
}

func (m *riskManager) GetRisk(ctx context.Context, id string) (Risk, error) {
	var out Risk
	err := m.take(m.q(ctx, roleRisk).Where("id = ?", id), roleRisk, &out, ErrRiskNotFound)
	return out, err
}

func (m *riskManager) ListRisks(ctx context.Context, filter RiskFilter) ([]Risk, error) {
	q := m.q(ctx, roleRisk)
	if filter.Status != "" {
		q = q.Where("status = ?", string(filter.Status))
	}
	if filter.Owner != "" {
		q = q.Where("owner = ?", filter.Owner)
	}
	if filter.ParentPathwayID != "" {
		q = q.Where("parent_pathway_id = ?", filter.ParentPathwayID)
	}
	q = containing(q, filter.Query, "title", "top_event", "hazard")
	return listOf[Risk](m, q.Order("created_at").Limit(capped(filter.Limit)), roleRisk)
}

func (m *riskManager) UpdateRisk(ctx context.Context, r Risk) (Risk, error) {
	held, err := m.liveRisk(ctx, r.ID)
	if err != nil {
		return Risk{}, err
	}

	held.Title = r.Title
	held.TopEvent = r.TopEvent
	held.Hazard = r.Hazard
	held.Owner = r.Owner
	held.Doc = r.Doc
	held.UpdatedAt = m.now()

	if err := m.checks(roleRisk, held); err != nil {
		return Risk{}, err
	}
	if held.Status != models.StatusDraft && held.Owner == "" {
		return Risk{}, fmt.Errorf("%w: a risk that is %s cannot be left without an owner", ErrInvalid, held.Status)
	}
	if err := m.update(ctx, roleRisk, held.ID, held); err != nil {
		return Risk{}, err
	}
	return held, nil
}

func (m *riskManager) SetRiskStatus(ctx context.Context, id string, next Status) (Risk, error) {
	held, err := m.GetRisk(ctx, id)
	if err != nil {
		return Risk{}, err
	}
	if err := m.checksField(roleRisk, "status", string(next)); err != nil {
		return Risk{}, err
	}
	if !held.Status.CanTransitionTo(next) {
		return Risk{}, fmt.Errorf("%w: a risk that is %s cannot become %s", ErrInvalid, held.Status, next)
	}
	if err := m.gate(ctx, held, next); err != nil {
		return Risk{}, err
	}

	held.Status = next
	held.UpdatedAt = m.now()
	if err := m.update(ctx, roleRisk, held.ID, held); err != nil {
		return Risk{}, err
	}
	return held, nil
}

func (m *riskManager) gate(ctx context.Context, held Risk, next Status) error {
	switch {
	case held.Status == models.StatusDraft && next == models.StatusOpen && held.Owner == "":
		return fmt.Errorf("%w: a risk needs an owner before it can open", ErrInvalid)
	case next == models.StatusAccepted:
		rated, err := m.count(ctx, roleAssessment, "risk_id = ? AND kind = ?", held.ID, string(models.AssessmentResidual))
		if err != nil {
			return err
		}
		if rated == 0 {
			return fmt.Errorf("%w: a risk cannot be accepted before its residual exposure has been rated", ErrInvalid)
		}
	}
	return nil
}
