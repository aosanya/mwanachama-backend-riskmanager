package models

type Pathway struct {
	ID        string         `json:"id"`
	RiskID    string         `json:"risk_id"`
	Side      Side           `json:"side"`
	Title     string         `json:"title"`
	Scenario  string         `json:"scenario,omitempty"`
	Doc       map[string]any `json:"doc,omitempty"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
}
