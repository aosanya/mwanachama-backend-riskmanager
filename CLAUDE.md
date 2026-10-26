# CLAUDE.md

Guidance for Claude Code working in this repository.

## Project: mwanachama-backend-riskmanager

A standalone risk register built on the bow-tie method: a centre event, the
threats that lead to it, the consequences that follow it, the measures that
stand on each route, and the evidence that those measures work. Module path
`github.com/aosanya/mwanachama-backend-riskmanager`. It began from the
owner's enterprise AI safety bow-tie, recorded in
[documentation/1. requirements/sources.md](documentation/1.%20requirements/sources.md).
**Enterprise AI safety is a domain this module runs, not what this module
is**; it ships as `spec/examples/ai_safety.riskmanager.json`, beside a food
safety domain.

## Not part of the electorate platform

No tie to Mwanachama's tier/member/campaign model. This is a domain-agnostic
backend service in the same family as `mwanachama-backend-digitaltwin`,
`mwanachama-backend-assetmanager` and `mwanachama-backend-taskmanager` — a
tool, not an extension of the tier-mobilization product.

Domain logic AND storage both live in this package, imported directly by
whatever mounts it — no separate service, no gRPC, no proto.

## Objects are declared, not written

The tables come from a JSON spec, not from Go row structs — see
[documentation/2. design/declared-domain.md](documentation/2.%20design/declared-domain.md)
and `developer/documentation/2. design/architecture-spec-driven-modules.md`.

- `riskmanager.blueprint.json` — **the module's seven objects, declared
  once**: `risk`, `pathway`, `control`, `barrier`, `evidence`, `assessment`,
  `action`. Embedded into the root package and reached through
  `Blueprint()`, `LoadSpec(path)` and `ParseSpec(raw)`. Load a domain spec
  through those, never through `spec.Load`, or its roled objects arrive with
  no fields.
- `mwanachama-backend-shared/spec`, `specstore` and `dispatch` are the
  engine. `spec.Migrate` creates a table per declared object and its
  indexes; `Provision(db, s)` is this module's name for it, and `cmd/ddl`
  prints the statements so a spec can be reviewed as SQL before it is
  trusted.
- `spec/examples/ai_safety.riskmanager.json` and
  `food_safety.riskmanager.json` — the same module under two domains. The
  AI safety spec declares an object of its own, `ai_system`, with no role;
  the food safety spec is a domain nobody has provisioned, so the shipped
  example is not also production config.
- `riskmanager.wakala.json` — **the default domain a mounting host uses**,
  as `engage`, `voting`, `files` and `sources` ship theirs. It fills each of
  the seven roles under its own name (`risk`, `pathway`, `control`,
  `barrier`, `evidence`, `assessment`, `action`) with plural tables, so it
  adds no vocabulary. It is embedded and reached through
  `SpecFor(instance)` and `SpecForMount(instance, mount)`, which stamp the
  host's instance (and mount) into it and validate the result; the host then
  calls `Provision(db, s)` and `NewRiskManager(db, s)`. **Nothing in
  `mwanachama-wakala-api` is wired yet.** Mounting is row RM1: it changes a
  sibling repo's registry and route table, so it waits for the owner. An
  instance or mount name carries no underscore — the engine refuses one.

**A domain names objects; it does not re-declare them.** Its spec supplies
`instance`, the name and table each role lands in, its own indexes, and a
**default** on a declared field — nothing else. A domain needing a field of
its own declares an object of its own, with no role.

## The rule that matters most: this module names no domain

**A word that means something in one domain and nothing in another does not
belong in this module.** It was started from an AI safety bow-tie and must
run a food chain, a construction site or a hospital ward without a line
changing. Bow-tie method words — threat, consequence, barrier, control,
evidence, hazard, top event, preventive, mitigating — are the module's own
vocabulary and stay. Words like prompt, retrieval, leak, model, agent,
vendor, batch or contamination are a domain's, and live in its spec, in a
`doc` document, or in an object the domain declares itself.

`domain_agnostic_test.go` enforces the rule over four surfaces: identifiers
in `models/`, `routes/` and the root package; every stored enum value; every
blueprint role, field, value and index; and every path and action in the
declared route table.

## Lifecycle rules live in Go

A spec cannot say these, so each stays with the type or manager method it is
about:

- `Status.CanTransitionTo` — `draft → open | retired`,
  `open → accepted | retired`, `accepted → open | retired`; `retired` is the
  end. A new risk is always created as a draft, whatever status it names.
