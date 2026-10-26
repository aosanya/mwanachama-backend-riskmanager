# mwanachama-backend-riskmanager

A standalone risk register built on the bow-tie: one risk is a centre event,
the threats that can cause it on the left, the consequences that follow on
the right, and the measures that stand on each route between them. What
makes it more than a diagram is that every measure carries evidence, a
judgement of its design and coverage, and an owner, so the register can say
which measures are proven, which are only claimed, and where nothing stands
at all.

It names no domain. The module declares seven objects — `risk`, `pathway`,
`control`, `barrier`, `evidence`, `assessment`, `action` — and a domain spec
says what they are called and where they land. It ships two:

| Domain | Spec | What it models |
|---|---|---|
| AI safety | `spec/examples/ai_safety.riskmanager.json` | An enterprise's AI systems: the data-leakage bow-tie, with an `ai_system` object of the domain's own |
| Food safety | `spec/examples/food_safety.riskmanager.json` | Produce reaching a buyer unfit to eat: hazard events, control points, checks, corrective actions |

Beside them it ships [`riskmanager.wakala.json`](riskmanager.wakala.json), the
default domain a mounting host uses, with each role under its own name. A host
calls `riskmanager.SpecFor(instance)` (or `SpecForMount(instance, mount)`),
then `Provision` and `NewRiskManager`, as it does for `engage` and `voting`.
Nothing mounts the module yet.

## The five questions

The module answers these directly, over REST and as MCP tools generated from
the same operations file. Each answer says how strongly a measure actually
stands: `proven`, `weak`, `unproven` or `failing`.

| Question | Address | Tool |
|---|---|---|
| What are the threats? | `GET /threats` | `riskmanager_threat_list` |
| What are the consequences? | `GET /consequences` | `riskmanager_consequence_list` |
| What are the preventive measures against…? | `GET /preventive-measures` | `riskmanager_preventive_measure_list` |
| What are the mitigating measures against…? | `GET /mitigating-measures` | `riskmanager_mitigating_measure_list` |
| What relies on this control? | `GET /controls/{controlID}/reliance` | `riskmanager_control_reliance` |

Beside them: `GET /risks/{riskID}/bowtie` returns the whole diagram, and
`GET /gaps` returns what is missing or weak across it, worst first. See
[documentation/2. design/questions.md](documentation/2.%20design/questions.md).

No tie to Mwanachama's tier/member/campaign model. No gRPC, no sub-service
shape. Storage and domain logic both live in this package, imported directly
by whatever mounts it.

Objects come from [`riskmanager.blueprint.json`](riskmanager.blueprint.json)
and addresses from
[`riskmanager.operations.json`](riskmanager.operations.json); the engine is
[mwanachama-backend-shared](../mwanachama-backend-shared)'s `spec`,
`specstore` and `dispatch`. There are no row structs and no hand-written
handlers — see
[documentation/2. design/declared-domain.md](documentation/2.%20design/declared-domain.md)
and [routes.md](documentation/2.%20design/routes.md).

```sh
make build     # compile
make test      # unit tests, sqlite in memory
make test-pg   # the jsonb behaviours, needs POSTGRES_URL
go run ./cmd/ddl spec/examples/ai_safety.riskmanager.json postgres
```

See [documentation/](documentation/) for requirements, design and the task
board, and [CLAUDE.md](CLAUDE.md) for the invariants.
