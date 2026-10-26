package riskmanager

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-riskmanager/models"
)

func (m *riskManager) RecordAssessment(ctx context.Context, a Assessment) (Assessment, error) {
	if a.AssessedAt == "" {
		a.AssessedAt = m.now()
	}
	if err := m.checks(roleAssessment, a); err != nil {
		return Assessment{}, err
	}
	if !a.OnTheScale() {
		return Assessment{}, fmt.Errorf("%w: likelihood %d and impact %d must each be between %d and %d",
			ErrInvalid, a.Likelihood, a.Impact, models.ScaleMin, models.ScaleMax)
	}
	if !isRFC3339(a.AssessedAt) {
		return Assessment{}, fmt.Errorf("%w: assessment.assessed_at %q is not an RFC 3339 instant", ErrInvalid, a.AssessedAt)
	}
	if _, err := m.referredRisk(ctx, a.RiskID); err != nil {
		return Assessment{}, err
	}

	a.ID = newID()
	a.CreatedAt = m.now()
	if err := m.insert(ctx, roleAssessment, a); err != nil {
		return Assessment{}, err
	}
	return a, nil
}

func (m *riskManager) ListAssessments(ctx context.Context, filter AssessmentFilter) ([]Assessment, error) {
	q := m.q(ctx, roleAssessment)
	if filter.RiskID != "" {
		q = q.Where("risk_id = ?", filter.RiskID)
	}
	if filter.Kind != "" {
		q = q.Where("kind = ?", string(filter.Kind))
	}
	return listOf[Assessment](m, q.Order("assessed_at").Order("created_at").Limit(capped(filter.Limit)), roleAssessment)
}
