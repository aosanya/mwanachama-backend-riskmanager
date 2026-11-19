# Hub import drift

Before the hub stops carrying its own copy of the risk manager module and imports this repository instead, this page lists every difference between the two. The hub copy is `projects/hub-steward/code/wakala-api/modules/riskmanager` (last changed in the hub at 09e613ca, 2026-11-17); this repository is the custodian. The method of the comparison is on [hub-copy-diff.md](hub-copy-diff.md).

Measured on 2026-11-19 with `scripts/hub-copy-diff.sh`, after rewriting the hub's import paths (`wakalaserver/modules/shared/` to `github.com/aosanya/mwanachama-backend-shared/`, `wakalaserver/modules/riskmanager` to `github.com/aosanya/mwanachama-backend-riskmanager`) and dropping comments and layout. The hub folder had no uncommitted changes.

## Result

This repository is already the union of both sides. No file and no behaviour exists only in the hub, so nothing was moved from the hub into this repository. The hub copy is the older side: it lacks the control-name work this repository made on 2026-11-16, and its one commit after the last sync (09e613ca, the unique-violation mapping in `CreateControl`) is covered by `nameTaken` here.

## Only in the hub

None. The script prints no `Only in hub` line, and every file of the hub copy exists here with the same content apart from the import path: the root package, `models/`, `routes/` (including `author_test.go` and `routes_cover_test.go`), `mcp/`, the three JSON files, and the hub's own tests (`ask_test.go`, `control_fold_test.go`, `council_pack_test.go`, `evidence_standing_test.go`, `gap_task_test.go`, `list_limit_test.go`, `riskmanager_test.go`).

## Differing beyond the import path

Every difference is a change this repository made that the hub copy does not have.

| File | Hub copy | This repository | Disposition |
|---|---|---|---|
| `control.go` | `CreateControl` maps a unique violation to the conflict answer (09e613ca); `UpdateControl` compares names with `plaintext.SameName`, checks and updates the control, and returns the raw database error on a unique violation | One mutex around create and rename, the folded `NameKey` written with the name, `nameTaken` mapping the unique violation to the conflict answer on create and on update, the owner looked up before the name lock is taken, `controlChangedTo` building the next row | Keep this repository's. It is a superset: the create mapping of 09e613ca is `nameTaken`. The hub takes it by importing the module |
| `manager.go` | no `controlNames` field | `controlNames sync.Mutex` on the manager | Keep this repository's |
| `models/control.go` | no `NameKey` | `NameKey string` with tag `json:"-"` | Keep this repository's |
| `provision.go` | `Provision` only migrates | `Provision` migrates, then `backfillControlNameKeys` fills the key of every control stored without one | Keep this repository's. The hub database gets the new column and unique index the first time `Provision` runs after the import; the backfill can refuse a database that already holds two controls whose names fold to the same key, which needs a look before the hub's first run |
| `riskmanager.blueprint.json` | no `name_key` field or index on `control` | `name_key` field and a unique partial index `name_key` (rows with an empty key left out) | Keep this repository's. The hub's `riskmanager.wakala.json` is identical to this repository's, so no spec change follows |

## Only in this repository

| Files | Reason | Disposition |
|---|---|---|
| `control_name_key.go` and its tests (`control_name_key_test.go`, `control_concurrent_test.go`, `control_owner_lock_test.go`, `control_two_managers_test.go`, `control_unique_violation_test.go`) | The control-name key, its backfill and the tests of the race the key closes (see [control-name-race.md](control-name-race.md)) | Stays here |
| Root-package tests beyond the hub's (`action_test.go`, `assessment_test.go`, `barrier_test.go`, `bowtie_test.go`, `gaps_*_test.go`, `identity_owner_test.go`, `owners_test.go`, `pathway_test.go`, `risk_test.go`, `seed_test.go`, `testdb_test.go`, `mount_test.go`, `examples_test.go`, `domain_agnostic_test.go` and the rest) | A wider suite over code that is identical on both sides | Stays here |
| `mcp/mcp_test.go`, `mcp/tool_conversation_meta_test.go`, `routes/routes_test.go`, `routes/gaps_query_test.go`, `routes/controls_failing_query_test.go`, `routes/tools_cover_test.go` | The same, for the tool and route packages | Stays here |
| `postgres_integration_test.go` | The Postgres suite behind the `integration` build tag; the hub tests on the memory backend only | Stays here |
| `library/`, `spec/` | The shipped risk libraries and example domains the tests read; the hub dropped its libraries again (b27efe61) | Stays here |
| `cmd/ddl`, `scripts/`, `documentation/` | Tooling and documentation of the module | Stays here |

## Pinned counts

The module declares 33 operations in `riskmanager.operations.json`, in this repository and in the hub copy (33 since `match_names`, 83963ee8). `mcp/mcp_test.go` and `routes/routes_test.go` pin 33. `routes/tools_cover_test.go` still pinned 32 and failed against the unchanged code; it now pins 33 and its first test is named `TestRoutesAndRoutesForBuildTheSameThirtyThreeActions`. No assertion was weakened: the value moved to what the operations file declares.

## Behaviour conflicts

None. Where the two sides differ they differ in this repository's favour: the hub answers a raced control rename with the raw constraint error and this repository answers the conflict. No operation behaves one way in the hub and another way here for a reason other than that missing fix.

## Hub wiring that is not part of the module

These stay in the hub and are not moved. They are the places the import has to touch when the hub switches from its in-tree path to this module's path.

| Hub file | Role |
|---|---|
| `internal/accounts/router.go` | `RiskManager(ctx, id)` on the router |
| `internal/agencies/registry.go`, `internal/agencies/caches.go`, `internal/agencies/test_copy.go` | Builds the manager per instance with `SpecForMount`, caches it, copies it into a test instance |
| `internal/api/http/risk_routes.go`, `operation_routes.go` | Mounts the module's routes and operations on the HTTP layer |
| `internal/api/http/risk_admission.go`, `membership.go` | Admission and membership checks that resolve the manager |
| `internal/api/http/approvals_apply.go`, `approvals_apply_risk.go` | Applies approved risk changes through the module's models |
| `internal/api/http/mcp_tools_risk.go` | The tool filter and resolver over the module's tools |
| `internal/api/http/misuse_removedmember_sweep_test.go`, `internal/api/http/group_role_risk_owner_test.go` | Hub tests that exercise the module through the mount |
| `modules/agency/models/permissions_catalog.json` | The permission catalogue that names `riskmanager.*` actions |

## For the import

- `go.mod` here carries a `replace` of the shared module to a sibling folder. The tagged release the hub requires must not carry it, and the shared module must be tagged first.
- This repository's `control_name_key.go` calls `plaintext.FoldDrawnName`, which the hub's shared copy and the shared repository both have.
- Tagging and pushing wait for the owner's explicit yes.
