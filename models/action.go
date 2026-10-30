package models

type Action struct {
	ID          string       `json:"id"`
	RiskID      string       `json:"risk_id"`
	BarrierID   string       `json:"barrier_id,omitempty"`
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Owner       string       `json:"owner,omitempty"`
	DueAt       string       `json:"due_at,omitempty"`
	Status      ActionStatus `json:"status"`
	ClosedAt    string       `json:"closed_at,omitempty"`
	CreatedAt   string       `json:"created_at"`
	UpdatedAt   string       `json:"updated_at"`
}

type ActionFilter struct {
	RiskID    string       `json:"risk_id" query:"risk_id"`
	BarrierID string       `json:"barrier_id" query:"barrier_id"`
	Owner     string       `json:"owner" query:"owner"`
	Status    ActionStatus `json:"status" query:"status"`
	Limit     int          `json:"limit" query:"limit"`
}
