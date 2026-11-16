# Control name race

Hub issue MUN706. Two requests creating a control with the same name at the same moment left one control stored and answered the loser with `internal error`, not the 409 a sequential pair gets.

## Why it raced

`CreateControl` checked the name with `nameIsFree`, which reads every control name and compares them through `plaintext.SameName`, and then inserted the row. Two requests both read before either inserted, both passed the check, and the second insert met the unique index on `name`. The database error (`UNIQUE constraint failed`, code 2067 on SQLite, 23505 on Postgres) was returned as it came, so nothing mapped it to `ErrConflict` and the HTTP layer answered 500. `UpdateControl` had the same gap between its check and its update.

The index on `name` only compares the raw text. A composed name and its decomposed twin are different to that index and the same to `SameName`, so the lock covered one process only: two managers on one database file (hub issue RM10, found in the review of MUN706) both passed the check on their own side and both stored the twin. Exactly equal names were safe in every process, because the raw index refused the second one.

## The fix

| Part | What it covers |
|---|---|
| `controlNameKey` (`control_name_key.go`), `plaintext.FoldDrawnName` of the name, stored in the `name_key` column on create and on rename | One rule for every comparison: composed form, lower case, runs of spaces made one, trailing dots and spaces dropped. The check in `nameIsFree` and the database index use the same key. |
| Unique index `name_key` on `control`, declared in the blueprint and left out for rows whose key is empty | Every process on the database: the second writer of a name whose key is taken meets the index, whatever spelling it used. |
| `controlNames` lock on the manager, held by `CreateControl` and `UpdateControl` across the free-name check and the write | Requests served by one process: exactly one passes the check, the others see the stored name and get the conflict without touching the index. |
| `nameTaken` in `control.go`, which turns a unique violation from the write into `ErrConflict` with the same text as the check, `a control called "X" already exists` | Another process: the index decides, a unique violation from either the `name` index or the `name_key` index becomes the 409. |

The error text is the same on every path, so a client cannot tell which one refused it. `name_key` is written by the module and never read from a caller: the model carries it as `NameKey` with `json:"-"`, and the codec finds the column by field name.

## What the key changes

The register compared names with `SameName` (composed form, edge spaces). It now compares the folded key, so `Network firewall`, `network FIREWALL`, `Network   firewall` and ` Network firewall. ` are one control name. Renaming a control to its own name in another form is still allowed, because the check leaves the control itself out.

## What runs under the lock

Only the free-name check and the write run under `controlNames` (hub issue RM11). The owner lookup calls another module, which may be a database or a network round trip, and a plain mutex does not give up when the waiter's context ends, so a lookup held under the lock stalled every other create and rename past its own deadline. `CreateControl` always looked the owner up before taking the lock. `UpdateControl` now does the same: it reads the control, runs the field checks and the owner lookup for a changed owner, and only then takes the lock, reads the control again and checks the name and writes.

The lookup compares the new owner with the owner the control held at the first read. If another request changed the owner in between, the write still stores the owner the caller named, which was either the owner the control already held or one the register just knew. The caller now gets an unknown owner before a name conflict, where it used to get the conflict first.

`control_owner_lock_test.go` parks an owner lookup of one update, then creates another control with a 200 ms context and expects it to finish; on the old code it was still waiting two seconds later.

## Controls stored before the key existed

`Provision` runs `spec.Migrate`, which adds the empty column and the index, and then fills `name_key` of every control whose key is not the fold of its name, oldest first (`created_at`, then `id`), in one transaction. Where two stored controls fold to one key, which can only be a pair stored by the race above, the oldest takes the key and the others keep an empty key; nothing is deleted, renamed or merged, and a warning with the count is logged on each `Provision`. A control with an empty key is still seen by the in-process check, which scans the names, but the index cannot refuse a new twin of it from another process until the pair is resolved. Which of the pair to keep, rename or merge is the owner's choice (todo "Owner choice" on RM10). A host that calls `spec.Migrate` itself instead of `Provision` has the column and the index but no backfill.

## Tests

`control_concurrent_test.go` races eight callers over twelve rounds on a SQLite file in WAL mode and on SQLite shared memory, for creating one name, for creating a composed and a decomposed twin, and for renaming eight controls to one name. Each round must end with one winner, a conflict naming the control for every other caller, and one control stored. All three fail without the lock and the mapping.

`control_two_managers_test.go` uses two managers on two connections of one WAL file. It races a composed and a decomposed spelling over thirty-six rounds, and it holds the race still: a create hook on the first connection lets the second manager store the twin after the first one passed its check and before its write, for a create and for a rename. Each must answer the conflict and leave one control. `control_name_key_test.go` builds the old `controls` table by hand and checks the backfill, a second `Provision` changing nothing, and the folded comparison. The twin pair is spelled with escapes and a guard test checks that the two spellings differ in bytes and are one name.

`control_unique_violation_test.go` is what reaches `nameTaken` (hub issue RM12). Before it, the three tests of `control_concurrent_test.go` ran one manager, so the lock settled every race and the mapping was never reached: with its two calls removed they all still passed. The new tests use two managers on one WAL file and hold the race still with a create hook on the first connection, so the second manager stores the exact name after the first one's check and before its write, once for a create and once for a rename. Each must answer a conflict naming the control. A third test races one exact name over the two managers for thirty-six rounds, and a fourth races the twin pair from one manager on a file and on shared memory. With the two `nameTaken` calls replaced by the bare error, six of these tests fail, the losers answering the raw `UNIQUE constraint failed` text instead of the conflict.

## The MUN706 commits

The first red commit of MUN706 (`4d63a10`) spelled its twin test with `Hálózati tűzfal` and `Hálózáti tűzfal`, which differ in letters and are not twins, and the spelling was corrected inside the fix commit `7091aaf`, so that pair is not a tests-only red. The commits are left as they are. A reviewer ran the corrected test on the `4d63a10` code and it failed with two winners in seven of twelve rounds, so the claim held. Since RM12 the twin pair lives in one helper, `twinSpellings`, written with escapes, and `TestControlTwinSpellings_AreDifferentBytesAndOneName` fails when the two strings are equal or are not one name under `plaintext.SameName`.

## Hub copy

The hub copy in `projects/hub-steward/code/wakala-api/modules/riskmanager` needs the same change until ISS-105 deletes it. `scripts/hub-copy-diff.sh` shows the difference.

## Not covered

The barrier placement (`pathway_control` unique index) has the same check-then-insert shape and is not changed here.
