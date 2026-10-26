package riskmanager

import (
	"context"

	"github.com/aosanya/mwanachama-backend-riskmanager/models"
)

func (m *riskManager) Bowtie(ctx context.Context, riskID string) (models.Bowtie, error) {
	risk, err := m.GetRisk(ctx, riskID)
	if err != nil {
		return models.Bowtie{}, err
	}
	pathways, err := listOf[Pathway](m, m.q(ctx, rolePathway).Where("risk_id = ?", riskID).Order("created_at"), rolePathway)
	if err != nil {
		return models.Bowtie{}, err
	}
	pathwayIDs := make([]string, len(pathways))
	for i, p := range pathways {
		pathwayIDs[i] = p.ID
	}
	arms, err := m.armsOf(ctx, pathways, pathwayIDs)
	if err != nil {
		return models.Bowtie{}, err
	}

	out := models.Bowtie{Risk: risk, Threats: []models.Arm{}, Consequences: []models.Arm{}}
	for _, arm := range arms {
		if arm.Pathway.Side == SideThreat {
			out.Threats = append(out.Threats, arm)
		} else {
			out.Consequences = append(out.Consequences, arm)
		}
	}
	if err := m.rateInto(ctx, &out); err != nil {
		return models.Bowtie{}, err
	}
	actions, err := m.ListActions(ctx, ActionFilter{RiskID: riskID, Limit: asmEverything})
	if err != nil {
		return models.Bowtie{}, err
	}
	out.Actions = actions
	if out.Actions == nil {
		out.Actions = []Action{}
	}
	return out, nil
}

func (m *riskManager) armsOf(ctx context.Context, pathways []Pathway, pathwayIDs []string) ([]models.Arm, error) {
	barriers, err := m.asmBarriersOn(ctx, pathwayIDs)
	if err != nil {
		return nil, err
	}
	placed, err := m.asmPlace(ctx, barriers)
	if err != nil {
		return nil, err
	}
	placedOn := map[string][]models.Placed{}
	for _, p := range placed {
		placedOn[p.Barrier.PathwayID] = append(placedOn[p.Barrier.PathwayID], p)
	}

	drilled := map[string][]Risk{}
	if len(pathwayIDs) > 0 {
		children, err := listOf[Risk](m, m.q(ctx, roleRisk).Where("parent_pathway_id IN ?", pathwayIDs).Order("created_at"), roleRisk)
		if err != nil {
			return nil, err
		}
		for _, r := range children {
			drilled[r.ParentPathwayID] = append(drilled[r.ParentPathwayID], r)
		}
	}

	arms := make([]models.Arm, 0, len(pathways))
	for _, p := range pathways {
		arm := models.Arm{Pathway: p, Barriers: placedOn[p.ID], Drilldowns: drilled[p.ID]}
		if arm.Barriers == nil {
			arm.Barriers = []models.Placed{}
		}
		if arm.Drilldowns == nil {
			arm.Drilldowns = []Risk{}
		}
		arms = append(arms, arm)
	}
	return arms, nil
}

func (m *riskManager) rateInto(ctx context.Context, b *models.Bowtie) error {
	held, err := m.ListAssessments(ctx, AssessmentFilter{RiskID: b.Risk.ID, Limit: asmEverything})
	if err != nil {
		return err
	}
	latest := map[AssessmentKind]Assessment{}
	for _, a := range held {
		seen, ok := latest[a.Kind]
		if !ok || a.AssessedAt > seen.AssessedAt || (a.AssessedAt == seen.AssessedAt && a.CreatedAt >= seen.CreatedAt) {
			latest[a.Kind] = a
		}
	}
	b.Inherent = ratedOf(latest, AssessmentInherent)
	b.Residual = ratedOf(latest, AssessmentResidual)
	b.Target = ratedOf(latest, AssessmentTarget)
	return nil
}

func ratedOf(latest map[AssessmentKind]Assessment, kind AssessmentKind) *models.Rated {
	a, ok := latest[kind]
	if !ok {
		return nil
	}
	return models.Rate(a)
}
