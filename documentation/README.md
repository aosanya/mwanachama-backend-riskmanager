# mwanachama-backend-riskmanager — documentation

## Layout

Four folders, in SDLC order, and everything lives under one of them.

| Folder | What's inside |
|--------|---------------|
| [1. requirements/](1.%20requirements/) | The problem, the scope, what is out of scope, and the source the module started from. |
| [2. design/](2.%20design/) | The bow-tie model, the five questions, what is declared and what stays in Go, the route table, the gaps, and the open questions. |
| [3. implementation/](3.%20implementation/) | The work: `todo.md` (open board), `todo_done.md` (completed rows + board context). |
| [4. qa/](4.%20qa/) | Test layout and results. |

## Boards and status

| File | What it holds |
| --- | --- |
| [todo.md](3.%20implementation/todo.md) | Open task board |
| [todo_done.md](3.%20implementation/todo_done.md) | Completed rows + board context |

## What this repo is

A standalone risk register built on the bow-tie: a centre event, its
threats and consequences, the measures standing on each route, and the
evidence that the measures work. No tie to Mwanachama's tier/member/campaign
model, and not mounted anywhere yet. See [CLAUDE.md](../CLAUDE.md) for the
invariants and [2. design/bowtie-model.md](2.%20design/bowtie-model.md) for
the model.
