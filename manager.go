package riskmanager

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/aosanya/mwanachama-backend-shared/identity"
	"github.com/aosanya/mwanachama-backend-shared/spec"
	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-riskmanager/models"
)

const DefaultPage = 200

const stampLayout = "2006-01-02T15:04:05.000000Z07:00"

type RiskManager interface {
	identity.Owner

	CreateRisk(ctx context.Context, r Risk) (Risk, error)
	GetRisk(ctx context.Context, id string) (Risk, error)
	ListRisks(ctx context.Context, filter RiskFilter) ([]Risk, error)
	UpdateRisk(ctx context.Context, r Risk) (Risk, error)
	SetRiskStatus(ctx context.Context, id string, next Status) (Risk, error)

	CreatePathway(ctx context.Context, p Pathway) (Pathway, error)
	GetPathway(ctx context.Context, id string) (Pathway, error)
	UpdatePathway(ctx context.Context, p Pathway) (Pathway, error)
	DeletePathway(ctx context.Context, id string) error

	CreateControl(ctx context.Context, c Control) (Control, error)
	GetControl(ctx context.Context, id string) (Control, error)
	ListControls(ctx context.Context, filter ControlFilter) ([]Control, error)
	UpdateControl(ctx context.Context, c Control) (Control, error)

	PlaceBarrier(ctx context.Context, b Barrier) (Barrier, error)
	ListBarriers(ctx context.Context, filter BarrierFilter) ([]Barrier, error)
	UpdateBarrier(ctx context.Context, change BarrierChange) (Barrier, error)
	RemoveBarrier(ctx context.Context, id string) error

	RecordEvidence(ctx context.Context, e Evidence) (Evidence, error)
	ListEvidence(ctx context.Context, filter EvidenceFilter) ([]Evidence, error)

	RecordAssessment(ctx context.Context, a Assessment) (Assessment, error)
	ListAssessments(ctx context.Context, filter AssessmentFilter) ([]Assessment, error)

	CreateAction(ctx context.Context, a Action) (Action, error)
	ListActions(ctx context.Context, filter ActionFilter) ([]Action, error)
	UpdateAction(ctx context.Context, a Action) (Action, error)
	SetActionStatus(ctx context.Context, id string, next ActionStatus) (Action, error)

	Threats(ctx context.Context, filter PathwayFilter) ([]PathwaySummary, error)
	Consequences(ctx context.Context, filter PathwayFilter) ([]PathwaySummary, error)
	PreventiveMeasures(ctx context.Context, filter MeasureFilter) ([]Measure, error)
	MitigatingMeasures(ctx context.Context, filter MeasureFilter) ([]Measure, error)
	Reliance(ctx context.Context, controlID string) ([]Measure, error)

	Bowtie(ctx context.Context, riskID string) (Bowtie, error)
	Gaps(ctx context.Context, filter GapFilter) ([]Gap, error)
}

type riskManager struct {
	db           *gorm.DB
	st           *store
	clock        func() time.Time
	owners       identity.Owner
	ownerObjects []string
	tasks        identity.Owner
}

func NewRiskManager(db *gorm.DB, s *spec.Spec, opts ...Option) (RiskManager, error) {
	st, err := newStore(db, s, map[string]any{
		roleRisk:       models.Risk{},
		rolePathway:    models.Pathway{},
		roleControl:    models.Control{},
		roleBarrier:    models.Barrier{},
		roleEvidence:   models.Evidence{},
		roleAssessment: models.Assessment{},
		roleAction:     models.Action{},
	})
	if err != nil {
		return nil, fmt.Errorf("NewRiskManager: %w", err)
	}
	m := &riskManager{db: db, st: st, clock: time.Now}
	for _, opt := range opts {
		opt(m)
	}
	return m, nil
}

func (m *riskManager) now() string { return m.clock().UTC().Format(stampLayout) }

func (m *riskManager) q(ctx context.Context, role string) *gorm.DB {
	return m.db.WithContext(ctx).Table(m.st.Table(role))
}

func (m *riskManager) take(q *gorm.DB, role string, out any, notFound error) error {
	var rows []map[string]any
	if err := q.Limit(1).Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return notFound
	}
	return decode(m.st.Object(role), rows[0], out)
}

func listOf[T any](m *riskManager, q *gorm.DB, role string) ([]T, error) {
	return specstore.List[T](m.st, unlimited(q), role)
}

func unlimited(q *gorm.DB) *gorm.DB {
	if c, ok := q.Statement.Clauses["LIMIT"]; ok {
		if l, ok := c.Expression.(clause.Limit); ok && l.Limit != nil {
			return q
		}
	}
	return q.Limit(-1)
}

func (m *riskManager) insert(ctx context.Context, role string, v any) error {
	row, err := encode(m.st.Object(role), v)
	if err != nil {
		return err
	}
	return m.q(ctx, role).Create(row).Error
}

func (m *riskManager) update(ctx context.Context, role string, id string, v any) error {
	row, err := encode(m.st.Object(role), v)
	if err != nil {
		return err
	}
	return m.q(ctx, role).Where("id = ?", id).Updates(row).Error
}

func (m *riskManager) count(ctx context.Context, role, where string, args ...any) (int, error) {
	var n int64
	err := m.q(ctx, role).Where(where, args...).Count(&n).Error
	return int(n), err
}

func (m *riskManager) remove(ctx context.Context, role, id string) error {
	return m.q(ctx, role).Where("id = ?", id).Delete(nil).Error
}

func capped(limit int) int {
	if limit <= 0 {
		return DefaultPage
	}
	return limit
}

func containing(q *gorm.DB, term string, columns ...string) *gorm.DB {
	term = strings.TrimSpace(term)
	if term == "" {
		return q
	}
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToLower(term))
	clauses := make([]string, len(columns))
	args := make([]any, len(columns))
	for i, c := range columns {
		clauses[i] = "lower(" + c + ") LIKE ? ESCAPE '\\'"
		args[i] = "%" + escaped + "%"
	}
	return q.Where("("+strings.Join(clauses, " OR ")+")", args...)
}

func isRFC3339(s string) bool {
	_, err := time.Parse(time.RFC3339, s)
	return err == nil
}
