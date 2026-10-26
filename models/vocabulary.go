package models

type Status string

const (
	StatusDraft    Status = "draft"
	StatusOpen     Status = "open"
	StatusAccepted Status = "accepted"
	StatusRetired  Status = "retired"
)

func (s Status) CanTransitionTo(next Status) bool {
	switch s {
	case StatusDraft:
		return next == StatusOpen || next == StatusRetired
	case StatusOpen:
		return next == StatusAccepted || next == StatusRetired
	case StatusAccepted:
		return next == StatusOpen || next == StatusRetired
	default:
		return false
	}
}

func (s Status) Frozen() bool { return s == StatusRetired }

type Side string

const (
	SideThreat      Side = "threat"
	SideConsequence Side = "consequence"
)

type Judgement string

const (
	JudgementUnassessed Judgement = "unassessed"
	JudgementAdequate   Judgement = "adequate"
	JudgementPartial    Judgement = "partial"
	JudgementInadequate Judgement = "inadequate"
)

type Maturity string

const (
	MaturityProposed  Maturity = "proposed"
	MaturityDesigned  Maturity = "designed"
	MaturityDeployed  Maturity = "deployed"
	MaturityTested    Maturity = "tested"
	MaturityMonitored Maturity = "monitored"
)

func (m Maturity) NeedsEvidence() bool { return m == MaturityTested || m == MaturityMonitored }

type EvidenceKind string

const (
	EvidenceTest          EvidenceKind = "test"
	EvidenceLog           EvidenceKind = "log"
	EvidenceAudit         EvidenceKind = "audit"
	EvidenceConfiguration EvidenceKind = "configuration"
	EvidenceIncident      EvidenceKind = "incident"
)

type Outcome string

const (
	OutcomePass    Outcome = "pass"
	OutcomePartial Outcome = "partial"
	OutcomeFail    Outcome = "fail"
)

type AssessmentKind string

const (
	AssessmentInherent AssessmentKind = "inherent"
	AssessmentResidual AssessmentKind = "residual"
	AssessmentTarget   AssessmentKind = "target"
)

type ActionStatus string

const (
	ActionOpen       ActionStatus = "open"
	ActionInProgress ActionStatus = "in_progress"
	ActionDone       ActionStatus = "done"
	ActionDropped    ActionStatus = "dropped"
)

func (s ActionStatus) CanTransitionTo(next ActionStatus) bool {
	switch s {
	case ActionOpen:
		return next == ActionInProgress || next == ActionDone || next == ActionDropped
	case ActionInProgress:
		return next == ActionOpen || next == ActionDone || next == ActionDropped
	default:
		return false
	}
}

func (s ActionStatus) Closed() bool { return s == ActionDone || s == ActionDropped }
