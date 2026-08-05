# Licenses Module

The `x/license` module provides on-chain license management for Cosmos SDK chains. It allows a namespace owner to define license types and delegated addresses to issue and revoke licenses.

## Overview

- **License Types** define templates (e.g. `node.license`, `validator.license`) with optional max supply and a declarative transferrability flag
- **Licenses** are individual instances issued to holders with start/end dates and active/revoked status
- **Ownership and permissions** live in the [`x/permission`](../permission/README.md) module under the `license` namespace: per-type `issue`/`revoke` grants delegate rights over existing types, and a module-wide `type.create` grant delegates the creation of new ones. The owner grants these rights rather than holding them implicitly
- **An EVM precompile** exposes the same transactions and queries to Solidity contracts, routed through the same handlers
- **Downstream consumers** read licenses through a narrow keeper surface — [`x/network`](../network/README.md) derives its per-node-type activation limits from a holder's active license count

## Installation

### As a dependency in another Cosmos SDK chain

```bash
go get github.com/webstack-sdk/webstack
```

### Wiring into app.go (manual)

```go
import (
    license "github.com/webstack-sdk/webstack/x/license"
    licensekeeper "github.com/webstack-sdk/webstack/x/license/keeper"
    licensetypes "github.com/webstack-sdk/webstack/x/license/types"
)
```

1. Add the store key:

```go
keys := storetypes.NewKVStoreKeys(
    // ... existing keys
    licensetypes.StoreKey,
)
```

2. Create the keeper and register its permission namespace. The keeper consumes
   the permission keeper (ownership and grants) and the account keeper (holder
   accounts are created on issuance), so it is constructed after both:

```go
app.LicenseKeeper = licensekeeper.NewKeeper(
    appCodec,
    runtime.NewKVStoreService(keys[licensetypes.StoreKey]),
    logger,
    authAddr, // governance authority address
    app.PermissionKeeper,
    app.AccountKeeper,
)
license.RegisterNamespace(app.PermissionKeeper, app.LicenseKeeper)
```

3. Register the module:

```go
app.ModuleManager = module.NewManager(
    // ... existing modules
    license.NewAppModule(appCodec, app.LicenseKeeper),
)
```

4. Add to genesis ordering, **before** `permission` — grant scopes reference
   license type ids, so they can only be validated once license state is
   imported:

```go
genesisModuleOrder := []string{
    // ... existing modules
    licensetypes.ModuleName,
    permissiontypes.ModuleName,
}
```

### Wiring via depinject

The module supports dependency injection. Add the module proto config to your app config and import the package:

```go
import _ "github.com/webstack-sdk/webstack/x/license"
```

The `init()` function in `depinject.go` automatically registers the module. The
`ProvideModule` function resolves the codec, store service, permission keeper,
and account keeper from the DI container, and calls `RegisterNamespace` itself.

## Concepts

### Namespace Owner

The license module registers the `license` namespace with the `x/permission`
module at wiring time, declaring the `issue`/`revoke`/`type.create` vocabulary
and a scope validator that checks license type ids exist. The namespace owner
is the only address that can:
- Update license types
- Grant and revoke permissions (via `x/permission` messages)

Ownership confers no other rights. Creating license types is gated on the
`type.create` grant exactly as issuing is gated on `issue`, so a new chain's
first step after setting the namespace owner is for that owner to grant
`type.create` — to itself, to a separate admin address, or both.

The namespace itself exists by virtue of the registration — it is never
created by a transaction. Its owner is set in the permission module's genesis
or by governance (`MsgUpdateNamespaceOwner`), and can be handed off by the
current owner (`MsgTransferOwnership`).

### License Types

A license type is a template with:

| Field | Description |
|-------|-------------|
| `id` | Unique string identifier (e.g. `node.license`) |
| `transferrable` | Declares whether licenses of this type may change hands. This module has no transfer message, so nothing on chain reads it — it is metadata for consumers to enforce |
| `max_supply` | Maximum number of *outstanding* licenses, checked against `active_count`. `0` = unlimited |
| `issued_count` | Lifetime licenses issued. May exceed `max_supply`, since revoking frees a slot |
| `active_count` | Currently active licenses. This is what `max_supply` caps |
| `revoked_count` | Licenses revoked to date |

