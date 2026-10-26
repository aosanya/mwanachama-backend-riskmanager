package models

type Control struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Owner       string         `json:"owner,omitempty"`
	Doc         map[string]any `json:"doc,omitempty"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}

type ControlFilter struct {
	Owner string `json:"owner" query:"owner"`
	Limit int    `json:"limit" query:"limit"`
}
