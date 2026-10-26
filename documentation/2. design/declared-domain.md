# The objects are declared

The tables come from [`riskmanager.blueprint.json`](../../riskmanager.blueprint.json),
not from Go row structs. The engine is `mwanachama-backend-shared`'s `spec`,
`specstore` and `dispatch`; the org-wide strategy is
`developer/documentation/2. design/architecture-spec-driven-modules.md`.

The repo was built declared from the start, so there is no conversion to
record. What follows is what the blueprint states, what it cannot, and what
the module deliberately does not name.

## The module names no domain

A word that means something in one domain and nothing in another does not
belong here. The module was begun from an AI safety bow-tie and has to run a
food chain, a building site or a hospital ward without a line changing.

| Kept in the module | Why | Where a domain's word goes |
|---|---|---|
| threat, consequence, pathway, barrier, control, evidence, assessment, action, hazard, top event | the bow-tie method's own words; a reader from any domain recognises them | — |
| `preventive`, `mitigating` | read off a pathway's side | — |
| `risk.doc`, `pathway.doc`, `control.doc` | free documents, queried in place by path | a category, a business unit, a regulation, the system a control runs in |
| prompt, retrieval, leakage, model, agent, vendor | AI safety's words | the AI safety spec, or `doc` |
| batch, contamination, allergen, supplier | food safety's words | the food safety spec, or `doc` |

The guard test reads identifiers, every stored enum value, every blueprint
role, field, value and index, and every path and action in the route table,
and fails on a domain word.

## Why a risk has no `category`

A register is normally grouped by category, but categories are a domain's
word — operational, strategic, regulatory in one; hygiene, allergen, supply
in another. A domain spec may set a default and nothing else, so a value set
a domain supplies is not available, and a free `category` column the module
never interprets would only be a second `doc`. The category lives in
`risk.doc`, and a domain that filters by it declares a path index: the AI
safety spec indexes `doc.category` on its `bowtie` object, the food safety
spec indexes `doc.batch_scope` on its `hazard_event`.

## Fields a domain can and cannot touch

A domain's spec supplies `instance`, the name and table each role lands in,
its own indexes, and a **default** on a declared field. Setting a type, a
description or a value set on a declared field is refused by name when the
spec loads.

A domain needing a field of its own declares an object of its own, with no
role, and no rule in the module touches it. The AI safety spec does this
with `ai_system` — one application, model or agent the enterprise runs — so
that coverage ("is the control deployed across every system it applies to?")
has something to be judged against. The module does not know the object
exists.

## What the blueprint declares

`required`, `immutable`, `unique`, enum `values` and `matches` are read off
the blueprint by `validate.go` and applied on the way in. `patterns.go` maps
the one pattern name this module supplies — `name` — to `spec.NamePattern`.

| Object | Required | Fixed after creation | Unique | Enums |
|---|---|---|---|---|
| `risk` | `title`, `top_event` | `parent_pathway_id` | — | `status` |
| `pathway` | `risk_id`, `title` | `risk_id`, `side` | — | `side` |
| `control` | `name` | — | `name` | — |
| `barrier` | `pathway_id`, `control_id` | `risk_id`, `pathway_id`, `control_id` | `(pathway_id, control_id)` | `design`, `coverage`, `maturity` |
| `evidence` | `barrier_id`, `summary` | — | — | `kind`, `outcome` |
| `assessment` | `risk_id` | — | — | `kind` |
| `action` | `risk_id`, `title` | `risk_id`, `barrier_id` | — | `status` |

An enum with no default is implicitly required: an empty value is not one of
its values and is refused. `evidence.kind`, `evidence.outcome`,
`assessment.kind` and `pathway.side` therefore must be supplied.

## Vocabulary

| Field | Values |
|---|---|
| `risk.status` | `draft` (default), `open`, `accepted`, `retired` |
| `pathway.side` | `threat`, `consequence` |
| `barrier.design` | `unassessed` (default), `adequate`, `partial`, `inadequate` |
| `barrier.coverage` | `unassessed` (default), `adequate`, `partial`, `inadequate` |
| `barrier.maturity` | `proposed` (default), `designed`, `deployed`, `tested`, `monitored` |
| `evidence.kind` | `test`, `log`, `audit`, `configuration`, `incident` |
| `evidence.outcome` | `pass`, `partial`, `fail` |
| `assessment.kind` | `inherent`, `residual`, `target` |
| `action.status` | `open` (default), `in_progress`, `done`, `dropped` |

