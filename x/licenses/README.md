# Licenses Module

The `x/licenses` module provides on-chain license management for Cosmos SDK chains. It allows a module owner to define license types, delegate admin permissions, and issue/revoke/transfer licenses to addresses.

## Overview

- **License Types** define templates (e.g. `node.license`, `validator.license`) with optional max supply and transferrability
- **Licenses** are individual instances issued to holders with start/end dates and active/revoked status
- **Admin Keys** grant granular permissions (issue, revoke, update) per license type to delegated addresses
- **Module Owner** (set via params) controls license type creation and admin key management

## Installation

### As a dependency in another Cosmos SDK chain

```bash
go get github.com/webstack-sdk/webstack
```

### Wiring into app.go (manual)

```go
import (
    licenses "github.com/webstack-sdk/webstack/x/licenses"
    licenseskeeper "github.com/webstack-sdk/webstack/x/licenses/keeper"
    licensestypes "github.com/webstack-sdk/webstack/x/licenses/types"
)
```

1. Add the store key:

```go
keys := storetypes.NewKVStoreKeys(
    // ... existing keys
    licensestypes.StoreKey,
)
```

2. Create the keeper:

```go
app.LicensesKeeper = licenseskeeper.NewKeeper(
    appCodec,
    runtime.NewKVStoreService(keys[licensestypes.StoreKey]),
    logger,
    authAddr, // governance authority address
)
```

3. Register the module:

```go
app.ModuleManager = module.NewManager(
    // ... existing modules
    licenses.NewAppModule(appCodec, app.LicensesKeeper),
)
```

4. Add to genesis ordering:

```go
genesisModuleOrder := []string{
    // ... existing modules
    licensestypes.ModuleName,
}
```

### Wiring via depinject

The module supports dependency injection. Add the module proto config to your app config and import the package:

```go
import _ "github.com/webstack-sdk/webstack/x/licenses"
```

The `init()` function in `depinject.go` automatically registers the module. The `ProvideModule` function resolves `codec.Codec` and `store.KVStoreService` from the DI container.

## Concepts

### Module Owner

The module has a single `owner` address set in params. Only the owner can:
- Create and update license types
- Set and remove admin keys

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
| `end_date` | End date in `YYYY-MM-DD` format (empty = no expiry) |
| `status` | `active` or `revoked` |

### Admin Keys

Admin keys delegate permissions to addresses. Each admin key has grants:

```json
{
  "address": "cosmos1abc...",
  "grants": [
    { "permission": "issue", "license_types": ["node.license", "validator.license"] },
    { "permission": "revoke", "license_types": ["node.license"] }
  ]
}
```

Valid permissions: `issue`, `revoke`, `update`

A wildcard `"*"` in `license_types` grants the permission for all types.

## Messages

### MsgUpdateParams
Update module parameters (governance only).

### MsgCreateLicenseType
Create a new license type. Signer must be the module owner.

```bash
webstackd tx licenses create-license-type node.license true 1000 --from owner
```

### MsgUpdateLicenseType
Update an existing license type. Cannot set `max_supply` below `issued_count`.

```bash
webstackd tx licenses update-license-type node.license true 2000 --from owner
```

### MsgSetAdminKey
Grant permissions to an address. Signer must be the module owner.

```bash
webstackd tx licenses set-admin-key cosmos1admin... issue,revoke node.license,validator.license --from owner
```

### MsgRemoveAdminKey
Remove all grants for an address. Signer must be the module owner.

### MsgIssueLicense
Issue one or more licenses. Signer must have `issue` permission for the license type.

```bash
webstackd tx licenses issue-license node.license cosmos1holder... --from admin
```

The `count` field (default 1) allows issuing multiple licenses in one transaction.

### MsgRevokeLicense
Delete a license and decrement the issued count. Signer must have `revoke` permission.

```bash
webstackd tx licenses revoke-license node.license 1 --from admin
```

### MsgUpdateLicense
Change a license's status to `active` or `revoked`. Signer must have `update` permission.

```bash
webstackd tx licenses update-license node.license 1 revoked --from admin
```

