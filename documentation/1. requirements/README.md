# Requirements

## Problem

An organisation that has decided a loss of control is worth guarding against
usually holds three separate things: a list of risks, a list of controls, and
a pile of audit findings. Nothing joins them, so nobody can answer the
plain questions: what are the ways this goes wrong, what happens if it does,
what stands in the way of each, and has anyone shown that the thing standing
there works. A policy listed against a risk is not a barrier until there is
evidence it stops, detects or limits the specific route, and a control that
works on one route may not reach the next.

`mwanachama-backend-digitaltwin` records what things measured and
`mwanachama-backend-assetmanager` records what is held; neither models a
failure and the safeguards against it.

## Vision

A standalone risk register in which each risk is drawn as a bow-tie, each
route on either side of it carries the measures standing on it, and each
measure carries the judgements and evidence that say how far it can be
trusted. The register answers the five plain questions directly —
threats, consequences, preventive measures against, mitigating measures
against, and what relies on a control — and reports its own gaps, so
assurance is a query and not a quarterly spreadsheet exercise.

It must work for an agent as well as a person: the same operations are
published as MCP tools, so an assistant can ask "what are the preventive
measures against this threat, and which of them are unproven?".

## Scope (session of 2026-10-26)

**In scope:**
- A risk as one bow-tie: a title, a precisely stated top event, a hazard, an
  owner, and a lifecycle (`draft → open → accepted`, `retired` at the end).
- Pathways on both sides of the top event — threats and consequences — with
  a concrete scenario each.
- Drill-down: a pathway of one risk may become the parent of another risk,
  so a branch of one bow-tie can be expanded into a bow-tie of its own.
- Controls, declared once and reusable across pathways and risks, each with
  an owner.
- Barriers: a control placed on a pathway, assessed for design, coverage and
  maturity (`proposed → designed → deployed → tested → monitored`).
- Evidence about a barrier — test, log, audit, configuration, incident — with
  an outcome of pass, partial or fail. Append-only.
- Assessments of a risk — inherent, residual, target — on a 1–5 likelihood
  and impact scale, giving a score and a band. Append-only.
- Actions that close a gap, with an owner, a due date and a status.
- The five plain questions, the whole-diagram read, and a derived list of
  gaps.
- Domain neutrality: two shipped domains, AI safety and food safety.

**Explicitly out of scope:**
- Any tie to Mwanachama's tier/member/campaign model — this is a standalone
  tool.
- A task manager. Actions here are lightweight commitments attached to a
  gap; `mwanachama-backend-taskmanager` owns work. Whether an action should
  point at a task is an open question.
- A policy or control library. The module stores the controls an
  organisation operates; it does not ship a catalogue of standard controls
  (an importable AI safety seed is a board row).
- An authorisation model. A mount supplies the `dispatch.Authorizer`.
- Resolving owners to people. `owner` is an opaque reference the mount
  fills.
- Quantitative risk analysis — probabilities as numbers, loss distributions,
  simulation. Ratings are the 1–5 judgement scale.
- Alerts or notifications when a gap opens.

**Not yet decided, flagged rather than assumed (see
[2. design/open-questions.md](../2.%20design/open-questions.md)):**
- Whether stale evidence should itself be a gap, and over what window.
- Whether a domain may supply its own likelihood and impact scales.
- Whether a barrier can be retired as distinct from removed.
- Whether a consequence can be drilled into as a threat can.
- How actions relate to taskmanager, and how owners relate to actor.
- How the module is mounted in wakala-api.
