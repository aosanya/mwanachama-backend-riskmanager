package models

const (
	ScaleMin = 1
	ScaleMax = 5
)

type Band string

const (
	BandLow      Band = "low"
	BandMedium   Band = "medium"
	BandHigh     Band = "high"
	BandCritical Band = "critical"
)

type Assessment struct {
	ID         string         `json:"id"`
	RiskID     string         `json:"risk_id"`
	Kind       AssessmentKind `json:"kind"`
	Likelihood int            `json:"likelihood"`
	Impact     int            `json:"impact"`
	Rationale  string         `json:"rationale,omitempty"`
	AssessedBy string         `json:"assessed_by,omitempty"`
	AssessedAt string         `json:"assessed_at"`
	CreatedAt  string         `json:"created_at"`
}

func (a Assessment) OnTheScale() bool {
	return within(a.Likelihood) && within(a.Impact)
}

func within(n int) bool { return n >= ScaleMin && n <= ScaleMax }

func (a Assessment) Score() int { return a.Likelihood * a.Impact }

func (a Assessment) Band() Band { return BandOf(a.Score()) }

func BandOf(score int) Band {
	switch {
	case score >= 16:
		return BandCritical
	case score >= 10:
		return BandHigh
	case score >= 5:
		return BandMedium
	default:
		return BandLow
	}
}

type AssessmentFilter struct {
	RiskID string         `json:"risk_id" query:"risk_id"`
	Kind   AssessmentKind `json:"kind" query:"kind"`
	Limit  int            `json:"limit" query:"limit"`
}
