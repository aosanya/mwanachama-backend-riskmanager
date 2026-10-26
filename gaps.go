package riskmanager

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-riskmanager/models"
)

func (m *riskManager) Gaps(ctx context.Context, filter GapFilter) ([]Gap, error) {
	if err := gapFilterKnown(filter); err != nil {
		return nil, err
	}
	risks, err := m.risksToReview(ctx, filter.RiskID)
	if err != nil {
		return nil, err
	}
	now := m.clock()
	ranked := []Gap{}
	for _, r := range risks {
		b, err := m.Bowtie(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		ranked = append(ranked, b.Gaps(now)...)
	}
	models.RankGaps(ranked)
	return pageOfGaps(narrowedGaps(ranked, filter), filter), nil
}

func (m *riskManager) risksToReview(ctx context.Context, riskID string) ([]Risk, error) {
	if riskID != "" {
		r, err := m.GetRisk(ctx, riskID)
		if err != nil {
			return nil, err
		}
		return []Risk{r}, nil
	}
	return listOf[Risk](m, m.q(ctx, roleRisk).Where("status <> ?", string(StatusRetired)).Order("created_at"), roleRisk)
}

func narrowedGaps(gaps []Gap, filter GapFilter) []Gap {
	out := make([]Gap, 0, len(gaps))
	for _, g := range gaps {
		if filter.Kind != "" && g.Kind != filter.Kind {
			continue
		}
		if filter.Severity != "" && g.Severity != filter.Severity {
			continue
		}
		out = append(out, g)
	}
	return out
}

func pageOfGaps(gaps []Gap, filter GapFilter) []Gap {
	offset := max(filter.Offset, 0)
	if offset >= len(gaps) {
		return []Gap{}
	}
	return gaps[offset:min(offset+capped(filter.Limit), len(gaps))]
}

func gapFilterKnown(filter GapFilter) error {
	if filter.Kind != "" && filter.Kind.Severity() == "" {
		return fmt.Errorf("%w: gap kind %q is not one this module reports", ErrInvalid, filter.Kind)
	}
	if filter.Severity != "" && !filter.Severity.Known() {
		return fmt.Errorf("%w: severity %q is not one of high, medium, low", ErrInvalid, filter.Severity)
	}
	return nil
}
