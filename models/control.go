package models

type Control struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Description     string         `json:"description,omitempty"`
	Owner           string         `json:"owner,omitempty"`
	Doc             map[string]any `json:"doc,omitempty"`
	FailingBarriers int            `json:"failing_barriers,omitempty" spec:"-"`
	CreatedAt       string         `json:"created_at"`
	UpdatedAt       string         `json:"updated_at"`
}

type ControlFilter struct {
	Owner   string `json:"owner" query:"owner"`
	Failing bool   `json:"failing" query:"failing"`
	Limit   int    `json:"limit" query:"limit"`
}
