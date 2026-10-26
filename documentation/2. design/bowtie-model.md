# The bow-tie model

A bow-tie is a picture of one loss of control: the event in the middle, the
ways it can come about fanning out to the left, and what follows fanning out
to the right, with measures drawn across each fan as vertical bars. The
module stores the picture as seven objects.

```
 threat pathways              top event              consequence pathways
 (side = threat)             (the risk)              (side = consequence)

 [pathway]--|barrier|--\                         /--|barrier|--[pathway]
 [pathway]--|barrier|---->   [ risk ]   <-------<
 [pathway]--|barrier|--/                         \--|barrier|--[pathway]
         preventive                                   mitigating
```

| Role | What one row is |
|---|---|
| `risk` | One bow-tie: a title, the top event stated precisely, the hazard, an owner, a status |
| `pathway` | One route on one side of one bow-tie, with a concrete scenario |
| `control` | One measure the organisation operates, declared once, with an owner |
| `barrier` | One control standing on one pathway, with judgements of its design, coverage and maturity |
| `evidence` | One piece of proof about one barrier: a test, a log, an audit, a configuration check, an incident |
| `assessment` | One rating of one risk — inherent, residual or target — on a 1–5 likelihood and impact scale |
| `action` | One piece of work to close a gap, with an owner, a due date and a status |

## Eight became seven

The first sketch had eight objects, with `threat` and `consequence` separate.
They had the same fields — a risk, a title, a scenario, a document — and
differed only in which side of the top event they sat on, so they became one
`pathway` with a `side` of `threat` or `consequence`. Two things follow:

- A barrier carries one `pathway_id`, not one of two nullable references
  with a rule that exactly one is set. There is no row that is both, or
  neither.
- Whether a barrier is **preventive** or **mitigating** is read from the
  side of its pathway and never stored on the barrier. A barrier on a
  threat prevents; a barrier on a consequence mitigates. A stored copy could
  disagree with the pathway; a read cannot.

`side` is fixed at creation. Every barrier on the pathway takes its meaning
from it, so changing it would silently turn preventive measures into
mitigating ones.

## A control is shared; a barrier is the claim

A control is declared once — "Input DLP", "Scoped credentials" — and has a
unique name and an owner. It stands on a pathway by way of a barrier, and
the same control may stand on many pathways of many risks.

What gets assessed is the barrier, not the control, because a control that
works in one place may not reach another. The source conversation's own
example is the right one: DLP on employee prompts, and nothing equivalent
on an agent's tool calls. The control exists; one route is guarded and
another is not. Design and coverage are therefore judgements on the barrier,
and evidence attaches to the barrier.

Sharing the control is also what makes a **common-cause failure** visible.
If "Scoped credentials" stands on four pathways in three risks and a test of
it fails, the question "what relies on this control" returns all four
barriers, with the risks and pathways they sit on. See
[questions.md](questions.md).

One control stands once per pathway: a unique index on
`(pathway_id, control_id)` and a pre-check that answers a conflict.

A barrier copies its `risk_id` from the pathway at placement. It is held
there so reading every barrier of a risk, or every risk a control serves,
does not join through the pathway on each read; it is fixed, so it cannot
disagree.

## Why evidence is append-only

A record that can be rewritten answers no question about the past. If a
passing test could be edited into a failing one, nobody could say when a
barrier stopped working, or whether the register knew. So `evidence` has no
update and no delete: a barrier that stopped working is shown by recording a
further, failing piece of evidence, and the **latest** by `recorded_at` is
the barrier's current word. The older passes stay.

`assessment` is append-only for the same reason: the history of a rating is
the history of how the organisation's view of the risk moved. The current
inherent, residual and target ratings are the newest of each kind.

Both carry `recorded_by` / `assessed_by` bound from the caller by the mount,
not read from the request.

The module-generated timestamps are fixed-width microsecond UTC, so "latest"
is deterministic when two records arrive in the same second. Caller-supplied
times in whole seconds still parse, and two of them in the same second tie.

## Why the maturity ladder is gated by evidence

`maturity` runs `proposed → designed → deployed → tested → monitored`. It is
an observed fact, not a workflow, so it can move in either direction. The one
rule is that a barrier cannot reach `tested` or `monitored` with no evidence
on record — a barrier cannot claim to have been tried with nothing recorded.
Any outcome counts: a failing test is still a test.

## Why gaps are derived

"Which pathways have nothing standing on them", "which barriers are
unevidenced", "which actions are overdue" are facts about the rest of the
register. Stored, they would be a second copy that could be wrong the moment
an evidence row arrived. So gaps, scores, bands and standings are computed
from the bow-tie on each call and have no column. See [gaps.md](gaps.md).

## Drill-down

A risk may name a `parent_pathway_id`: the pathway on another risk that it
expands. The source conversation's recommended next step is exactly this —
the RAG leakage pathway of the data-leak bow-tie becomes the top event of a
bow-tie of its own. In the whole-diagram read, each arm lists the risks that
drill into its pathway, and the threat and consequence summaries carry a
count.

The parent is fixed at creation, so the tree cannot be rewired into a loop,
and a pathway with child risks cannot be deleted. Only a pathway can be a
parent, which includes consequence pathways; whether that is useful is an
[open question](open-questions.md).

## The risk lifecycle

```
draft --> open --> accepted
  |        |  ^       |
  |        |  +-------+
  v        v          v
          retired (the end)
```

| Step | Rule |
|---|---|
| a risk is created | always as `draft`, whatever status the body names |
| `draft → open` | needs an owner |
| `open → accepted` | needs a residual assessment on record |
| `accepted → open` | allowed |
| any → `retired` | allowed; the end of the road |
| any write to a retired risk or anything under it | refused with a conflict |

An open or accepted risk cannot be updated to have no owner.
