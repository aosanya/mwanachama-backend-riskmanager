package routes

import (
	"github.com/aosanya/mwanachama-backend-shared/dispatch"
	"github.com/aosanya/mwanachama-backend-shared/httpwire"

	riskmanager "github.com/aosanya/mwanachama-backend-riskmanager"
)

type Route = httpwire.Route

type Mount = dispatch.Mount

var Sentinels = map[string]error{
	"ErrInvalid":          riskmanager.ErrInvalid,
	"ErrInvalidReference": riskmanager.ErrInvalidReference,
	"ErrConflict":         riskmanager.ErrConflict,
	"ErrRiskNotFound":     riskmanager.ErrRiskNotFound,
	"ErrPathwayNotFound":  riskmanager.ErrPathwayNotFound,
	"ErrControlNotFound":  riskmanager.ErrControlNotFound,
	"ErrBarrierNotFound":  riskmanager.ErrBarrierNotFound,
	"ErrActionNotFound":   riskmanager.ErrActionNotFound,
}

var AnonymousActions = []string{}

var Table = dispatch.NewTable(riskmanager.Operations(), Sentinels, AnonymousActions...)

func Build(rm riskmanager.RiskManager) ([]Route, error) { return Table.Build(rm, Mount{}) }

func BuildFor(rm riskmanager.RiskManager, m Mount) ([]Route, error) { return Table.Build(rm, m) }

func Routes(rm riskmanager.RiskManager) []Route { return Table.Routes(rm, Mount{}) }

func RoutesFor(rm riskmanager.RiskManager, m Mount) []Route { return Table.Routes(rm, m) }

func Split(rm riskmanager.RiskManager, m Mount) dispatch.Split { return Table.Split(rm, m) }

func PublicRoutes(rm riskmanager.RiskManager) []Route {
	return Table.Split(rm, Mount{}).Anonymous
}

func OperatorRoutes(rm riskmanager.RiskManager, m Mount) []Route {
	return Table.Split(rm, m).Gated
}
