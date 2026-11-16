# Control name race

Hub issue MUN706. Two requests creating a control with the same name at the same moment left one control stored and answered the loser with `internal error`, not the 409 a sequential pair gets.

## Why it raced

`CreateControl` checked the name with `nameIsFree`, which reads every control name and compares them through `plaintext.SameName`, and then inserted the row. Two requests both read before either inserted, both passed the check, and the second insert met the unique index on `name`. The database error (`UNIQUE constraint failed`, code 2067 on SQLite, 23505 on Postgres) was returned as it came, so nothing mapped it to `ErrConflict` and the HTTP layer answered 500. `UpdateControl` had the same gap between its check and its update.

The index only compares the raw text. A composed name and its decomposed twin are different to the index and the same to `SameName`, so the race also stored both when the two requests passed the check together.

## The fix

| Part | What it covers |
|---|---|
| `controlNames` lock on the manager, held by `CreateControl` and `UpdateControl` across the free-name check and the write | Requests served by one process: exactly one passes the check, the others see the stored name and get the conflict, including a decomposed twin. |
| `nameTaken` in `control.go`, which turns a unique violation from the write into `ErrConflict` with the same text as the check, `a control called "X" already exists` | Another process on the same database, and an exact twin that slips past the lock: the index decides and the loser still gets the 409. |

The error text is the same on both paths, so a client cannot tell which one refused it.

## Tests

`control_concurrent_test.go` races eight callers over twelve rounds on a SQLite file in WAL mode and on SQLite shared memory, for creating one name, for creating a composed and a decomposed twin, and for renaming eight controls to one name. Each round must end with one winner, a conflict naming the control for every other caller, and one control stored. All three fail without the lock and the mapping.

## Hub copy

The hub copy in `projects/hub-steward/code/wakala-api/modules/riskmanager` needs the same change until ISS-105 deletes it. `scripts/hub-copy-diff.sh` shows the difference.

## Not covered

The barrier placement (`pathway_control` unique index) has the same check-then-insert shape and is not changed here.
