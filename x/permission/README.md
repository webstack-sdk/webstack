# Permission Module

The `x/permission` module provides generic, capability-style permission grants for Cosmos SDK chains. Any module can delegate its "who may do what, on which resource" checks here instead of maintaining its own grant state.

## Overview

- **Namespaces** — each consuming module owns one namespace, keyed by its module name. Namespaces are never created by transactions: they exist for exactly the modules registered with the permission keeper at app wiring, and state only carries each namespace's **owner** — the one address that can grant and revoke permissions within it.
- **Grants** — flat `(module, grantee, permission, scope)` keys, so a permission check is a single point-read.
- **Permissions** are strings (e.g. `issue`, `revoke`) registered in-process by the consuming module at wiring time — not an enum, so each module brings its own vocabulary.
- **Scopes** are opaque resource identifiers owned by the consuming module (e.g. a license type id). Modules that don't scope their permissions use the empty scope (module-wide grants), and a module that does scope can still exempt individual permissions.

Owners are set in genesis or by governance (`MsgUpdateNamespaceOwner`, an upsert that also serves as recovery); the current owner can also hand off directly (`MsgTransferOwnership`).

## Consuming the module

### 0. Wire the keeper

The keeper takes no module dependencies, so it is constructed early — consuming
modules need it in hand to register their namespaces:

```go
app.PermissionKeeper = permissionkeeper.NewKeeper(
    appCodec,
    runtime.NewKVStoreService(keys[permissiontypes.StoreKey]),
    logger,
    authAddr, // governance authority address
)
```

Add `permission.NewAppModule(appCodec, app.PermissionKeeper)` to the module
manager, and place `permissiontypes.ModuleName` in the genesis ordering
**after** every module whose state its grant scopes reference.

Via depinject, `import _ "github.com/webstack-sdk/webstack/x/permission"` — the
`init()` in `depinject.go` registers the module and `ProvideModule` resolves the
codec and store service.

### 1. Register a namespace spec at wiring time

```go
app.PermissionKeeper.RegisterNamespace(licensetypes.ModuleName, permissiontypes.NamespaceSpec{
    Permissions: []string{"issue", "revoke", "type.create"},
    // Optional: validate scope identifiers against module state. When nil,
    // scopes are unconstrained and may be empty (module-wide grants).
    ScopeExists: func(ctx context.Context, scope string) (bool, error) {
        _, found, err := app.LicenseKeeper.GetLicenseType(ctx, scope)
        return found, err
    },
    // Optional: permissions exempt from ScopeExists, granted module-wide.
    Unscoped: []string{"type.create"},
})
```

Registration is static wiring — every node registers the same specs during app construction, so consulting them is deterministic. Grants for unregistered modules are rejected, at both msg handling and genesis import.

`Unscoped` exists for permissions with no resource to point at — the right to
create the very resources the other permissions are scoped to cannot name one,
since it does not exist yet. Grants for these must carry the **empty** scope; a
non-empty one is rejected even if it identifies a real resource, so each
`(grantee, permission)` has exactly one key form and a permission check cannot
miss a grant filed under a scope it did not think to look up.

Malformed specs are wiring bugs and panic at startup: `Unscoped` without a
`ScopeExists`, a name outside the vocabulary, duplicates, or every permission
listed (drop `ScopeExists` instead).

### 2. Check permissions from your keeper

```go
ok, err := permissionKeeper.Has(ctx, "license", issuer, "issue", licenseTypeID)
isOwner, err := permissionKeeper.IsOwner(ctx, "license", sender)
```

`Has` returns `(false, nil)` for a missing grant but surfaces store errors, so a
read failure fails the tx instead of silently denying. `HasPermission` is the
yes/no convenience form that flattens errors to "no"; prefer `Has` in handlers.
`IsOwner` returns `ErrNamespaceNotFound` when no owner is set, so callers can
tell "not the owner" from "no owner configured".