`TestVocabularyMatchesTheBlueprint` holds `models/vocabulary.go` and these
declared values to each other in both directions.

## What stays in Go

Only what a spec cannot say, each piece with the type it is about:

- `Status.CanTransitionTo` and `ActionStatus.CanTransitionTo` — state
  machines, not value sets. `Status.Frozen` — a retired risk takes no
  changes.
- `Maturity.NeedsEvidence` — `tested` and `monitored` need a piece of
  evidence, a check that reads another table.
- `Assessment.OnTheScale` — likelihood and impact are each 1 to 5. `Score`
  and `BandOf` — the matrix, banded low 1–4, medium 5–9, high 10–15,
  critical 16–25.
- The manager's cross-object rules — an owner before a risk opens, a
  residual rating before it is accepted, a barrier's pathway and control
  existing, a barrier with evidence not removable. These are in
  [CLAUDE.md](../../CLAUDE.md).
- `Placed.Verdict` — a measure's standing, from the barrier and its evidence.
- `Bowtie.Gaps` — every derived gap; see [gaps.md](gaps.md).

`Check(spec, role, value)` is the same validation without a database, for a
bulk import validating what it has read before it opens a connection.

## Timestamps are strings

`spec.TypeTimestamp` is stored as text, so every dialect compares it the same
way, and the codec reads the field with `reflect.Value.String()`. Every
timestamp on a model is an RFC 3339 `string`. Module-generated stamps use a
fixed-width microsecond UTC layout so that text order is time order, which
is what keeps "latest evidence" and "current rating" deterministic.

## Two domains, and one of them is unprovisioned

`spec/examples/ai_safety.riskmanager.json` is the origin domain: the `risk`
role lands as `bowtie`, `action` as `remediation`, `evidence` with the table
`evidence_records`, plus the domain's own `ai_system` object and a path index
on `doc.category`.

`spec/examples/food_safety.riskmanager.json` is produce reaching a buyer
unfit to eat, which nobody has provisioned — the shipped example must not
also be production config. It names its objects `hazard_event`, `route`,
`control_measure`, `control_point`, `check`, `rating` and
`corrective_action`, and indexes `doc.batch_scope` on the hazard event.

`TestEveryExampleFitsTheTypes` builds a manager over every spec in
`spec/examples/`, not the one a test happened to load, and
`TestTwoDomainsCoexist` migrates both into one database and proves a write
under one is invisible to the other.

## The wakala domain, and how a host reaches it

`riskmanager.wakala.json` is the third spec the module ships and the only one
that is not an example: the default domain a mounting host uses, as `engage`,
`voting`, `files` and `sources` ship theirs. It fills the seven roles under
their own names, with plural tables — `risk`/`risks`, `pathway`/`pathways`,
`control`/`controls`, `barrier`/`barriers`, `evidence`/`evidence_records`,
`assessment`/`assessments`, `action`/`actions` — so it adds no vocabulary the
platform has to learn and carries no domain word. It declares no object of
its own and no index. Its `instance` of `app` is a placeholder the host
replaces.

A host does not read the file; it calls one of two helpers in `blueprint.go`
and the module does the rest:

| Call | What it returns |
|---|---|
| `SpecFor(instance)` | the wakala domain with `instance` stamped in, validated against the blueprint |
| `SpecForMount(instance, mount)` | the same with a mount segment as well; an empty mount keeps the engine's default |

The host then calls `Provision(db, s)` and `NewRiskManager(db, s)`. The
engine refuses an instance or mount that is not a name segment — lowercase
letters and digits, starting with a letter — with the error `is not a usable
name segment`, so `ai_safety` is refused and `aisafety` is not. That is why
the two example specs carry the domain label `ai_safety` and the instance
`aisafety`.

A table is `<instance>_hashOf(<mount>)_hashOf(riskmanager_<object>)`, so two
mounts of one instance and two instances of one mount never share a table.
`mount_test.go` asserts both through `TableFor` and `RawNameFor`, and that a
manager over each reads only its own rows. `TestEveryExampleFitsTheTypes`,
`TestTwoDomainsCoexist` and `TestRequiredFieldsHaveNoDefault` walk the wakala
spec along with the examples, and
`TestTheWakalaDomainNamesEveryRoleForItselfAndFillsEachOnce` holds it to one
object per role, each named for its role, with no domain word in a name or a
table.

Nothing mounts the module yet. Registering it in `mwanachama-wakala-api` is
RM1 on [todo.md](../3.%20implementation/todo.md); it changes that repo's
registry and route table, so it waits for the owner.