The address that created a type is **not recorded**. The `type.create` grant is
the whole authorization for creating one, and it confers no continuing
authority over the result — downstream modules gate on their own grants, not on
who created a license type. `x/network`, for instance, gates node type
registration on its `nodetype.create` grant and only checks that the license
type exists.

### Licenses

Each license is an instance of a license type:

| Field | Description |
|-------|-------------|
| `id` | Auto-incremented uint64, unique chain-wide across all license types |
| `type` | The license type ID this belongs to |
| `holder` | Bech32 address of the current holder |
| `start_date` | Start date in `YYYY-MM-DD` format |
| `end_date` | End date in `YYYY-MM-DD` format (empty = no expiry); keeps its issued value, revocation never modifies it |
| `status` | `LicenseStatus` enum: `active` or `revoked` |
| `revoked_date` | Block date of revocation in `YYYY-MM-DD` format; empty unless revoked |

Licenses are stored under their `id` alone and never deleted; revocation flips
`status` and stamps `revoked_date`, leaving the issued `end_date` intact.
Because ids are unique chain-wide, **a bare id is a complete handle** — the
single-license query takes only an id, and the type comes off the stored
record.

Two secondary indexes support the listing queries:

- `(type, id)` lists a type's licenses, active and revoked alike. A license's
  type never changes, so entries are written once at issuance and never moved
  or removed.
- `(holder, type, id)` tracks **active** licenses only — written on issue,
  removed on revoke. A license's holder never changes, so no entry is ever
  moved. This powers the holder queries and the revoke-most-recent-first walk.

The next-id sequence is a single chain-wide counter and is its own piece of
state (exported in genesis as `next_license_id`), independent of the
`issued_count` stats counter on the license type. Ids start at 1, so `0` is
never a valid license id. Ids are never reused: revoking frees a `max_supply`
slot but not an id, so a reissued license always gets a fresh number.

### Permissions

The owner delegates `issue`/`revoke` rights per license type through the
`x/permission` module:

```bash
webstackd tx permission grant-permissions license webstack1admin... issue,revoke node.license,validator.license --from owner
webstackd tx permission revoke-permissions license webstack1admin... issue:node.license --from owner
webstackd query permission grants-by-grantee license webstack1admin...
```

Each grant's scope must refer to an existing license type (enforced by the
registered scope validator). Wildcards are not supported — grants must
explicitly name each license type. The license keeper answers "may X issue Y?"
with a single point-read via `permissionKeeper.Has(ctx, "license", addr,
"issue", licenseTypeID)`.

`type.create` is the exception. It authorizes creating license types, so there
is no existing id to scope it to — the namespace declares it module-wide
(`Unscoped` on the registered `NamespaceSpec`), and it is granted with `-` in
place of a scope list:

```bash
webstackd tx permission grant-permissions license webstack1admin... type.create - --from owner
```

Because one invocation applies the same scopes to every permission it names,
`type.create` cannot be granted in the same command as `issue`/`revoke`.
Supplying a scope for it is rejected at grant time, so the grant only ever
exists under the empty scope and the keeper checks it as
`permissionKeeper.Has(ctx, "license", addr, "type.create", "")`.

## Messages

### MsgCreateLicenseType
Create a new license type. Signer must hold the module-wide `type.create`
permission; owning the namespace is not sufficient on its own.

```bash
# One-time, from the namespace owner:
webstackd tx permission grant-permissions license webstack1admin... type.create - --from owner

# max_supply is a flag, not a positional arg; it defaults to 0 (unlimited).
webstackd tx license create-license-type node.license true --max-supply 1000 --from admin
```

### MsgUpdateLicenseType
Update an existing license type's `transferrable` flag. `max_supply` is fixed
at creation and cannot be changed. The flag is declarative
— this module has no transfer message — so changing it signals intent to
consumers rather than altering any on-chain behaviour.