### MsgTransferLicense
Transfer a license to a new holder. Signer must be the current holder and the license type must be transferrable.

```bash
webstackd tx licenses transfer-license node.license 1 cosmos1recipient... --from holder
```

### MsgBatchIssueLicense
Issue licenses to multiple holders in a single transaction. Signer must have `issue` permission.

```bash
webstackd tx licenses batch-issue-license node.license \
  cosmos1aaa...:2026-01-01:2027-01-01 \
  cosmos1bbb...:2026-01-01 \
  --from admin
```

Each entry is `holder:start_date[:end_date]`.

## Queries

All queries are available via gRPC, REST, and CLI (auto-generated via autocli).

| Query | Description | CLI |
|-------|-------------|-----|
| `Params` | Module parameters | `webstackd q licenses params` |
| `LicenseType` | Single license type by ID | `webstackd q licenses license-type node.license` |
| `LicenseTypes` | All license types (paginated) | `webstackd q licenses license-types` |
| `License` | Single license by type + ID | `webstackd q licenses license node.license 1` |
| `LicensesByType` | All licenses for a type | `webstackd q licenses licenses-by-type node.license` |
| `LicensesByHolder` | All licenses for a holder | `webstackd q licenses licenses-by-holder cosmos1...` |
| `LicensesByHolderAndType` | Licenses by holder + type | `webstackd q licenses licenses-by-holder-and-type cosmos1... node.license` |
| `AdminKey` | Grants for an address | `webstackd q licenses admin-key cosmos1...` |
| `AdminKeys` | All admin keys (paginated) | `webstackd q licenses admin-keys` |
| `AdminKeysByLicenseType` | Admins for a license type | `webstackd q licenses admin-keys-by-license-type node.license` |

### REST endpoints

All queries are available at `http://localhost:1317/licenses/...`:

```
GET /licenses/params
GET /licenses/license_type/{id}
GET /licenses/license_types
GET /licenses/license/{type_id}/{id}
GET /licenses/licenses_by_type/{type_id}
GET /licenses/licenses_by_holder/{holder}
GET /licenses/licenses_by_holder/{holder}/{type_id}
GET /licenses/admin_key/{address}
GET /licenses/admin_keys
GET /licenses/admin_keys_by_license_type/{license_type_id}
```

## Genesis

Example genesis configuration:

```json
{
  "licenses": {
    "params": {
      "owner": "cosmos1owneraddress..."
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
    "admin_keys": [
      {
        "address": "cosmos1adminaddress...",
        "grants": [
          {
            "permission": "issue",
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
| `set_admin_key` | `address`, `permissions`, `grant_license_types` |
| `remove_admin_key` | `address` |
| `issue_license` | `license_type_id`, `holder`, `count` |
| `batch_issue_license` | `license_type_id`, `count` |
| `revoke_license` | `license_type_id`, `license_id` |
| `update_license` | `license_type_id`, `license_id`, `status` |
| `transfer_license` | `license_type_id`, `license_id`, `holder`, `recipient` |
| `update_params` | `owner` |

## State Storage

The module uses the `cosmossdk.io/collections` framework for type-safe state management:

| Collection | Key | Value |
|------------|-----|-------|
| `Params` | (singleton) | `Params` |
| `LicenseTypes` | `string` (type ID) | `LicenseType` |
| `Licenses` | `(string, uint64)` (type ID, license ID) | `License` |
| `LicenseCounts` | `string` (type ID) | `uint64` |
| `AdminKeys` | `string` (address) | `AdminKey` |
| `LicenseByHolder` | `(string, string, uint64)` (holder, type ID, license ID) | `uint64` |

## Module Versioning

The module uses Cosmos SDK's consensus versioning. The current version is `1`. To add a state migration:

1. Bump `ConsensusVersion` in `module.go`
2. Create `keeper/migrator.go` with the migration function
3. Register the migration in `RegisterServices`
4. Add an upgrade handler in the app that calls `RunMigrations`

See the [Cosmos SDK migration docs](https://docs.cosmos.network/main/build/building-modules/upgrade) for details.

## Testing

```bash
go test ./x/licenses/...
```

Tests cover all message handlers, query handlers, and genesis validation.
