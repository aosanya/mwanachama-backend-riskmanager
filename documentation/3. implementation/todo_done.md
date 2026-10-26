# mwanachama-backend-riskmanager — completed tasks

| Task | Title | Completed | Notes |
|------|-------|-----------|-------|
| RM0 | Build the repo | 2026-10-26 | New repo, `mwanachama-backend-riskmanager`, following the `mwanachama-backend-<domain>` naming and the declared-domain shape of `mwanachama-backend-digitaltwin`. Source: the owner's AI safety bow-tie conversation, [sources.md](../1.%20requirements/sources.md). What shipped, one line each: |
| RM9 | Rank the gaps and cut the noise on a fresh register | 2026-10-26 | A barrier placed and not yet judged used to raise three gaps at once (design, coverage, untested): on the data-leak example 60 of 64 gaps, burying the three unguarded threats and the one failing test. Now every gap carries a `severity` (`GapKind.Severity()`: **high** `pathway_unguarded`, `barrier_failing`, `action_overdue`; **medium** `risk_unowned`, `risk_one_sided`, `risk_unrated`, `control_unowned`, `action_undated`, `barrier_design`, `barrier_coverage`; **low** `barrier_unassessed`, `barrier_untested`) and `Bowtie.Gaps` and `Gaps` return them ranked high to low, stably, across the whole register before filtering. `barrier_design` and `barrier_coverage` fire only for a judgement of `partial` or `inadequate`; a judgement not yet made is one new `barrier_unassessed` gap per barrier that names which of the two is missing, so there are twelve kinds. `GapFilter` gains `severity`, `limit` (200 when not positive) and `offset`, applied after risk and kind, and an unknown `kind` or `severity` is a 400 rather than an empty list. The data-leak parent risk now answers 44 gaps — the four that matter first, then 40 low — and the whole register 53, 10 of them high. The paging half of RM5 is done; what remains of RM5 is on [todo.md](todo.md). Tests: `gaps_severity_test.go`, `gaps_paging_test.go`, `gaps_judgement_test.go`, `data_leak_gaps_test.go` and `routes/gaps_query_test.go`, including one that reads the constants out of `models/gap.go` so a kind cannot be added without a severity. |
| RM10 | Make the module mount-ready | 2026-10-26 | Every module a host mounts per agency ships a default domain and two helpers; this one now does. `riskmanager.wakala.json` fills each of the seven roles under its own name with plural tables; `SpecFor(instance)` and `SpecForMount(instance, mount)` stamp the host's instance (and mount) into it and validate the result, and a name with an underscore is refused by the engine. The shipped specs walked by the guard tests now include it. `mount_test.go` holds the contract a host relies on: two mounts and two instances in one database keep separate tables and a manager over each reads only its own rows. Mounting itself is RM1. |

## RM0 — what shipped

| Piece | What it is |
|---|---|
| Blueprint | [riskmanager.blueprint.json](../../riskmanager.blueprint.json): seven roles — `risk`, `pathway`, `control`, `barrier`, `evidence`, `assessment`, `action` — every field described, enums declared, indexes for the register listing, the pathway sides, one control per pathway, and evidence and ratings in time order. |
| Models | `models/`: the types, the vocabularies Go compares against, the state machines, the 1–5 scale with its score and bands, and the view types (`Bowtie`, `Arm`, `Placed`, `Measure`, `PathwaySummary`, `Gap`). |
| Manager | `RiskManager` and its methods: the lifecycle rules (owner before open, residual rating before accepted, retired frozen), one control once per pathway, maturity gated by evidence, removal and delete guards, unique control names. |
| The five questions | `Threats`, `Consequences`, `PreventiveMeasures`, `MitigatingMeasures` and `Reliance`, each with its filters, and `Placed.Verdict` giving every measure a standing of `proven`, `weak`, `unproven` or `failing`. |
| Views | `Bowtie` (the whole diagram with drill-downs, current ratings and actions) and `Gaps` (twelve derived gap kinds, ranked high to low and pageable since RM9). |
| Operations | [riskmanager.operations.json](../../riskmanager.operations.json): thirty-two operations, deny by default, caller-bound `recorded_by` and `assessed_by`, an error table that maps every sentinel. |
| Routes and MCP | `routes/` as an adapter over `dispatch.Table` and `dispatch.Tools`; `mcp/` registering one tool per operation with an `instance_id`. |
| Provisioning | `Provision(db, spec)` over `spec.Migrate`, and `cmd/ddl` printing a spec's DDL for review. |
| Domains | `spec/examples/ai_safety.riskmanager.json` (with an `ai_system` object of its own) and `food_safety.riskmanager.json`. |
| Tests | Behaviour tests per object, the five questions, routes, MCP, and the guard tests the standard asks for, plus the Postgres integration test behind `-tags=integration`. First full run is recorded in [4. qa](../4.%20qa/README.md). |
| Documentation | This folder and the root `README.md` and `CLAUDE.md`. |

## Archived board context

`mwanachama-backend-riskmanager` — Go library, module
`github.com/aosanya/mwanachama-backend-riskmanager`. Nothing consumes it yet;
mounting is RM1 on [todo.md](todo.md).

### Why this repo exists

The owner shared a conversation about drawing an enterprise AI safety
bow-tie and then drilling into data leakage: the top event, eight threat
pathways, the barriers on them, four questions to ask of each barrier, and
the enterprise risk management attributes that turn the diagram into a
register. It was asked for as a repo that looks like the digital twin,
`comm` and the asset manager — declared objects, a declared route table,
thin routes and MCP over the same table — and as a module that answers
plain questions. The conversation's domain, AI safety, is one of two the
module ships; the module itself names none.
