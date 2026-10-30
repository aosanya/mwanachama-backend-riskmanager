# Gaps

A gap is something the register can see is missing or weak. Gaps are
**derived** — computed from the bow-tie on each call, never stored — so a new
piece of evidence closes one the moment it is recorded and there is no second
copy to correct.

`GET /gaps` (`riskmanager_gap_list`) returns them, most serious first.
`GET /risks/{riskID}/bowtie` is the same data as a diagram. Each gap is
`{kind, severity, risk_id, subject, subject_id, message}`: the subject is
`risk`, `pathway`, `barrier`, `control` or `action`, and the message names the
thing in words.

| Filter | Meaning |
|---|---|
| `risk_id` | Only this risk's gaps, whatever its status. An unknown risk is a 404. |
| `kind` | Only gaps of this kind. A kind this module does not report is a 400, not an empty list. |
| `severity` | Only gaps of this severity: `high`, `medium` or `low`. Any other value is a 400. |
| `offset` | Skip this many gaps of the ranked, filtered list. A negative offset counts as 0. |
| `limit` | Return at most this many. 200 when left out or not positive. |

With no `risk_id`, every risk that is not `retired` is reviewed, draft ones
included. A retired risk has no gaps: it is frozen.

## The kinds

Thirteen kinds, each with a severity.

| Kind | Severity | Subject | Fires when |
|---|---|---|---|
| `pathway_unguarded` | high | pathway | A threat or a consequence has no barrier standing on it. This is the source conversation's case: the control exists, and a major route is still open. |
| `barrier_failing` | high | barrier | The latest evidence has an outcome other than `pass`. |
| `action_overdue` | high | action | An open or in-progress action's due date has passed. |
| `risk_unowned` | medium | risk | The risk's owner is empty. A risk cannot open without one, so this is what a draft shows. |
| `risk_one_sided` | medium | risk | The risk has no threats, or has no consequences. Both missing gives two gaps. |
| `risk_unrated` | medium | risk | There is no inherent assessment, or no residual assessment. Both missing gives two gaps. A target rating is never required. |
| `control_unowned` | medium | control | A control standing in the bow-tie has no owner. Reported once per control per bow-tie. |
| `action_undated` | medium | action | An open or in-progress action has no due date. |
| `barrier_design` | medium | barrier | The barrier's `design` has been judged `partial` or `inadequate`. A design not yet judged is `barrier_unassessed`, not this. |
| `barrier_coverage` | medium | barrier | The barrier's `coverage` has been judged `partial` or `inadequate`. A coverage not yet judged is `barrier_unassessed`, not this. |
| `barrier_stale` | medium | barrier | The barrier has a `retest_days` window above zero and its latest evidence was recorded longer ago than that, whatever the outcome. A window of zero never goes stale, and a barrier with no evidence is `barrier_untested`, not this. A fresh piece of evidence clears it. |
| `barrier_unassessed` | low | barrier | The barrier's `design`, its `coverage` or both are still `unassessed`. One gap per barrier, and its message names which of the two. |
| `barrier_untested` | low | barrier | Its `maturity` is below `tested` — `proposed`, `designed` or `deployed`. The message gives the evidence count, so "deployed, with two configuration checks" reads differently from "proposed, with none". |

A `done` or `dropped` action never produces a gap. A dropped action says the
gap is accepted as it is, which is a decision and not a fix, so the gap it
was raised against is still reported on the next call.

A `barrier_untested` barrier with no evidence at all is not separately
flagged as unevidenced: a barrier cannot reach `tested` with none, so
`barrier_untested` already says it.

## Severity

| Severity | Kinds | What it says |
|---|---|---|
| `high` | `pathway_unguarded`, `barrier_failing`, `action_overdue` | A route is open, a barrier is known not to work, or a commitment has been missed. Someone should act now. |
| `medium` | `risk_unowned`, `risk_one_sided`, `risk_unrated`, `control_unowned`, `action_undated`, `barrier_design`, `barrier_coverage`, `barrier_stale` | The register is incomplete, or a barrier was judged short of adequate. A finding to schedule. |
| `low` | `barrier_unassessed`, `barrier_untested` | Nothing is known yet: a barrier placed and not yet judged or tried. Expected of a register still being drawn, so it comes last and is not an alarm. |

