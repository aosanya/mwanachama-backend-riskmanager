package riskmanager

import (
	"context"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-riskmanager/models"
)

func (m *riskManager) bowtiesOf(ctx context.Context, risks []Risk, riskIDs func() *gorm.DB) ([]models.Bowtie, error) {
	if len(risks) == 0 {
		return []models.Bowtie{}, nil
	}
	barrierIDs := func() *gorm.DB { return m.q(ctx, roleBarrier).Select("id").Where("risk_id IN (?)", riskIDs()) }
	controlIDs := func() *gorm.DB { return m.q(ctx, roleBarrier).Select("control_id").Where("risk_id IN (?)", riskIDs()) }

	pathways, err := listOf[Pathway](m, m.q(ctx, rolePathway).Where("risk_id IN (?)", riskIDs()).Order("created_at"), rolePathway)
	if err != nil {
		return nil, err
	}
	barriers, err := listOf[Barrier](m, m.q(ctx, roleBarrier).Where("risk_id IN (?)", riskIDs()).Order(`"position"`).Order("created_at"), roleBarrier)
	if err != nil {
		return nil, err
	}
	controls, err := listOf[Control](m, m.q(ctx, roleControl).Where("id IN (?)", controlIDs()), roleControl)
	if err != nil {
		return nil, err
	}
	evidence, err := listOf[Evidence](m, m.q(ctx, roleEvidence).Where("barrier_id IN (?)", barrierIDs()).Order("recorded_at").Order("created_at"), roleEvidence)
	if err != nil {
		return nil, err
	}
	assessments, err := listOf[Assessment](m, m.q(ctx, roleAssessment).Where("risk_id IN (?)", riskIDs()), roleAssessment)
	if err != nil {
		return nil, err
	}
	actions, err := listOf[Action](m, m.q(ctx, roleAction).Where("risk_id IN (?)", riskIDs()).Order("created_at"), roleAction)
	if err != nil {
		return nil, err
	}

	controlByID := make(map[string]Control, len(controls))
	for _, c := range controls {
		controlByID[c.ID] = c
	}
	evidenceOn := map[string][]Evidence{}
	for _, e := range evidence {
		evidenceOn[e.BarrierID] = append(evidenceOn[e.BarrierID], e)
	}
	placedOn := map[string][]models.Placed{}
	for _, b := range barriers {
		placedOn[b.PathwayID] = append(placedOn[b.PathwayID], placedOf(b, controlByID[b.ControlID], evidenceOn[b.ID]))
	}
	armsOf := map[string][]models.Arm{}
	for _, p := range pathways {
		arm := models.Arm{Pathway: p, Barriers: placedOn[p.ID], Drilldowns: []Risk{}}
		if arm.Barriers == nil {
			arm.Barriers = []models.Placed{}
		}
		armsOf[p.RiskID] = append(armsOf[p.RiskID], arm)
	}
	assessedOn := map[string][]Assessment{}
	for _, a := range assessments {
		assessedOn[a.RiskID] = append(assessedOn[a.RiskID], a)
	}
	actionsOn := map[string][]Action{}
	for _, a := range actions {
		actionsOn[a.RiskID] = append(actionsOn[a.RiskID], a)
	}

	out := make([]models.Bowtie, 0, len(risks))
	for _, r := range risks {
		b := models.Bowtie{Risk: r, Threats: []models.Arm{}, Consequences: []models.Arm{}, Actions: []Action{}}
		for _, arm := range armsOf[r.ID] {
			if arm.Pathway.Side == SideThreat {
				b.Threats = append(b.Threats, arm)
			} else {
				b.Consequences = append(b.Consequences, arm)
			}
		}
		b.Inherent, b.Residual, b.Target = latestRatings(assessedOn[r.ID])
		b.Actions = append(b.Actions, actionsOn[r.ID]...)
		out = append(out, b)
	}
	return out, nil
}
