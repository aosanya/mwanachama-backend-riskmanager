package models

type Rated struct {
	Assessment Assessment `json:"assessment"`
	Score      int        `json:"score"`
	Band       Band       `json:"band"`
}

func Rate(a Assessment) *Rated {
	return &Rated{Assessment: a, Score: a.Score(), Band: a.Band()}
}

type Placed struct {
	Barrier       Barrier   `json:"barrier"`
	Control       Control   `json:"control"`
	Latest        *Evidence `json:"latest_evidence,omitempty"`
	EvidenceCount int       `json:"evidence_count"`
}

type Arm struct {
	Pathway    Pathway  `json:"pathway"`
	Barriers   []Placed `json:"barriers"`
	Drilldowns []Risk   `json:"drilldowns"`
}

type Bowtie struct {
	Risk         Risk     `json:"risk"`
	Inherent     *Rated   `json:"inherent,omitempty"`
	Residual     *Rated   `json:"residual,omitempty"`
	Target       *Rated   `json:"target,omitempty"`
	Threats      []Arm    `json:"threats"`
	Consequences []Arm    `json:"consequences"`
	Actions      []Action `json:"actions"`
}
