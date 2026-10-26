package models

type Risk struct {
	ID              string         `json:"id"`
	Title           string         `json:"title"`
	TopEvent        string         `json:"top_event"`
	Hazard          string         `json:"hazard,omitempty"`
	Owner           string         `json:"owner,omitempty"`
	Status          Status         `json:"status"`
	ParentPathwayID string         `json:"parent_pathway_id,omitempty"`
	Doc             map[string]any `json:"doc,omitempty"`
	CreatedAt       string         `json:"created_at"`
	UpdatedAt       string         `json:"updated_at"`
}

type RiskFilter struct {
	Status          Status `json:"status" query:"status"`
	Owner           string `json:"owner" query:"owner"`
	ParentPathwayID string `json:"parent_pathway_id" query:"parent_pathway_id"`
	Query           string `json:"query" query:"query"`
	Limit           int    `json:"limit" query:"limit"`
}