```bash
webstackd tx license update-license-type node.license true --from owner
```

### MsgIssueLicenses
Issue licenses in a single transaction. Each entry carries its own license
type, holder, dates, and count, so one message can issue to multiple holders
across multiple license types. Signer must have `issue` permission for every
referenced license type. Returned ids are flattened in entry order.

```bash
webstackd tx license issue-licenses \
  node.license:webstack1aaa...:1:2026-01-01:2027-01-01 \
  validator.license:webstack1bbb...:3:2026-01-01 \
  --from admin
```

Each entry is `license_type_id:holder:count:start_date[:end_date]`. A message
carries at most `MaxIssueBatchSize` (100) entries.

All entries are validated — permissions, addresses, dates, counts — and supply
caps are checked with the requested counts aggregated per license type, before
any license is issued, so a message that would breach a cap issues nothing.

Issuance **creates the holder's account** if it does not exist, so a wallet
holding only a license can sign its first transaction (e.g. the gasless
activation-key authorization in [`x/network`](../network/README.md)) without a
prior funding transfer.

### MsgRevokeLicenses
Revoke active licenses for a holder, most recently issued first. Sets status to `revoked` and records the current block date as `revoked_date`; the issued `end_date` is left unchanged. Signer must have `revoke` permission.

```bash
webstackd tx license revoke-licenses node.license webstack1abc... 2 --from admin
```

## Queries

All queries are available via gRPC, REST, and CLI (auto-generated via autocli).

| Query | Description | CLI |
|-------|-------------|-----|
| `LicenseType` | Single license type by ID | `webstackd q license license-type node.license` |
| `LicenseTypes` | All license types (paginated) | `webstackd q license license-types` |
| `License` | Single license by ID | `webstackd q license license 1` |
| `Licenses` | All licenses across all types (paginated) | `webstackd q license licenses` |
| `LicensesByType` | All licenses for a type (paginated) | `webstackd q license licenses-by-type node.license` |
| `LicensesByHolder` | Active licenses for a holder (paginated) | `webstackd q license licenses-by-holder webstack1...` |
| `LicensesByHolderAndType` | Active licenses by holder + type (paginated) | `webstackd q license licenses-by-holder-and-type webstack1... node.license` |

Permission grants are served by the `x/permission` queries (e.g.
`webstackd q permission grants license`).

### REST endpoints

All queries are available at `http://localhost:1317/webstack/license/...`:

```
GET /webstack/license/license_type/{id}
GET /webstack/license/license_types
GET /webstack/license/license/{id}
GET /webstack/license/licenses
GET /webstack/license/licenses_by_type/{type_id}
GET /webstack/license/licenses_by_holder/{holder}
GET /webstack/license/licenses_by_holder/{holder}/{type_id}
```

## EVM precompile

The module ships an EVM precompiled contract that exposes the same transactions
and queries to Solidity contracts. The interface is
[`LicenseI.sol`](precompile/LicenseI.sol); its ABI is embedded from
[`abi.json`](precompile/abi.json).

Default address (`licensetypes.PrecompileAddress`):

```
0x776562737461636B000000000000000000000001
   ^ ascii("webstack")                ^ per-precompile slot id
```

The ASCII prefix puts the address far above any plausible upstream
`cosmos/evm` precompile, so silent collision with a future upstream release is
effectively impossible. Operators may register it elsewhere; app wiring panics
at start-up if the EVM keeper's static precompile map already holds this
address.

```go
licensePrecompile := licenseprecompile.NewPrecompile(
    app.LicenseKeeper,
    evmAddrCodec,
    common.HexToAddress(licensetypes.PrecompileAddress),
)
// add to staticPrecompiles before constructing the EVM keeper
```

| Kind | Methods |
|---|---|
| Transactions | `createLicenseType`, `updateLicenseType`, `issueLicenses`, `revokeLicenses` |
| Queries (`view`) | `licenseType`, `licenseTypes`, `license`, `licenses`, `licensesByType`, `licensesByHolder`, `licensesByHolderAndType` |

