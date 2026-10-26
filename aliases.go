package riskmanager

import "github.com/aosanya/mwanachama-backend-riskmanager/models"

type (
	Risk             = models.Risk
	RiskFilter       = models.RiskFilter
	Pathway          = models.Pathway
	Control          = models.Control
	ControlFilter    = models.ControlFilter
	Barrier          = models.Barrier
	BarrierFilter    = models.BarrierFilter
	Evidence         = models.Evidence
	EvidenceFilter   = models.EvidenceFilter
	Assessment       = models.Assessment
	AssessmentFilter = models.AssessmentFilter
	Action           = models.Action
	ActionFilter     = models.ActionFilter
	Bowtie           = models.Bowtie
	Gap              = models.Gap
	GapKind          = models.GapKind
	Severity         = models.Severity
	Status           = models.Status
	Side             = models.Side
	Judgement        = models.Judgement
	Maturity         = models.Maturity
	EvidenceKind     = models.EvidenceKind
	Outcome          = models.Outcome
	AssessmentKind   = models.AssessmentKind
	ActionStatus     = models.ActionStatus
	Band             = models.Band
	Standing         = models.Standing
	Measure          = models.Measure
	PathwaySummary   = models.PathwaySummary
	PathwayFilter    = models.PathwayFilter
	MeasureFilter    = models.MeasureFilter
	GapFilter        = models.GapFilter
	BarrierChange    = models.BarrierChange
)

const (
	StatusDraft    = models.StatusDraft
	StatusOpen     = models.StatusOpen
	StatusAccepted = models.StatusAccepted
	StatusRetired  = models.StatusRetired

	SideThreat      = models.SideThreat
	SideConsequence = models.SideConsequence

	JudgementUnassessed = models.JudgementUnassessed
	JudgementAdequate   = models.JudgementAdequate
	JudgementPartial    = models.JudgementPartial
	JudgementInadequate = models.JudgementInadequate

	MaturityProposed  = models.MaturityProposed
	MaturityDesigned  = models.MaturityDesigned
	MaturityDeployed  = models.MaturityDeployed
	MaturityTested    = models.MaturityTested
	MaturityMonitored = models.MaturityMonitored

	EvidenceTest          = models.EvidenceTest
	EvidenceLog           = models.EvidenceLog
	EvidenceAudit         = models.EvidenceAudit
	EvidenceConfiguration = models.EvidenceConfiguration
	EvidenceIncident      = models.EvidenceIncident

	OutcomePass    = models.OutcomePass
	OutcomePartial = models.OutcomePartial
	OutcomeFail    = models.OutcomeFail

	AssessmentInherent = models.AssessmentInherent
	AssessmentResidual = models.AssessmentResidual
	AssessmentTarget   = models.AssessmentTarget

	ActionOpen       = models.ActionOpen
	ActionInProgress = models.ActionInProgress
	ActionDone       = models.ActionDone
	ActionDropped    = models.ActionDropped

	StandingProven   = models.StandingProven
	StandingWeak     = models.StandingWeak
	StandingUnproven = models.StandingUnproven
	StandingFailing  = models.StandingFailing
)