- **`draft → open` needs an owner**, and an open or accepted risk cannot be
  updated to have none.
- **`open → accepted` needs a residual assessment** on record.
- **A retired risk is frozen.** Every write to its pathways, barriers,
  evidence, assessments and actions is refused with `ErrConflict`.
- **`parent_pathway_id` is fixed at creation**, so the drill-down tree can
  never be rewired into a loop; the parent pathway's risk must not be
  retired.
- **A control stands once per pathway** (unique `pathway_control` index, and
  a pre-check that answers `ErrConflict`).
- **A barrier cannot reach `tested` or `monitored` with no evidence**, on
  placement or on update.
- **A barrier with evidence or actions cannot be removed**; a pathway with
  barriers or drill-down risks cannot be deleted.
- **A control name is unique**; renaming onto a taken name is `ErrConflict`.
- **An assessment's likelihood and impact are each 1 to 5.**
- **An action's status moves `open ⇄ in_progress → done | dropped`**;
  `done` and `dropped` are the end and set `closed_at`.

`TestVocabularyMatchesTheBlueprint` holds `models/vocabulary.go` and the
blueprint's declared `values` to each other in **both** directions.

## What is derived, never stored

Nothing here is a column, so nothing can drift from the facts it summarises:

- **Gaps** (`Gaps`, `Bowtie.Gaps`) — computed from the bow-tie on each call:
  twelve kinds, each with a severity from `GapKind.Severity()`, which is the
  single source. They are ranked high, medium, low (stable within a
  severity) across the whole register, then filtered by risk, kind and
  severity, then paged by `offset` and `limit` (200 when none is given). An
  unknown kind or severity is a 400, not an empty list. A barrier whose
  design or coverage is still `unassessed` raises one `barrier_unassessed`
  gap; `barrier_design` and `barrier_coverage` fire only for `partial` or
  `inadequate`.
- **Score and band** — likelihood × impact, banded low (1–4), medium (5–9),
  high (10–15), critical (16–25), in `models/assessment.go`.
- **Standing** of a measure — `proven`, `weak`, `unproven`, `failing`, from
  the barrier's judgements and its latest evidence (`Placed.Verdict`).
- **A barrier's side** — read from its pathway's `side`, never stored on it.
- **Latest evidence and the current rating** — the newest by `recorded_at`
  and `assessed_at`.

`evidence` and `assessment` are **append-only**: nothing edits or deletes
what was said. A wrong or outdated record is corrected by recording another.

## The route table is declared

`riskmanager.operations.json` declares all thirty-two addresses, and
`routes/routes.go` is an adapter over `mwanachama-backend-shared/dispatch`.
There are no hand-written handlers. See
[documentation/2. design/routes.md](documentation/2.%20design/routes.md).

`AnonymousActions` is empty: this module publishes nothing anonymously, and
an operation added later and not named there arrives gated (deny by
default). The module has **no auth model of its own** — a mount supplies the
`dispatch.Authorizer`. `recorded_by` and `assessed_by` are bound from the
caller, never read from a request body.

The MCP tools are the same table: `routes.BuildTools` runs
`dispatch.Tools` over the same operations file, so a REST address and its
tool cannot disagree.

## Storage invariants

- **A column is found by field name, never by json tag.** `RiskID` is
  `risk_id`. A tag-reading codec silently stops storing any `json:"-"`
  field.
- **Every declared column is written on every write.** A map missing a key
  means "leave it alone" to an update, so omitting empty values would make
  clearing a field impossible.
- **A required field must not carry a default.**
  `TestRequiredFieldsHaveNoDefault` asserts it over the blueprint and every
  shipped spec.
- **Timestamps are RFC 3339 strings, not `time.Time`**, because the codec
  stores a timestamp as text.
- **Stamps the module generates are fixed-width microsecond UTC**
  (`2006-01-02T15:04:05.000000Z07:00`), unlike the second-resolution stamps
  of sibling modules. This is what makes "latest evidence" and "current
  rating" deterministic when two records arrive within one second.
  Caller-supplied times in plain second resolution still parse, and tie
  within the same second.
- **A table is `<instance>_hashOf(<mount>)_hashOf(<module>_<object>)`.**
  Assert on `spec.RawNameFor` in tests, never on a physical name literal,
  and exclude `%_spec_table_names` from anything that counts tables.
