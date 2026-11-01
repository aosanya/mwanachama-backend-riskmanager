# mwanachama-backend-riskmanager (Go)

Open tasks only — 🚀 In Progress · 📋 Not Started · ⏸️ Blocked.
Everything else (completed rows, board context) is in
[todo_done.md](todo_done.md).

| Task | Title | Status | Notes |
|------|-------|--------|-------|
| RM3 | **Owners that are actors.** Resolve `risk.owner`, `control.owner` and `action.owner` against the actor module through a declared reference, so an unknown owner is a 400 and "everything this person owns" is a query. | 📋 | Open question #7. Needs the actor module's identity shape; a sibling-repo question. **Partly done 2026-10-30 (`fe61a3f`):** `NewRiskManager(db, s, WithOwners(identity.Owner, objects...))` makes an unresolvable owner on a risk, control or action a 400 (`ErrInvalidReference`), and `list_actions` takes `owner`. Left: the actor module implements no `identity.Owner`, so nothing real resolves; `spec.Field` has no field-level `refers`, so the owner fields are declared in `owners.go`, not in the blueprint. |
| RM8 | **A screen for the bow-tie.** The whole-diagram read and the five questions in the studio or the member app, with the standing of each measure shown on it. | 📋 | Belongs on that app's own board once RM1 is done; filed here so the module's side — what each screen needs from `get_bowtie` — is not forgotten. |
