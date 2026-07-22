# Licenses Module

The `x/license` module provides on-chain license management for Cosmos SDK chains. It allows a module owner to define license types, delegate permissions, and issue/revoke/transfer licenses to addresses.

## Overview

- **License Types** define templates (e.g. `node.license`, `validator.license`) with optional max supply and transferrability
- **Licenses** are individual instances issued to holders with start/end dates and active/revoked status
- **Permissions** grant granular rights (issue, revoke) per license type to delegated addresses
- **Module Owner** (set via params) controls license type creation and permission management

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

2. Create the keeper:

```go
app.LicenseKeeper = licensekeeper.NewKeeper(
    appCodec,
    runtime.NewKVStoreService(keys[licensetypes.StoreKey]),
    logger,
    authAddr, // governance authority address
)
```

3. Register the module:

```go
app.ModuleManager = module.NewManager(
    // ... existing modules
    license.NewAppModule(appCodec, app.LicenseKeeper),
)
```

4. Add to genesis ordering:

```go
genesisModuleOrder := []string{
    // ... existing modules
    licensetypes.ModuleName,
}
```

### Wiring via depinject

The module supports dependency injection. Add the module proto config to your app config and import the package:

```go
import _ "github.com/webstack-sdk/webstack/x/license"
```

The `init()` function in `depinject.go` automatically registers the module. The `ProvideModule` function resolves `codec.Codec` and `store.KVStoreService` from the DI container.

## Concepts

### Module Owner

The module has a single `owner` address set in params. Only the owner can:
- Create and update license types
- Grant and revoke permissions

The owner is initially set via genesis or governance (`MsgUpdateParams`).

### License Types

A license type is a template with:

| Field | Description |
|-------|-------------|
| `id` | Unique string identifier (e.g. `node.license`) |
| `transferrable` | Whether licenses of this type can be transferred between holders |
| `max_supply` | Maximum number of active licenses. `0` = unlimited |
| `issued_count` | Current number of active licenses (managed automatically) |

### Licenses

Each license is an instance of a license type:

| Field | Description |
|-------|-------------|
| `id` | Auto-incremented uint64, unique within the license type |
| `type` | The license type ID this belongs to |
| `holder` | Bech32 address of the current holder |
| `start_date` | Start date in `YYYY-MM-DD` format |
| `end_date` | End date in `YYYY-MM-DD` format (empty = no expiry); keeps its issued value, revocation never modifies it |
| `status` | `LicenseStatus` enum: `active` or `revoked` |
| `revoked_date` | Block date of revocation in `YYYY-MM-DD` format; empty unless revoked |

Licenses are stored under `(type, id)` and never deleted; revocation flips
`status` and stamps `revoked_date`, leaving the issued `end_date` intact. A
secondary index keyed `(holder, type, id)` tracks **active** licenses only —
it is written on issue, moved on transfer, and removed on revoke — which is
what powers the holder queries and the revoke-most-recent-first walk.

The per-type next-id sequence is its own piece of state (exported in genesis
as `license_counts`), independent of the `issued_count` stats counter on the
license type.

### Permissions

The owner delegates permissions to addresses. Each address's permissions are a set of grants:

```json
{
  "address": "webstack1abc...",
  "grants": [
    { "permission": "PERMISSION_ISSUE", "license_types": ["node.license", "validator.license"] },
    { "permission": "PERMISSION_REVOKE", "license_types": ["node.license"] }
  ]
}
```

Valid permissions: `issue`, `revoke` (the `Permission` proto enum; the CLI and
EVM precompile accept and return these lowercase forms, while gRPC/genesis
JSON uses the enum names `PERMISSION_ISSUE` / `PERMISSION_REVOKE`).

Each license type in a grant must refer to an existing license type. Wildcards are not supported — grants must explicitly specify each license type.

On disk, grants are stored as a flat set of
`(address, permission, license_type_id)` keys, so permission checks are a
single point-read. The grouped shape above is the genesis and query API view,
reconstructed on demand.

## Messages

### MsgUpdateParams
Update module parameters (governance only).

### MsgCreateLicenseType
Create a new license type. Signer must be the module owner.

```bash
webstackd tx license create-license-type node.license true 1000 --from owner
```

### MsgUpdateLicenseType
Update an existing license type. Cannot set `max_supply` below `issued_count`.

```bash
webstackd tx license update-license-type node.license true 2000 --from owner
```

### MsgGrantPermissions
Grant permissions to an address. Signer must be the module owner. Grants are
**merged** with any existing grants for the address: the (permission, license
type) pairs in the message are added to whatever is already stored, with
duplicates deduped. Existing grants are never removed by `MsgGrantPermissions`;
use `MsgRevokePermissions` to remove specific pairs.

```bash
webstackd tx license grant-permissions webstack1admin... issue,revoke node.license,validator.license --from owner
```

