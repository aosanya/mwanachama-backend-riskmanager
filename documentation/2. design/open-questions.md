# Open questions

Flagged rather than assumed. Each one is a decision the module has not made;
none is hidden behind a default that looks like an answer.

## 1. Should stale evidence be a gap?

A passing test from three years ago still makes a barrier `proven`, because
the standing uses only the latest outcome. A barrier relied on for a year
with no retest is exposed, and the register cannot say so.

The shape is a retest window — a number of days after which the latest
evidence no longer counts. The unsettled part is whose window it is: one for
the mount, one per control, one per barrier, or one per risk band, with
critical risks retested more often. A per-barrier column is the simplest and
the easiest to leave unset; a window in the mount's settings needs the
settings mechanism `spec` already has.

Built as the smallest option, pending the owner: a per-barrier `retest_days`,
zero meaning no window. A mount-wide default or a per-control window is still
open; the barrier column is the one every other choice would sit on top of.

## 2. Can a domain supply its own scales?

Likelihood and impact are 1 to 5, and the bands are fixed at low 1–4, medium
5–9, high 10–15, critical 16–25. A domain with a three-point scale, or a
different matrix, cannot change either, because a domain spec may set a
default and nothing else. Either the module accepts a scale in its settings,
or a different rating shape is a different module.

## 3. Retired barrier or removed barrier?

A barrier with evidence or actions cannot be removed, because the record is
part of the history. So a measure that was withdrawn — a control switched
off — stays standing on its pathway forever, still counted, still judged. A
`retired` state on a barrier would let it leave the live diagram and keep
its history. Alternatives: a barrier whose maturity can regress to a
`proposed` that means withdrawn, which overloads a word, or leaving the
barrier and recording failing evidence, which is what happens today.

## 4. Can a consequence be drilled into?

`parent_pathway_id` takes any pathway, so a consequence can be the parent of
a risk. The source conversation only drilled a threat. A consequence
expanded as a bow-tie of its own — "regulatory action" as the top event of a
bow-tie about what follows it — is a recognised extension of the method and
would chain consequences into consequences. Nothing stops it and nothing
rolls it up. Whether the whole-diagram read should follow a chain, and how
far, is unsettled.

## 5. How do actions relate to taskmanager?

An action is a commitment with an owner, a due date and a status.
`mwanachama-backend-taskmanager` already has work items, assignment and
dependencies. Running both means a gap's remediation is tracked twice or not
in the system people actually use. The options: an optional task reference on
an action; actions that are tasks and a risk that only points at them; or
leaving the module's actions as the light, self-contained thing it is. Each
changes what `action` is, so this should be decided before a screen is built
on it.

Built as the smallest of the three, pending the owner: an optional `task_ref`
on the action, the task module's own stable reference to the task that tracks
it, with an `ActionFilter.TaskRef` to find the actions a task tracks. It is
opaque unless the mount passes `WithTasks(directory)`, any shared
`identity.Owner`, in which case a reference that does not resolve is refused as
`ErrInvalidReference`. The task module implements no `identity.Owner` today, so
nothing resolves yet. The action stays a self-contained commitment with its own
owner, date and status; nothing is copied to or from the task, so a done task
does not close the action. Actions that are tasks, or no link at all, remain
open for the owner, and the column can be dropped without losing any other
data.

## 6. How is the module mounted in wakala-api?

Sibling modules that wakala-api mounts carry a `<module>.wakala.json` naming
the objects the mount provisions, and wakala-api's `internal/specs/` holds
the deployment specs. This module has neither, and mounting it is a change
to a sibling repo, which needs the owner's go-ahead. The gateway the older
modules joined is retired; wakala-api provisions, so no SQL mirror is
needed.

## 7. Owners are opaque strings

`risk.owner`, `control.owner` and `action.owner` hold whatever the mount puts
there. Nothing checks that the owner exists, is still a member, or may own a
risk. Resolving them to actors, as a declared reference to the actor
module's object, would give "everything this person owns" and "which owners
have left", and would turn a typo from a silent gap into a refusal. It would
also tie this module to the actor module's shape, which it does not know today.

Built without the actor module's shape. `NewRiskManager(db, s, WithOwners(owners, objects...))`
takes any shared `identity.Owner` as the directory of who may own. When it is
set, a non-empty `risk.owner`, `control.owner` or `action.owner` has to
`Resolve` through it: a string that is not a raw, another module's raw or a
raw with no row is refused as `ErrInvalidReference` (a 400), and the optional
`objects` list restricts which resolved objects may own (for example only
`actor`). An owner that has not changed on an update is not checked again, so
a row kept before the directory was set can still be edited. With no
directory the owner stays the opaque string it was. An empty owner is still a
gap and not a refusal. "Everything this person owns" is the `owner` filter on
`list_risks`, `list_controls` and `list_actions`.

What is still open, and belongs to the owner and the actor repo: the actor
module implements no `identity.Owner` today, so the raw an actor would have,
and whether a group may own, are undecided; and the shared engine has no
field-level reference (`spec.Field` has no `refers`; only an operation
argument can be marked), so the three owner fields are declared as references
in this module's code, in `owners.go`, and not in the blueprint.

## 8. Is a gap an event?

Gaps are computed when asked. Nothing is told when one opens — when evidence
fails, when an action goes overdue, when a risk loses its owner. A
notification would use `mwanachama-backend-comm`, which this module does not
import. Out of scope for the first build; the question is whether it is
ever this module's job, or a routine that calls `list_gaps` and tells
someone.
