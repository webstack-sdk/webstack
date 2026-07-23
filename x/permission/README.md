# Permission Module

The `x/permission` module provides generic, capability-style permission grants for Cosmos SDK chains. Any module can delegate its "who may do what, on which resource" checks here instead of maintaining its own grant state.

## Overview

- **Namespaces** — each consuming module owns one namespace, keyed by its module name. Namespaces are never created by transactions: they exist for exactly the modules registered with the permission keeper at app wiring, and state only carries each namespace's **owner** — the one address that can grant and revoke permissions within it.
- **Grants** — flat `(module, grantee, permission, scope)` keys, so a permission check is a single point-read.
- **Permissions** are strings (e.g. `issue`, `revoke`) registered in-process by the consuming module at wiring time — not an enum, so each module brings its own vocabulary.
- **Scopes** are opaque resource identifiers owned by the consuming module (e.g. a license type id). Modules that don't scope their permissions use the empty scope (module-wide grants).

Owners are set in genesis or by governance (`MsgUpdateNamespaceOwner`, an upsert that also serves as recovery); the current owner can also hand off directly (`MsgTransferOwnership`).

## Consuming the module

### 1. Register a namespace spec at wiring time

```go
app.PermissionKeeper.RegisterNamespace(licensetypes.ModuleName, permissiontypes.NamespaceSpec{
    Permissions: []string{"issue", "revoke"},
    // Optional: validate scope identifiers against module state. When nil,
    // scopes are unconstrained and may be empty (module-wide grants).
    ScopeExists: func(ctx context.Context, scope string) (bool, error) {
        _, found, err := app.LicenseKeeper.GetLicenseType(ctx, scope)
        return found, err
    },
})
```

Registration is static wiring — every node registers the same specs during app construction, so consulting them is deterministic. Grants for unregistered modules are rejected, at both msg handling and genesis import.

### 2. Check permissions from your keeper

```go
ok, err := permissionKeeper.Has(ctx, "license", issuer, "issue", licenseTypeID)
isOwner, err := permissionKeeper.IsOwner(ctx, "license", sender)
```

## Messages

| Message | Signer | Effect |
|---|---|---|
| `MsgUpdateNamespaceOwner` | authority (gov) | Set or rotate a registered module's namespace owner (upsert; also the recovery path) |
| `MsgTransferOwnership` | namespace owner | Hand the namespace to a new owner |
| `MsgGrantPermissions` | namespace owner | Union (permission, scope) pairs onto a grantee |
| `MsgRevokePermissions` | namespace owner | Remove specific (permission, scope) pairs (idempotent) |

Grants merge: existing pairs are never removed by `MsgGrantPermissions`. Per-message slice lengths are bounded by `MaxGrants` (100).

## Queries

| Query | Path |
|---|---|
| `Modules` (every registered module; owner empty if unset) | `/webstack/permission/modules` |
| `Module` (namespace owner + registered vocabulary) | `/webstack/permission/module/{module}` |
| `Grants` | `/webstack/permission/grants/{module}` |
| `GrantsByGrantee` | `/webstack/permission/grants/{module}/{grantee}` |
| `GrantsByScope` | `/webstack/permission/grants_by_scope/{module}/{scope}` |
| `HasPermission` | `/webstack/permission/has_permission/{module}/{grantee}/{permission}` |

All list queries are paginated. `GrantsByScope` is a filtered walk (scope is the last key component); prefer `GrantsByGrantee`/`HasPermission` for hot paths.

## CLI

```bash
# Grant: one entry per permission, each covering all listed scopes.
webstackd tx permission grant-permissions license webstack1abc... issue,revoke node.license,validator.license --from owner

# Module-wide grant (unscoped namespaces): "-" as the scopes argument.
webstackd tx permission grant-permissions mymod webstack1abc... operate - --from owner

# Revoke specific pairs (permission:scope; bare permission = module-wide grant).
webstackd tx permission revoke-permissions license webstack1abc... issue:node.license --from owner

# Hand off a namespace.
webstackd tx permission transfer-ownership license webstack1new... --from owner

# Queries
webstackd query permission modules
webstackd query permission module license
webstackd query permission grants license
webstackd query permission grants-by-grantee license webstack1abc...
webstackd query permission grants-by-scope license node.license --permission issue
webstackd query permission has-permission license webstack1abc... issue node.license
```

## Genesis

```json
{
  "namespaces": [
    { "module": "license", "owner": "webstack1..." }
  ],
  "grants": [
    { "module": "license", "grantee": "webstack1...", "permission": "issue", "scope": "node.license" }
  ]
}
```

Stateless validation checks shape and referential integrity (grants must reference a declared namespace, no duplicates). `InitGenesis` additionally enforces the registered specs: the module must be registered in the binary, the permission must be in its vocabulary, and the scope must pass its `ScopeExists` check. In `app.go` the permission module initializes **after** the modules whose state its scopes reference.

## State

| | Key | Value |
|---|---|---|
| Namespaces | `0x01 \| module` | `Namespace{module, owner}` |
| Grants | `0x02 \| module \| grantee \| permission \| scope` | keyset (no value) |

The key order `(module, grantee, permission, scope)` makes `Has` a point-read and `GrantsByGrantee` a prefix walk.