Module and permission names must be non-empty lowercase alphanumerics plus `.`,
`_`, and `-`. The character set deliberately excludes the `,` and `:` delimiters
used at the CLI boundary. Scopes are opaque to this module and carry no such
restriction.

## Messages

| Message | Signer | Effect |
|---|---|---|
| `MsgUpdateNamespaceOwner` | authority (gov) | Set or rotate a registered module's namespace owner (upsert; also the recovery path) |
| `MsgTransferOwnership` | namespace owner | Hand the namespace to a new owner |
| `MsgGrantPermissions` | namespace owner | Union (permission, scope) pairs onto a grantee |
| `MsgRevokePermissions` | namespace owner | Remove specific (permission, scope) pairs (idempotent) |

Grants merge: existing pairs are never removed by `MsgGrantPermissions`, and
re-granting an existing pair is an idempotent overwrite. Every pair in a message
is resolved and validated before anything is written, so a partially-invalid
message grants nothing.

`MaxGrants` (100) bounds both the top-level `grants`/`permissions` list and the
inner `scopes` list of each grant entry.

## Queries

| Query | Path |
|---|---|
| `Modules` (every registered module; owner empty if unset) | `/webstack/permission/modules` |
| `Module` (namespace owner + registered vocabulary) | `/webstack/permission/module/{module}` |
| `Grants` | `/webstack/permission/grants/{module}` |
| `GrantsByGrantee` | `/webstack/permission/grants/{module}/{grantee}` |
| `GrantsByScope` | `/webstack/permission/grants_by_scope/{module}/{scope}` |
| `HasPermission` | `/webstack/permission/has_permission/{module}/{grantee}/{permission}` |

The grant queries are paginated. `Modules` is not: the registered-module set is
bounded by the modules compiled into the binary. `GrantsByScope` is a filtered
walk (scope is the last key component); prefer `GrantsByGrantee`/`HasPermission`
for hot paths.

## CLI

```bash
# Grant: one entry per permission, each covering all listed scopes.
webstackd tx permission grant-permissions license webstack1abc... issue,revoke node.license,validator.license --from owner

# Module-wide grant: "-" as the scopes argument. Used both for namespaces that
# don't scope at all and for individual permissions declared Unscoped. Since
# every permission in one invocation shares the same scopes, a mix of scoped
# and module-wide permissions takes two commands.
webstackd tx permission grant-permissions mymod webstack1abc... operate - --from owner
webstackd tx permission grant-permissions license webstack1abc... type.create - --from owner

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

## Events

| Event | Attributes |
|---|---|
| `update_namespace_owner` | `module`, `owner` |
| `transfer_ownership` | `module`, `owner` (the new owner) |
| `grant_permissions` | `module`, `grantee`, `permissions` (comma-joined), `scopes` (per-permission scope lists, comma-joined within an entry and semicolon-joined between entries) |
| `revoke_permissions` | `module`, `grantee`, `permissions` (comma-joined), `scopes` (comma-joined, positionally paired with `permissions`) |

## State

| | Key | Value |
|---|---|---|
| Namespaces | `0x01 \| module` | `Namespace{module, owner}` |
| Grants | `0x02 \| module \| grantee \| permission \| scope` | keyset (no value) |

The key order `(module, grantee, permission, scope)` makes `Has` a point-read and `GrantsByGrantee` a prefix walk.

The registered namespace specs are **not** state — they live in an in-process
map built at wiring time. State carries only namespace owners and grants.

## Consumers in this repo

| Module | Vocabulary | Scoping |
|---|---|---|
| [`x/license`](../license/README.md) | `issue`, `revoke`, `type.create` | Scoped to license type ids, except `type.create` (declared `Unscoped`) |
| [`x/network`](../network/README.md) | `wallet.create`, `nodetype.create` | Module-wide throughout (no `ScopeExists`) |

## Module Versioning

The module uses Cosmos SDK's consensus versioning. The current version is `1`.

## Testing

```bash
go test ./x/permission/...
```

Tests cover the namespace registry and spec validation, all message handlers,
query handlers, and genesis round-tripping.
