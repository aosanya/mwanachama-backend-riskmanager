package models

type Standing string

const (
	StandingProven   Standing = "proven"
	StandingWeak     Standing = "weak"
	StandingUnproven Standing = "unproven"
	StandingFailing  Standing = "failing"
)

func (p Placed) Verdict() Standing {
	switch {
	case p.Latest != nil && p.Latest.Outcome != OutcomePass:
		return StandingFailing
	case p.EvidenceCount == 0:
		return StandingUnproven
	case p.Barrier.Design != JudgementAdequate || p.Barrier.Coverage != JudgementAdequate:
		return StandingWeak
	default:
		return StandingProven
	}
}

type Measure struct {
	RiskID    string  `json:"risk_id"`
	RiskTitle string  `json:"risk_title"`
	Pathway   Pathway `json:"pathway"`
	Placed
	Standing Standing `json:"standing"`
}

type PathwaySummary struct {
	Pathway
	RiskTitle    string `json:"risk_title"`
	BarrierCount int    `json:"barrier_count"`
	Drilldowns   int    `json:"drilldowns"`
}

type PathwayFilter struct {
	RiskID string `json:"risk_id" query:"risk_id"`
	Query  string `json:"query" query:"query"`
	Limit  int    `json:"limit" query:"limit"`
}

type MeasureFilter struct {
	RiskID    string `json:"risk_id" query:"risk_id"`
	PathwayID string `json:"pathway_id" query:"pathway_id"`
	Query     string `json:"query" query:"query"`
	Limit     int    `json:"limit" query:"limit"`
}

type GapFilter struct {
	RiskID   string   `json:"risk_id" query:"risk_id"`
	Kind     GapKind  `json:"kind" query:"kind"`
	Severity Severity `json:"severity" query:"severity"`
	Limit    int      `json:"limit" query:"limit"`
	Offset   int      `json:"offset" query:"offset"`
}