Calls route through the same msg and query servers as the Cosmos path, so every
authorization rule above applies unchanged — the EVM caller address is
converted to bech32 and becomes the message signer. Ownership and permission
grants are **not** exposed through this precompile; manage them with the
`x/permission` messages.

Solidity events mirror the module's: `LicenseTypeCreated`, `LicenseTypeUpdated`,
`LicenseIssued`, `LicenseRevoked`.

## Consumed keeper surface

Other modules read license state through a narrow surface rather than the
records themselves:

```go
// Bounded count of a holder's active licenses across license types.
// stopAt != 0 stops the walk once the count is decisive; 0 counts everything.
count, err := k.CountActiveLicenses(ctx, holder, []string{"node.license"}, stopAt)

// Existence — the check x/network runs before binding a node type to a type.
found, err := k.HasLicenseType(ctx, "node.license")

// The full record, used as the x/permission scope validator.
lt, found, err := k.GetLicenseType(ctx, "node.license")
```

"Active" means "not revoked": this module never enforces `end_date`, so an
expired-but-unrevoked license still counts. License types meant for counting
should be issued with an empty `end_date` (revocation-only lifecycle).

## Genesis

Example genesis configuration:

```json
{
  "license": {
    "license_types": [
      {
        "id": "node.license",
        "transferrable": true,
        "max_supply": "100",
        "issued_count": "0",
        "active_count": "0",
        "revoked_count": "0"
      }
    ],
    "licenses": [],
    "next_license_id": "1"
  },
  "permission": {
    "namespaces": [
      { "module": "license", "owner": "webstack1owneraddress..." }
    ],
    "grants": [
      { "module": "license", "grantee": "webstack1adminaddress...", "permission": "type.create", "scope": "" },
      { "module": "license", "grantee": "webstack1adminaddress...", "permission": "issue", "scope": "node.license" }
    ]
  }
}
```

The permission module initializes after the license module, so genesis grants
can be validated against the license types declared above.

License types listed in genesis are written directly and need no `type.create`
grant — that permission gates the `MsgCreateLicenseType` handler, which genesis
import does not go through. The grant is what lets types be added once the
chain is running.

## Events

All state-changing operations emit events:

| Event | Attributes |
|-------|------------|
| `create_license_type` | `license_type_id` |
| `update_license_type` | `license_type_id` |
| `issue_licenses` | `license_type_id`, `holder`, `count` (one event per entry) |
| `revoke_licenses` | `license_type_id`, `holder`, `count` |

## State Storage

The module uses the `cosmossdk.io/collections` framework for type-safe state management:

| Collection | Key | Value |
|------------|-----|-------|
| `LicenseTypes` | `string` (type ID) | `LicenseType` |
| `Licenses` | `uint64` (license ID) | `License` |
| `NextLicenseID` | (item) | `uint64` (chain-wide next-id sequence, exported in genesis as `next_license_id`) |
| `LicensesByType` | `(string, uint64)` (type ID, license ID) | (keyset, no value; active and revoked) |
| `ActiveLicensesByHolder` | `(string, string, uint64)` (holder, type ID, license ID) | (keyset, no value; active licenses only) |

Permission grants are stored in the `x/permission` module under the `license`
namespace.

## Module Versioning

The module uses Cosmos SDK's consensus versioning. The current version is `1`. To add a state migration:

1. Bump `ConsensusVersion` in `module.go`
2. Create `keeper/migrator.go` with the migration function
3. Register the migration in `RegisterServices`
4. Add an upgrade handler in the app that calls `RunMigrations`

See the [Cosmos SDK migration docs](https://docs.cosmos.network/main/build/building-modules/upgrade) for details.

## Testing

```bash
go test ./x/license/...
```

Tests cover all message handlers, query handlers, genesis validation, and the
EVM precompile (ABI conformance, transaction and query methods, type
conversion).
