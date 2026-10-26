package models

type Barrier struct {
	ID        string    `json:"id"`
	RiskID    string    `json:"risk_id"`
	PathwayID string    `json:"pathway_id"`
	ControlID string    `json:"control_id"`
	Position  int       `json:"position"`
	Design    Judgement `json:"design"`
	Coverage  Judgement `json:"coverage"`
	Maturity  Maturity  `json:"maturity"`
	Notes     string    `json:"notes,omitempty"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

type BarrierFilter struct {
	RiskID    string `json:"risk_id" query:"risk_id"`
	PathwayID string `json:"pathway_id" query:"pathway_id"`
	ControlID string `json:"control_id" query:"control_id"`
	Limit     int    `json:"limit" query:"limit"`
}

type BarrierChange struct {
	ID       string     `json:"id"`
	Position *int       `json:"position,omitempty"`
	Design   *Judgement `json:"design,omitempty"`
	Coverage *Judgement `json:"coverage,omitempty"`
	Maturity *Maturity  `json:"maturity,omitempty"`
	Notes    *string    `json:"notes,omitempty"`
}
