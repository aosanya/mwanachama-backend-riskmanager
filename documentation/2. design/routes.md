# The route table is declared

Every address this module answers is an entry in
[`riskmanager.operations.json`](../../riskmanager.operations.json). There are
no hand-written handlers: `routes/routes.go` is an adapter that hands the
operations file, the sentinel map and the anonymous allowlist to
`mwanachama-backend-shared/dispatch`, which builds the `http.HandlerFunc` for
each entry by reflecting on `RiskManager`.

Adding an address is an edit to the JSON file plus a method on the manager.
If the two disagree — a wrong arity, a missing method, a sentinel the spec
maps but nobody supplies — `dispatch.Dispatch` refuses to build the table at
all, so the failure arrives at startup rather than on the first request.

## The thirty-two operations

| Operation | Method | Path | Action | Manager call | Status |
|---|---|---|---|---|---|
| `create_risk` | POST | `/risks` | `riskmanager.risk.create` | `CreateRisk` | 201 |
| `get_risk` | GET | `/risks/{riskID}` | `riskmanager.risk.read` | `GetRisk` | 200 |
| `list_risks` | GET | `/risks` | `riskmanager.risk.list` | `ListRisks` | 200 |
| `update_risk` | PUT | `/risks/{riskID}` | `riskmanager.risk.update` | `UpdateRisk` | 200 |
| `set_risk_status` | POST | `/risks/{riskID}/status` | `riskmanager.risk.set_status` | `SetRiskStatus` | 200 |
| `create_pathway` | POST | `/pathways` | `riskmanager.pathway.create` | `CreatePathway` | 201 |
| `get_pathway` | GET | `/pathways/{pathwayID}` | `riskmanager.pathway.read` | `GetPathway` | 200 |
| `update_pathway` | PUT | `/pathways/{pathwayID}` | `riskmanager.pathway.update` | `UpdatePathway` | 200 |
| `delete_pathway` | DELETE | `/pathways/{pathwayID}` | `riskmanager.pathway.delete` | `DeletePathway` | 204 |
| `create_control` | POST | `/controls` | `riskmanager.control.create` | `CreateControl` | 201 |
| `get_control` | GET | `/controls/{controlID}` | `riskmanager.control.read` | `GetControl` | 200 |
| `list_controls` | GET | `/controls` | `riskmanager.control.list` | `ListControls` | 200 |
| `update_control` | PUT | `/controls/{controlID}` | `riskmanager.control.update` | `UpdateControl` | 200 |
| `place_barrier` | POST | `/barriers` | `riskmanager.barrier.place` | `PlaceBarrier` | 201 |
| `list_barriers` | GET | `/barriers` | `riskmanager.barrier.list` | `ListBarriers` | 200 |
| `update_barrier` | PUT | `/barriers/{barrierID}` | `riskmanager.barrier.update` | `UpdateBarrier` | 200 |
| `remove_barrier` | DELETE | `/barriers/{barrierID}` | `riskmanager.barrier.remove` | `RemoveBarrier` | 204 |
| `record_evidence` | POST | `/evidence` | `riskmanager.evidence.record` | `RecordEvidence` | 201 |
| `list_evidence` | GET | `/evidence` | `riskmanager.evidence.list` | `ListEvidence` | 200 |
| `record_assessment` | POST | `/assessments` | `riskmanager.assessment.record` | `RecordAssessment` | 201 |
| `list_assessments` | GET | `/assessments` | `riskmanager.assessment.list` | `ListAssessments` | 200 |
| `create_action` | POST | `/actions` | `riskmanager.action.create` | `CreateAction` | 201 |
| `list_actions` | GET | `/actions` | `riskmanager.action.list` | `ListActions` | 200 |
| `update_action` | PUT | `/actions/{actionID}` | `riskmanager.action.update` | `UpdateAction` | 200 |
| `set_action_status` | POST | `/actions/{actionID}/status` | `riskmanager.action.set_status` | `SetActionStatus` | 200 |
| `list_threats` | GET | `/threats` | `riskmanager.threat.list` | `Threats` | 200 |
| `list_consequences` | GET | `/consequences` | `riskmanager.consequence.list` | `Consequences` | 200 |
| `list_preventive_measures` | GET | `/preventive-measures` | `riskmanager.preventive_measure.list` | `PreventiveMeasures` | 200 |
| `list_mitigating_measures` | GET | `/mitigating-measures` | `riskmanager.mitigating_measure.list` | `MitigatingMeasures` | 200 |
| `list_reliance` | GET | `/controls/{controlID}/reliance` | `riskmanager.control.reliance` | `Reliance` | 200 |
| `get_bowtie` | GET | `/risks/{riskID}/bowtie` | `riskmanager.risk.bowtie` | `Bowtie` | 200 |
| `list_gaps` | GET | `/gaps` | `riskmanager.gap.list` | `Gaps` | 200 |

