# Hub copy diff

`mwanachama-backend-riskmanager` is the custodian of the risk manager module. The hub (`projects/hub-steward/code/wakala-api/modules/riskmanager`) holds a copy made on 2026-11-04 and changed it afterwards. Every change made in that copy exists here, so the two differ by the import path only (hub issue RM9, the riskmanager part of ISS-105).

## Command

```
scripts/hub-copy-diff.sh [path-to-hub-modules-riskmanager]
```

The default path is `../projects/hub-steward/code/wakala-api/modules/riskmanager`, relative to this repository's parent folder. The exit code is 1 when the output has any line starting with `diff `, `Binary files ` or `Only in hub`, and 0 otherwise, so `Only in backend` lines alone (this repository's own tests, `library/` and `spec/`) leave it at 0. The script and `scripts/hubdiff` come from `mwanachama-backend-git` (G78), with the module name changed; the exit rule is the fix of hub issue S70.

## What it does

1. Copies the `*.go` and `*.json` files of this repository and of the hub folder into two temporary folders, leaving out `cmd/`, `documentation/`, `scripts/` and `.git/`.
2. Rewrites the hub's imports: `wakalaserver/modules/shared/` becomes `github.com/aosanya/mwanachama-backend-shared/` and `wakalaserver/modules/riskmanager` becomes `github.com/aosanya/mwanachama-backend-riskmanager`.
3. Runs every Go file on both sides through `scripts/hubdiff`, which parses it, drops all comments and prints it with `gofmt`, so comments and layout do not show as differences.
4. Runs `diff -r backend hub` and prints its output.
5. Exits 1 if that output has a `diff ` line (a file differs, JSON files included), a `Binary files ` line or an `Only in hub` line (the hub has something this repository lacks); otherwise exits 0.

## Result of 2026-11-16

No file differs and nothing is `Only in hub`, so the script exits 0. What remains is `Only in backend` lines, all of them test files, test data or the example domains:

| Difference | Reason |
|---|---|
| The tests of the root package (`action_test.go` to `risk_test.go`, `seed_test.go`, `testdb_test.go`, `mount_test.go`, `examples_test.go`, `domain_agnostic_test.go` and the rest) | The backend suite is wider than the hub's, which carries `riskmanager_test.go` and the tests of its own commits. They test code that is identical in both. |
| `mcp/mcp_test.go`, `mcp/tool_conversation_meta_test.go`, `routes/routes_test.go`, `routes/gaps_query_test.go`, `routes/controls_failing_query_test.go` | The same, for the tool and route packages. |
| `postgres_integration_test.go` | The Postgres suite behind the `integration` build tag. The hub runs on the memory backend only. |
| `library/` | The shipped risk libraries the library tests read. The hub dropped its municipality libraries again (b27efe61). |
| `spec/` | The example domains the backend tests load. The hub builds its spec with `SpecForMount`. |

The seeded-domain helper of the backend root tests is `newAISafetyManager`, so the hub's `newManager(t, opts...)` in `riskmanager_test.go` can exist beside it.

## Result of 2026-11-16 after MUN706

The backend changed `control.go` and `manager.go` for hub issue MUN706 (simultaneous control creates and renames of one name answer the conflict; commits 4d63a10, 7091aaf, d8d7813). The hub copy has not received that change, so the live run prints `diff control.go hub/control.go` and `diff manager.go hub/manager.go` and exits 1. That is a real difference, not a script fault. A scratch hub copy given the same two files exits 0.

## The hub commits carried over

One commit here per hub commit or group, each naming the hub hash: f6cc1497 and de289377 (owner and add-a-risk tests), 83963ee8 (find by name, gaps by owner), 97ba9274, 34bf2ecb and 83754c63 (walk tests, one commit each), c556f11d (author tests), cd42c031 and 98579506 (control names fold), e09d6105 (500-row cap, offset paging), 5a2fbc4b (route coverage). The hub's municipality libraries and tests (08d3f0bc, 4347d412) were added and removed again by b27efe61 and net to nothing, so they are not carried.

`gaps_scale_test.go` reads the 520 gaps of 130 risks in pages of `MaxPage`, because one answer is capped at 500 rows since e09d6105. The operation counts in `mcp/mcp_test.go` and `routes/routes_test.go` are 33 since `match_names` (83963ee8).

## Result of 2026-11-19

The live run still prints the two MUN706 differences (`control.go`, `manager.go`) plus `models/control.go`, `provision.go` and `riskmanager.blueprint.json`, which carry the same change, and nothing is `Only in hub`. Every difference and its disposition is on [hub-import-drift.md](hub-import-drift.md).
