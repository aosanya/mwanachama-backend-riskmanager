package riskmanager

import (
	"context"
	"strings"
)

const (
	matchKindRisk    = "risk"
	matchKindControl = "control"
)

func (m *riskManager) Match(ctx context.Context, filter MatchFilter) (Matches, error) {
	term := strings.TrimSpace(filter.Query)
	out := Matches{Query: term, Options: []MatchOption{}}
	if term == "" {
		return out, nil
	}
	limit := capped(filter.Limit)
	risks, err := listOf[Risk](m, containing(m.q(ctx, roleRisk), term, "title").Order("created_at").Limit(limit), roleRisk)
	if err != nil {
		return Matches{}, err
	}
	controls, err := listOf[Control](m, containing(m.q(ctx, roleControl), term, "name").Order("name").Limit(limit), roleControl)
	if err != nil {
		return Matches{}, err
	}
	for _, r := range risks {
		out.Options = append(out.Options, MatchOption{Kind: matchKindRisk, ID: r.ID, Label: r.Title})
	}
	for _, c := range controls {
		out.Options = append(out.Options, MatchOption{Kind: matchKindControl, ID: c.ID, Label: c.Name})
	}
	out.Ambiguous = len(out.Options) > 1
	return out, nil
}
