package routes

import (
	"github.com/aosanya/mwanachama-backend-shared/dispatch"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
)

type Tool = dispatch.Tool

func BuildTools(tm riskmanager.RiskManager, m Mount) ([]Tool, error) {
	s, err := dispatch.Parse(riskmanager.Operations())
	if err != nil {
		return nil, err
	}
	return dispatch.Tools(s, dispatch.Deps{
		Manager: tm, Errors: Sentinels, Authorize: m.Authorize, Caller: m.Caller, Device: m.Device,
	})
}