### MsgRevokePermissions
Remove specific `(license_type_id, permission)` pairs from an address's permissions.
Signer must be the module owner.

Pairs that aren't currently granted are silently ignored. A grant whose
license-type list becomes empty is dropped, and if no grants remain the
address's permissions entry disappears.

```bash
webstackd tx license revoke-permissions webstack1admin... \
  node.license:issue validator.license:revoke --from owner
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

Each entry is `license_type_id:holder:count:start_date[:end_date]`.

### MsgRevokeLicenses
Revoke active licenses for a holder, most recently issued first. Sets status to `revoked` and records the current block date as `revoked_date`; the issued `end_date` is left unchanged. Signer must have `revoke` permission.

```bash
webstackd tx license revoke-licenses node.license webstack1abc... 2 --from admin
```

### MsgTransferLicense
Transfer a license to a new holder. Signer must be the current holder and the license type must be transferrable.

```bash
webstackd tx license transfer-license node.license 1 webstack1recipient... --from holder
```

## Queries

All queries are available via gRPC, REST, and CLI (auto-generated via autocli).

| Query | Description | CLI |
|-------|-------------|-----|
| `Params` | Module parameters | `webstackd q license params` |
| `LicenseType` | Single license type by ID | `webstackd q license license-type node.license` |
| `LicenseTypes` | All license types (paginated) | `webstackd q license license-types` |
| `License` | Single license by type + ID | `webstackd q license license node.license 1` |
| `Licenses` | All licenses across all types (paginated) | `webstackd q license licenses` |
| `LicensesByType` | All licenses for a type (paginated) | `webstackd q license licenses-by-type node.license` |
| `LicensesByHolder` | Active licenses for a holder (paginated) | `webstackd q license licenses-by-holder webstack1...` |
| `LicensesByHolderAndType` | Active licenses by holder + type (paginated) | `webstackd q license licenses-by-holder-and-type webstack1... node.license` |
| `PermissionsByAddress` | Grants for an address | `webstackd q license permissions-by-address webstack1...` |
| `Permissions` | Grants of every address (paginated) | `webstackd q license permissions` |
| `PermissionsByLicenseType` | Addresses with grants for a license type | `webstackd q license permissions-by-license-type node.license` |

### REST endpoints

All queries are available at `http://localhost:1317/webstack/license/...`:

```
GET /webstack/license/params
GET /webstack/license/license_type/{id}
GET /webstack/license/license_types
GET /webstack/license/license/{type_id}/{id}
GET /webstack/license/licenses
GET /webstack/license/licenses_by_type/{type_id}
GET /webstack/license/licenses_by_holder/{holder}
GET /webstack/license/licenses_by_holder/{holder}/{type_id}
GET /webstack/license/permissions_by_address/{address}
GET /webstack/license/permissions
GET /webstack/license/permissions_by_license_type/{license_type_id}
```

## Genesis

Example genesis configuration:

```json
{
  "license": {
    "params": {
      "owner": "webstack1owneraddress..."
    },
    "license_types": [
      {
        "id": "node.license",
        "transferrable": true,
        "max_supply": "100",
        "issued_count": "0"
      }
    ],
    "licenses": [],
    "license_counts": [],
    "permissions": [
      {
        "address": "webstack1adminaddress...",
        "grants": [
          {
            "permission": "PERMISSION_ISSUE",
            "license_types": ["node.license"]
          }
        ]
      }
    ]
  }
}
```

## Events

All state-changing operations emit events:

| Event | Attributes |
|-------|------------|
| `create_license_type` | `license_type_id` |
| `update_license_type` | `license_type_id` |
| `grant_permissions` | `address`, `permissions`, `grant_license_types` |
| `revoke_permissions` | `address`, `permissions`, `grant_license_types` |
| `issue_licenses` | `license_type_id`, `holder`, `count` (one event per entry) |
| `revoke_licenses` | `license_type_id`, `holder`, `count` |
| `transfer_license` | `license_type_id`, `license_id`, `holder`, `recipient` |
| `update_params` | `owner` |

## State Storage

The module uses the `cosmossdk.io/collections` framework for type-safe state management:

| Collection | Key | Value |
|------------|-----|-------|
| `Params` | (singleton) | `Params` |
| `LicenseTypes` | `string` (type ID) | `LicenseType` |
| `Licenses` | `(string, uint64)` (type ID, license ID) | `License` |
| `LicenseCounts` | `string` (type ID) | `uint64` (next-id sequence, exported in genesis as `license_counts`) |
| `Permissions` | `(string, int32, string)` (address, permission, type ID) | (keyset, no value) |
| `ActiveLicensesByHolder` | `(string, string, uint64)` (holder, type ID, license ID) | (keyset, no value; active licenses only) |

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

Tests cover all message handlers, query handlers, and genesis validation.