Action ids follow `riskmanager.<object>.<verb>`. The MCP tool for an
operation is the action id with dots as underscores —
`riskmanager.threat.list` is `riskmanager_threat_list`. Operations are
grouped below by what they are for.

| Group | Operations |
|---|---|
| The register | `create_risk`, `get_risk`, `list_risks`, `update_risk`, `set_risk_status` |
| The two sides | `create_pathway`, `get_pathway`, `update_pathway`, `delete_pathway` |
| The measures | `create_control`, `get_control`, `list_controls`, `update_control`, `place_barrier`, `list_barriers`, `update_barrier`, `remove_barrier` |
| The record | `record_evidence`, `list_evidence`, `record_assessment`, `list_assessments` |
| The work | `create_action`, `list_actions`, `update_action`, `set_action_status` |
| The questions | `list_threats`, `list_consequences`, `list_preventive_measures`, `list_mitigating_measures`, `list_reliance` |
| The views | `get_bowtie`, `list_gaps` |

The five question operations and the two view operations are the reason the
module exists; see [questions.md](questions.md) and [gaps.md](gaps.md).

## The gate is data

`AnonymousActions` is an allowlist of action ids and it is **empty**. This
module publishes nothing anonymously: every operation arrives gated.

The allowlist names what is public, never what is protected. An operation
added to the file later and not named there arrives gated, so the failure
direction is a 401 rather than an unpublished surface on the open internet.

This module has **no auth model of its own**. A mount supplies a
`dispatch.Authorizer` through `routes.Mount`, and a refused action answers
403 without reaching the manager. Because the action id is per operation, a
mount can grant reading the five questions to an assessor without granting
`riskmanager.evidence.record`.

## The address outranks the body, and the caller outranks both

- `update_risk`, `update_pathway`, `update_control`, `update_barrier` and
  `update_action` bind the id from the path with `into: ID`, applied after
  the body is decoded. Without it a caller could `PUT /risks/a` with a body
  claiming to be `b` and rewrite a different risk.
- `record_evidence` binds `recorded_by` and `record_assessment` binds
  `assessed_by` with `{"from": "caller", "into": ...}`. The mount supplies
  who is asking; no request can name a different author, and the field is
  kept out of every MCP tool schema.

## Errors

The `errors` block maps every exported sentinel to a status. An unmapped
sentinel is redacted to a 500 with `"internal error"`, so
`TestEverySentinelIsMappedToAStatus` fails if a sentinel is exported without
a status.

| Sentinel | Status |
|---|---|
| `ErrInvalid` | 400 |
| `ErrInvalidReference` | 400 |
| `ErrConflict` | 409 |
| `ErrRiskNotFound` | 404 |
| `ErrPathwayNotFound` | 404 |
| `ErrControlNotFound` | 404 |
| `ErrBarrierNotFound` | 404 |
| `ErrActionNotFound` | 404 |

`ErrConflict` carries the lifecycle refusals that are not a bad request: a
write to a retired risk, a duplicate control name, a control already placed
on a pathway, a delete or removal blocked by dependants. `ErrInvalidReference`
is a body that names a risk, pathway, control or barrier that does not exist.

## Two reads that are not lists

`get_bowtie` answers one object, the whole diagram, and `list_gaps` answers
a list of derived findings, ranked high to low and filterable by `risk_id`,
`kind` and `severity` and pageable by `limit` and `offset`. Neither is a
stored table; both are assembled per call.

## MCP tools

`mcp.RegisterTools(server, resolve, mount)` registers one MCP tool per
declared operation, generated by `dispatch.Tools` over
`riskmanager.operations.json` through `routes.BuildTools`, following
`mwanachama-backend-digitaltwin`'s `mcp/` package: every tool takes an
`instance_id` beside its declared arguments, and `resolve` turns it into the
`RiskManager` for that call. An unknown instance, or none named, is a tool
error and never reaches a manager. Every operation takes a plain object
body, so none is declared `"tool": false` and there is no hand-written tool.
