# The five questions

The module exists to answer these without the caller joining anything. Each
is its own operation, so it is its own REST address, its own action id a
mount can grant, and its own MCP tool, generated from the same operations
file.

| Question | Operation | Address | MCP tool |
|---|---|---|---|
| What are the threats? | `list_threats` | `GET /threats` | `riskmanager_threat_list` |
| What are the consequences? | `list_consequences` | `GET /consequences` | `riskmanager_consequence_list` |
| What are the preventive measures against…? | `list_preventive_measures` | `GET /preventive-measures` | `riskmanager_preventive_measure_list` |
| What are the mitigating measures against…? | `list_mitigating_measures` | `GET /mitigating-measures` | `riskmanager_mitigating_measure_list` |
| What relies on this control? | `list_reliance` | `GET /controls/{controlID}/reliance` | `riskmanager_control_reliance` |

The tool name is the action id with dots as underscores. Every tool also
takes an `instance_id`, resolved to the `RiskManager` for that call, as in
the digital twin's `mcp/` package.

Two more operations answer a wider question. `get_bowtie`
(`GET /risks/{riskID}/bowtie`, `riskmanager_risk_bowtie`) returns the whole
diagram of one risk. `list_gaps` (`GET /gaps`, `riskmanager_gap_list`)
returns what is missing or weak, most serious first; see [gaps.md](gaps.md). Finding a risk by
words is `list_risks` with `query`, which matches the title, top event and
hazard.

## 1 and 2: threats and consequences

`GET /threats` and `GET /consequences` return pathway summaries.

| Filter | Meaning |
|---|---|
| `risk_id` | Only this risk's pathways. An unknown risk is a 404. |
| `query` | Case-insensitive text matched against the pathway's title and scenario. |
| `limit` | At most this many; 200 when left out. |

Each row is the pathway's own fields plus `risk_title`, `barrier_count` and
`drilldowns`. The count is the point: a threat with `barrier_count: 0` has
nothing standing on it, so the answer to "what are the threats" also says
which of them are open. `drilldowns` counts the risks that expand the
pathway.

## 3 and 4: preventive and mitigating measures against

`GET /preventive-measures` returns the barriers standing on **threat**
pathways. `GET /mitigating-measures` returns the barriers standing on
**consequence** pathways. "Against" is whatever the caller narrows to.

| Filter | Meaning |
|---|---|
| `risk_id` | Every measure against this risk, on that side. An unknown risk is a 404. |
| `pathway_id` | Measures against this one threat (or consequence). An unknown pathway is a 404; a pathway on the wrong side — a consequence asked of the preventive question — is a 400. |
| `query` | Text matched against the pathway's title and scenario and the control's name and description. Applied after the measures are loaded. |
| `limit` | At most this many; 200 when left out. |

With neither `risk_id` nor `pathway_id` the question is asked of the whole
register. A `pathway_id` that does not belong to the given `risk_id` returns
an empty list.

Each row is a **measure**: `risk_id`, `risk_title`, the `pathway`, the
`barrier` (design, coverage, maturity, notes), the `control`, the
`latest_evidence`, the `evidence_count` and a `standing`.

## What a standing means

The standing says how far a measure can be trusted, in the order the tests
are applied — the first that holds wins.

| Standing | Holds when |
|---|---|
| `failing` | The latest evidence has an outcome that is not `pass` — that is, `fail` or `partial`. |
| `unproven` | There is no evidence at all. |
| `weak` | There is evidence and the latest passed, but the barrier's `design` or `coverage` is anything other than `adequate`, including `unassessed`. |
| `proven` | The latest evidence passed, and both design and coverage are `adequate`. |

A measure that has failed and also has weak coverage is `failing`; the
worse fact is the one reported. `proven` is not "has a control on it": it is
that someone judged the design and the coverage adequate and the newest
evidence agrees. Maturity does not enter the standing; a barrier whose
maturity is below `tested` shows up as a gap instead.

The standing is computed on each read and never stored.

## 5: what relies on this control

`GET /controls/{controlID}/reliance` returns a measure for every barrier the
control stands in, across every risk, in the same shape as above. An unknown
control is a 404; a control that stands nowhere returns an empty list.

This is the common-cause question. If one control is `failing` on several
barriers, the same weakness is behind several risks, and fixing the control
once fixes them all. The risks are named in each row, so the answer reads as
a list of what is exposed.

## Worked examples

These use the data-leak bow-tie from the source conversation. The pathway and
barrier names are the conversation's. The consequences, the evidence and the
standings are **illustrative**: the conversation's consequences were drawn
only in an image, and no evidence was given. See
[sources.md](../1.%20requirements/sources.md).

The risk is "Loss of control over sensitive enterprise data through AI".

**What are the threats?**
`GET /threats?risk_id=<id>` returns eight rows: prompt input leakage; RAG /
retrieval leakage; prompt injection; agent / tool misuse; model output
leakage; provider and integration leakage; logging and observability
leakage; cross-tenant / session leakage. If nothing stands on agent / tool
misuse yet, its row says `barrier_count: 0` and no other call is needed to
see that the route is open.

**Which threats mention retrieval?**
`GET /threats?query=retrieval` returns the RAG row from any risk that has
one, and the title of the risk it belongs to.

**What are the consequences?**
`GET /consequences?risk_id=<id>` returns whatever has been drawn on the right
of that risk, each with its barrier count.

**What are the preventive measures against RAG leakage?**
`GET /preventive-measures?pathway_id=<rag pathway>` returns the barriers on
that threat — in the conversation's table: document-level authorisation,
permission-aware retrieval, tenant isolation — each with its standing. If
permission-aware retrieval has a passing test and `adequate` design and
coverage it is `proven`; if it has never been tested it is `unproven`.

**What are the mitigating measures against this risk?**
`GET /mitigating-measures?risk_id=<id>` returns the barriers on the
consequence side. The conversation names the kinds: revoking access,
containing exposure, notifying affected parties, recovering.

**What relies on scoped credentials?**
`GET /controls/<id>/reliance` returns the barrier on prompt injection and
the barrier on agent / tool misuse, if both rely on it, with the risk each
sits under. A failing test against the control shows as `failing` on both.

**One question, one tool.** An agent asked "which preventive measures
against prompt injection are unproven?" calls `riskmanager_threat_list` with
`query: "injection"` to find the pathway, then
`riskmanager_preventive_measure_list` with that `pathway_id`, and reads the
`standing` of each row.
