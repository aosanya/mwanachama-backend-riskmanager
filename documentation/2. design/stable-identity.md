# A thing's stable identity

`RiskManager` embeds the shared `identity.Owner` (shared S56), so a host that resolves a manager per instance already holds the owner for it.

`StableIdentity(ctx, object, id)` answers the raw `riskmanager/<instance>/<mount>/<role>/<id>` for a row. The object is the role (`risk`, `pathway`, `control`, `barrier`, `evidence`, `assessment`, `action`), not the domain's name for it, so a raw reads the same under every domain. The id is the row id: this module has no fork or draft that would copy a row under a new id, so the id is already stable. The mount is in the raw because two mounts of the module in one instance hold separate registers.

`Resolve(ctx, raw)` answers the `(role, id)` back:

- `identity.ErrInvalidRaw` for a string that does not parse;
- `identity.ErrNotOwned` for another module's raw, another instance's or another mount's;
- `identity.ErrNotFound` for a role the module does not hold, or a row that is not there;
- `ErrNoStableIdentity` from `StableIdentity` for a role the module does not hold, mapped to a 400.

Tests: `identity_owner_test.go`.
