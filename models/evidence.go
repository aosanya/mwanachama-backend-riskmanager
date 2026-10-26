package models

type Evidence struct {
	ID         string       `json:"id"`
	BarrierID  string       `json:"barrier_id"`
	Kind       EvidenceKind `json:"kind"`
	Outcome    Outcome      `json:"outcome"`
	Summary    string       `json:"summary"`
	Reference  string       `json:"reference,omitempty"`
	RecordedBy string       `json:"recorded_by,omitempty"`
	RecordedAt string       `json:"recorded_at"`
	CreatedAt  string       `json:"created_at"`
}

type EvidenceFilter struct {
	BarrierID string `json:"barrier_id" query:"barrier_id"`
	From      string `json:"from" query:"from"`
	To        string `json:"to" query:"to"`
	Limit     int    `json:"limit" query:"limit"`
}
