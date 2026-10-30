package models

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type GapKind string

const (
	GapRiskUnowned       GapKind = "risk_unowned"
	GapRiskOneSided      GapKind = "risk_one_sided"
	GapRiskUnrated       GapKind = "risk_unrated"
	GapPathwayUnguarded  GapKind = "pathway_unguarded"
	GapBarrierDesign     GapKind = "barrier_design"
	GapBarrierCoverage   GapKind = "barrier_coverage"
	GapBarrierUnassessed GapKind = "barrier_unassessed"
	GapBarrierUntested   GapKind = "barrier_untested"
	GapBarrierFailing    GapKind = "barrier_failing"
	GapBarrierStale      GapKind = "barrier_stale"
	GapControlUnowned    GapKind = "control_unowned"
	GapActionUndated     GapKind = "action_undated"
	GapActionOverdue     GapKind = "action_overdue"
)

type Severity string

const (
	SeverityHigh   Severity = "high"
	SeverityMedium Severity = "medium"
	SeverityLow    Severity = "low"
)

func (s Severity) Known() bool { return s.rank() < len(severityOrder) }

func (s Severity) rank() int {
	for i, held := range severityOrder {
		if s == held {
			return i
		}
	}
	return len(severityOrder)
}

var severityOrder = []Severity{SeverityHigh, SeverityMedium, SeverityLow}

func (k GapKind) Severity() Severity {
	switch k {
	case GapPathwayUnguarded, GapBarrierFailing, GapActionOverdue:
		return SeverityHigh
	case GapRiskUnowned, GapRiskOneSided, GapRiskUnrated, GapControlUnowned, GapActionUndated, GapBarrierDesign, GapBarrierCoverage, GapBarrierStale:
		return SeverityMedium
	case GapBarrierUnassessed, GapBarrierUntested:
		return SeverityLow
	default:
		return ""
	}
}

type Gap struct {
	Kind      GapKind  `json:"kind"`
	Severity  Severity `json:"severity"`
	RiskID    string   `json:"risk_id"`
	Subject   string   `json:"subject"`
	SubjectID string   `json:"subject_id"`
	Message   string   `json:"message"`
}

func RankGaps(gaps []Gap) {
	slices.SortStableFunc(gaps, func(a, b Gap) int { return a.Severity.rank() - b.Severity.rank() })
}

func (b Bowtie) Gaps(now time.Time) []Gap {
	if b.Risk.Status.Frozen() {
		return nil
	}
	out := b.riskGaps()
	owned := map[string]bool{}
	for _, arm := range append(append([]Arm{}, b.Threats...), b.Consequences...) {
		out = append(out, b.armGaps(arm, owned, now)...)
	}
	out = append(out, b.actionGaps(now)...)
	RankGaps(out)
	return out
}

func (b Bowtie) gap(kind GapKind, subject, subjectID, format string, args ...any) Gap {
	return Gap{Kind: kind, Severity: kind.Severity(), RiskID: b.Risk.ID, Subject: subject, SubjectID: subjectID, Message: fmt.Sprintf(format, args...)}
}

func (b Bowtie) riskGaps() []Gap {
	var out []Gap
	if b.Risk.Owner == "" {
		out = append(out, b.gap(GapRiskUnowned, "risk", b.Risk.ID, "nobody is accountable for %q", b.Risk.Title))
	}
	if len(b.Threats) == 0 {
		out = append(out, b.gap(GapRiskOneSided, "risk", b.Risk.ID, "%q has no threats, so nothing says how it comes about", b.Risk.Title))
	}
	if len(b.Consequences) == 0 {
		out = append(out, b.gap(GapRiskOneSided, "risk", b.Risk.ID, "%q has no consequences, so nothing says why it matters", b.Risk.Title))
	}
	if b.Inherent == nil {
		out = append(out, b.gap(GapRiskUnrated, "risk", b.Risk.ID, "%q has no inherent rating", b.Risk.Title))
	}
	if b.Residual == nil {
		out = append(out, b.gap(GapRiskUnrated, "risk", b.Risk.ID, "%q has no residual rating", b.Risk.Title))
	}
	return out
}

func (b Bowtie) armGaps(arm Arm, ownedControls map[string]bool, now time.Time) []Gap {
	var out []Gap
	if len(arm.Barriers) == 0 {
		out = append(out, b.gap(GapPathwayUnguarded, "pathway", arm.Pathway.ID, "nothing stands on the %s %q", arm.Pathway.Side, arm.Pathway.Title))
	}
	for _, p := range arm.Barriers {
		out = append(out, b.barrierGaps(arm.Pathway, p, now)...)
		if p.Control.Owner == "" && !ownedControls[p.Control.ID] {
			ownedControls[p.Control.ID] = true
			out = append(out, b.gap(GapControlUnowned, "control", p.Control.ID, "nobody operates the control %q", p.Control.Name))
		}
	}
	return out
}

func (b Bowtie) barrierGaps(pathway Pathway, p Placed, now time.Time) []Gap {
	var out []Gap
	where := fmt.Sprintf("%q on the %s %q", p.Control.Name, pathway.Side, pathway.Title)
	if found(p.Barrier.Design) {
		out = append(out, b.gap(GapBarrierDesign, "barrier", p.Barrier.ID, "the design of %s is %s", where, p.Barrier.Design))
	}
	if found(p.Barrier.Coverage) {
		out = append(out, b.gap(GapBarrierCoverage, "barrier", p.Barrier.ID, "the coverage of %s is %s", where, p.Barrier.Coverage))
	}
	if unassessed := unassessedJudgements(p.Barrier); len(unassessed) > 0 {
		out = append(out, b.gap(GapBarrierUnassessed, "barrier", p.Barrier.ID, "%s has not had its %s assessed", where, strings.Join(unassessed, " and ")))
	}
	if !p.Barrier.Maturity.NeedsEvidence() {
		out = append(out, b.gap(GapBarrierUntested, "barrier", p.Barrier.ID, "%s has got no further than %s, with %d piece(s) of evidence", where, p.Barrier.Maturity, p.EvidenceCount))
	}
	if p.Stale(now) {
		out = append(out, b.gap(GapBarrierStale, "barrier", p.Barrier.ID, "the latest evidence for %s was recorded %s, more than the %d day(s) it stays current", where, p.Latest.RecordedAt, p.Barrier.RetestDays))
	}
	if p.Latest != nil && p.Latest.Outcome != OutcomePass {
		out = append(out, b.gap(GapBarrierFailing, "barrier", p.Barrier.ID, "the latest evidence for %s was a %s: %s", where, p.Latest.Outcome, p.Latest.Summary))
	}
	return out
}

func found(j Judgement) bool { return j != JudgementAdequate && j != JudgementUnassessed }

func unassessedJudgements(b Barrier) []string {
	var out []string
	if b.Design == JudgementUnassessed {
		out = append(out, "design")
	}
	if b.Coverage == JudgementUnassessed {
		out = append(out, "coverage")
	}
	return out
}

func (b Bowtie) actionGaps(now time.Time) []Gap {
	var out []Gap
	for _, a := range b.Actions {
		if a.Status.Closed() {
			continue
		}
		due, err := time.Parse(time.RFC3339, a.DueAt)
		switch {
		case a.DueAt == "":
			out = append(out, b.gap(GapActionUndated, "action", a.ID, "the work %q has no due date", a.Title))
		case err == nil && due.Before(now):
			out = append(out, b.gap(GapActionOverdue, "action", a.ID, "the work %q was due %s", a.Title, a.DueAt))
		}
	}
	return out
}