The mapping is `GapKind.Severity()` in `models/gap.go`. A kind the module
does not declare has no severity, which is also how an unknown `kind` filter
is recognised. `TestEveryGapKindHasTheSeverityTheModuleDeclaresForIt` reads
the constants out of that file, so a kind cannot be added without a severity
being chosen for it in both places.

## Order

Gaps come out ranked high, then medium, then low. Within one severity they
keep the order the register is walked in.

- **Inside one risk** (`Bowtie.Gaps`): the risk's own gaps first, then its
  threats and then its consequences with their barriers in position order,
  then its actions — and then the whole list ranked by severity, stably.
- **Across risks** (`Gaps`): the gaps of every risk reviewed are ranked
  together, so a high gap of the newest risk comes before a low gap of the
  oldest. Equal severities stay in the order the risks were created.

Ranking happens **before** filtering and paging, so a page is a window onto
the ranked list and not a ranking of a window.

## Paging

The filters apply in this order: `risk_id`, then `kind`, then `severity`,
then `offset`, then `limit`. A `kind` or `severity` filter therefore narrows
the list first and a page counts only what is left. An offset at or past the
end returns `[]`, never `null`. Walking `offset` upward in steps of `limit`
visits every gap once, with none repeated and none skipped, as long as the
register does not change between calls.

## A worked example

The data-leak bow-tie from the source conversation, built as
`data_leak_example_test.go` builds it. The consequences, the evidence and the
barrier judgements are **illustrative** (see
[sources.md](../1.%20requirements/sources.md)): only input DLP is judged and
tested, and one test of permission-aware retrieval has failed.

`GET /gaps?risk_id=<the parent risk>` returns 44 gaps — 4 high and 40 low:

| Rank | Severity | Kind | Message |
|---|---|---|---|
| 1 | high | `barrier_failing` | the latest evidence for "permission-aware retrieval" on the threat "RAG / retrieval leakage" was a fail |
| 2 | high | `pathway_unguarded` | nothing stands on the threat "Provider and integration leakage" |
| 3 | high | `pathway_unguarded` | nothing stands on the threat "Logging and observability leakage" |
| 4 | high | `pathway_unguarded` | nothing stands on the threat "Cross-tenant / session leakage" |
| 5–44 | low | `barrier_unassessed`, `barrier_untested` | one of each for each of the 20 barriers not yet judged or tried |

The four things that need attention are the first four rows. Before gaps were
ranked, the same register raised 64 in no useful order, because an unjudged
barrier raised a design gap, a coverage gap and an untested gap at once.

`GET /gaps` with no `risk_id` also reviews the sub-bow-tie drilled from the
RAG threat, which has six threats of its own and nothing standing on them. It
returns 53 gaps: 10 high (nine unguarded threats and the one failing barrier),
3 medium (the sub-bow-tie has no consequences and no ratings) and 40 low.
`?severity=high` returns the ten alone. For the parent risk on its own,
`?risk_id=<id>&limit=6` returns the first six rows above and
`?risk_id=<id>&limit=6&offset=6` the six after them.

## Standing is not a gap

A measure's `standing` — `proven`, `weak`, `unproven`, `failing` — is the
per-barrier answer used by the measure questions. A stale barrier is `unproven`
at best: a latest failure stays `failing`, and anything else drops to
`unproven`. Gaps are the per-register
answer. They draw on the same facts but are separate on purpose: the
standing ignores maturity, and the gaps include owners, ratings and work.

## Scale

`Gaps` reads the register in seven queries however many risks it holds: the
risks to review, then the pathways, barriers, controls, evidence, ratings and
actions of all of them at once, each scoped by a subquery on the risks rather
than a list of ids, so the number of parameters is fixed too. The bow-ties are
assembled in memory and ranked together. `limit` and `offset` still cut the
ranked list after every risk has been evaluated, because the ranking needs all
of them, so the work grows with the register but the number of round trips does
not. The drill-downs are not read, because no gap looks at them.

`query` on the measure questions is matched in SQL: the barriers, their
pathways, risks and controls are joined, the text is matched against the
pathway's title and scenario and the control's name and description, the
ranked order is applied and `limit` cuts it, so only the measures asked for
are loaded. The pathway questions did this already.

The tests that hold this are `gaps_scale_test.go` (the query count is the same
for 3 risks and 150, and one risk's gaps are the gaps its bow-tie reports
alone) and `measures_scale_test.go` (the rows loaded for one match out of 60).