- **Identifiers reach SQL as text** — `spec.NamePattern` and
  `spec.DocPathPattern` refuse anything outside a strict alphabet. Never
  relax either to "escape it instead". Free-text search (`query`) escapes
  `%`, `_` and `\` and lowercases both sides.

## Guard tests

The same six the declared-domain standard asks of every converted module:
`domain_agnostic_test.go`; `TestEveryExampleFitsTheTypes` (a manager over
every shipped spec, the wakala default included); `TestTwoDomainsCoexist`;
`TestRequiredFieldsHaveNoDefault`;
`TestVocabularyMatchesTheBlueprint`; and `postgres_integration_test.go`
(`//go:build integration`, gated on `POSTGRES_URL`, run by `make test-pg`,
not part of `go test ./...`). Routes also carry
`TestEverySentinelIsMappedToAStatus`.

## Scope decisions (session of 2026-10-26)

- **A bow-tie is one `risk`; threat and consequence are one `pathway` with a
  `side`.** The first sketch had eight objects with a separate `threat` and
  `consequence`; they had identical fields, so they merged and a barrier
  needs one pathway reference rather than one of two.
- **`control` is shared and `barrier` is the claim.** A control is declared
  once and may stand on many pathways of many risks; what gets assessed is
  the placement, because a control that works in one place may not reach
  another. This is what makes a common-cause failure visible.
- **Owners are opaque strings** (`risk.owner`, `control.owner`,
  `action.owner`) the mount fills with its own reference to a person or
  role. Wiring them to the actor module is an open row.
- **Actions are lightweight.** `mwanachama-backend-taskmanager` owns work;
  an action here is a commitment attached to a gap, with an owner and a due
  date, not a task tracker.

## Status (2026-10-26)

New repo, built 2026-10-26, and mount-ready the same day: it ships
`riskmanager.wakala.json` with `SpecFor` and `SpecForMount`, and
`mount_test.go` holds the contract a host relies on (two mounts and two
instances in one database keep separate tables, and a manager over each
reads only its own rows). Nothing mounts it yet: no wakala-api wiring (RM1,
which changes a sibling repo and waits for the owner), no studio or app
screen. Open work is on
[documentation/3. implementation/todo.md](documentation/3.%20implementation/todo.md).

## Conventions

- Task status lives on
  [documentation/3. implementation/todo.md](documentation/3.%20implementation/todo.md).
- Four-phase `documentation/` layout — see
  [documentation/README.md](documentation/README.md).
- `go test ./...` (sqlite via `glebarez/sqlite`) is the expected way to
  verify a change here — do not reach for a real Postgres. The `jsonb`
  behaviours SQLite cannot stand in for live in `postgres_integration_test.go`
  (`//go:build integration`, gated on `POSTGRES_URL`, run by `make test-pg`),
  which is **not** part of `go test ./...`.
- No Go file over 300 lines; split by responsibility.
- There are no hand-written route builders: an address is an entry in
  `riskmanager.operations.json` plus a method on `RiskManager`.

## Code comments

Write code with no comments. Not one-liners above a function, not section
banners, not doc comments on exported symbols, not "why" notes next to a
tricky line. A name, a type, or a smaller function carries it instead.

Anything that genuinely needs explaining goes in this repo's `documentation/`
folder, under the phase it belongs to (`1. requirements`, `2. design`,
`3. implementation`, `4. qa`) — never inline.

**Why:** inline prose drifts out of sync with the code, duplicates what
`documentation/` already owns, and buries the explanation where nobody
looking for it will search.

**How to apply:**

- New code ships without comments. If a line seems to need one, rename or
  split until it doesn't.
- Touching code that already has comments: strip the ones in the code you are
  changing. Do not sweep untouched files unless asked.
- If the reasoning matters, add or update the matching `documentation/` page
  in the same change and leave nothing behind in the source.
- Machine-read directives are not comments and stay: build tags, `//go:embed`,
  `//go:generate`, linter pragmas (`//nolint`, `// eslint-disable-next-line`,
  `// ignore:`), license headers, codegen "do not edit" banners, and generated
  files as a whole.
- Commit messages, PR descriptions, and test names carry the narration that
  used to go in comments.

This rule is repeated verbatim in every mwanachama repo's `CLAUDE.md` so that
it reaches sessions that do not load this machine's user-level config —
scheduled cloud routines, other machines, and other agent harnesses.
