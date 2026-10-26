# QA

Test coverage and results for `mwanachama-backend-riskmanager`.

## How to run

```sh
make test      # go test ./... — sqlite in memory, no database needed
make test-pg   # go test -tags=integration ./... — needs POSTGRES_URL
make vet       # go vet ./...
go run ./cmd/ddl spec/examples/ai_safety.riskmanager.json postgres
go run ./cmd/ddl spec/examples/food_safety.riskmanager.json sqlite
```

`go test ./...` is the expected way to verify a change. The `jsonb`
behaviours SQLite cannot stand in for live in `postgres_integration_test.go`
behind `//go:build integration`, gated on `POSTGRES_URL`; they are not part
of `go test ./...`. The integration test provisions under a random instance
name and drops its tables afterwards.

## Test layout

| Where | What it covers |
|---|---|
| `risk_test.go`, `pathway_test.go`, `barrier_test.go`, `evidence_test.go`, `assessment_test.go`, `action_test.go` | Each object's behaviour through `RiskManager`: the lifecycle and cross-object rules in [CLAUDE.md](../../CLAUDE.md), append-only records, the guards on removal and deletion |
| `questions_test.go` | The threat, consequence and measure questions and their filters: unknown risk, text search in any case with `%` and `_` literal, wrong-side pathway refused, another risk's pathway answers nothing, empty answers are `[]` and never null |
| `seed_test.go`, `testdb_test.go` | Shared builders and the in-memory database; no assertions of their own |
| `domain_agnostic_test.go` | The module names no domain, over identifiers, stored enum values, the blueprint and the route table |
| `examples_test.go` | A manager over every shipped spec, two domains in one database, no required field with a default, the vocabulary matching the blueprint both ways, every sentinel mapped to a status |
| `routes/routes_test.go` | The declared table builds, every route is gated until named anonymous, a refused action never reaches the manager, the address and the caller outrank the body, sentinels arrive at their declared status, deletes answer 204 and writes 201, the query filter binds whole |
| `mcp/mcp_test.go` | Every declared operation is a tool that names its instance, the caller is never offered to an agent, an agent's questions about a bow-tie work end to end, each instance reaches only its own register, a refused action never reaches the manager |
| `postgres_integration_test.go` | `jsonb` round trips and anything else SQLite cannot stand in for |

Each guard test should be seen to fail once: break the thing it guards, watch
it go red, put it back. The first round is recorded under Results.

## Results

Run of 2026-10-26, from a clean tree.

| Check | Command | Result |
|---|---|---|
| Unit and behaviour tests | `go test -count=1 -race ./...` | root, `routes` and `mcp` pass; 233 test functions and subtests, 0 failures |
| Vet | `go vet ./...` and `go vet -tags=integration ./...` | clean |
| Format | `gofmt -l .` | nothing listed |
| House rules | no Go file over 300 lines; no comments in code | both hold |
| Integration test | `make test-pg` | not run; it needs `POSTGRES_URL` |

**Guard tests seen to fail.** Eleven deliberate defects were applied one at a
time to a scratch copy of the repo, and each turned the intended test red. The
round was run on the first build and run again on the final one, after the
mount-ready spec and the gap ranking landed, with the same result:

| Defect | Test that went red |
|---|---|
| a domain word in an identifier | `TestNoDomainWordsInIdentifiers` |
| a domain word in a stored enum value | `TestNoDomainWordsInTheBlueprintsRolesFieldsAndStoredValues` |
| a domain word in a route path | `TestNoDomainWordsInTheDeclaredRouteTable` |
| a blueprint value with no Go constant | `TestVocabularyMatchesTheBlueprint` |
| a required field given a default | `TestRequiredFieldsHaveNoDefault` |
| a blueprint column no Go type carries | `TestEveryExampleFitsTheTypes` |
| two domains landing in one table | `TestTwoDomainsCoexist` |
| a sentinel with no declared status | `TestEverySentinelIsMappedToAStatus` |
| a route named anonymous | `TestEveryRouteIsGatedUntilNamedAnonymous` |
| the caller binding dropped from evidence | `TestTheCallerOutranksTheBody`, `TestTheCallerIsNeverOfferedToAnAgent` |
| the address no longer outranking the body | `TestTheAddressOutranksTheBody` |

The behaviour tests were mutation-tested while they were written, by the
forks that wrote them and not repeated here: fifteen deliberate defects on the
first build (the owner gate, the residual-rating gate, the retired freeze, the
evidence rule for `tested`, unescaped `%` in search, the standing order, the
evidence order and a band boundary among them), ten on the gap ranking and
paging, and five on the mount-ready spec (a role not named for itself, a
domain word in a table name, a role left unfilled, `SpecForMount` ignoring the
mount, and ignoring the instance). Each turned a test red. The gap round also
found that nothing exercised `Bowtie.Gaps` directly, because the manager's own
re-ranking hid it; a test was added for it.

## Not covered

- Anything above the route table: no authorisation model is exercised beyond
  the mount's `Authorizer` being asked, and nothing is mounted in wakala-api.
- Scale. `Gaps` is N+1 over risks and the measure questions' `query` filters
  after the load; neither has a test at hundreds of risks. See
  [gaps.md](../2.%20design/gaps.md) and RM5 on [todo.md](../3.%20implementation/todo.md).
- Evidence staleness, which is not yet a gap — RM2.
