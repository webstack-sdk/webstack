package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/webstack-sdk/webstack/x/licenses/types"
)

type Keeper struct {
	cdc    codec.BinaryCodec
	logger log.Logger

	// state management
	Schema        collections.Schema
	Params        collections.Item[types.Params]
	LicenseTypes  collections.Map[string, types.LicenseType]
	Licenses      collections.Map[collections.Pair[string, uint64], types.License]
	LicenseCounts collections.Map[string, uint64]

	// Permissions is the flat set of (address, permission, license_type_id)
	// grant pairs; the permission component is the Permission enum value. The
	// grouped AddressPermissions view served by queries and genesis is reconstructed
	// from this keyset; see GetPermissionsByAddress / GetAllPermissions.
	Permissions collections.KeySet[collections.Triple[string, int32, string]]

	// ActiveLicensesByHolder indexes (holder, license_type_id, license_id) for
	// active licenses only: entries are added on issue, moved on transfer, and
	// removed on revoke. Revoked licenses remain in Licenses.
	ActiveLicensesByHolder collections.KeySet[collections.Triple[string, string, uint64]]

	authority string
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	logger log.Logger,
	authority string,
) Keeper {
	logger = logger.With(log.ModuleKey, "x/"+types.ModuleName)

	sb := collections.NewSchemaBuilder(storeService)

	if authority == "" {
		authority = authtypes.NewModuleAddress(govtypes.ModuleName).String()
	}

	k := Keeper{
		cdc:    cdc,
		logger: logger,

		Params:        collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		LicenseTypes:  collections.NewMap(sb, types.LicenseTypePrefix, "license_types", collections.StringKey, codec.CollValue[types.LicenseType](cdc)),
		Licenses:      collections.NewMap(sb, types.LicensePrefix, "licenses", collections.PairKeyCodec(collections.StringKey, collections.Uint64Key), codec.CollValue[types.License](cdc)),
		LicenseCounts: collections.NewMap(sb, types.LicenseCountPrefix, "license_counts", collections.StringKey, collections.Uint64Value),
		Permissions:   collections.NewKeySet(sb, types.PermissionPrefix, "permissions", collections.TripleKeyCodec(collections.StringKey, collections.Int32Key, collections.StringKey)),

		ActiveLicensesByHolder: collections.NewKeySet(sb, types.ActiveLicensesByHolderPrefix, "active_licenses_by_holder", collections.TripleKeyCodec(collections.StringKey, collections.StringKey, collections.Uint64Key)),

		authority: authority,
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}

	k.Schema = schema

	return k
}

func (k Keeper) Logger() log.Logger {
	return k.logger
}

func (k Keeper) GetAuthority() string {
	return k.authority
}

// GetParams returns the module params.
func (k Keeper) GetParams(ctx context.Context) types.Params {
	p, err := k.Params.Get(ctx)
	if err != nil {
		return types.DefaultParams()
	}
	return p
}

// SetParams sets the module params.
func (k Keeper) SetParams(ctx context.Context, p types.Params) error {
	return k.Params.Set(ctx, p)
}

// GetLicenseType returns a license type by id and whether it was found.
func (k Keeper) GetLicenseType(ctx context.Context, id string) (types.LicenseType, bool, error) {
	lt, err := k.LicenseTypes.Get(ctx, id)
	if err != nil {
		return types.LicenseType{}, false, nil
	}
	return lt, true, nil
}

// GetLicense returns a license by type and id and whether it was found.
func (k Keeper) GetLicense(ctx context.Context, typeID string, id uint64) (types.License, bool, error) {
	l, err := k.Licenses.Get(ctx, collections.Join(typeID, id))
	if err != nil {
		return types.License{}, false, nil
	}
	return l, true, nil
}

// HasPermission reports whether an address has a specific permission for a
// license type. It treats any underlying store error as "no permission" so
// callers that only need a yes/no answer (queries, tests) stay simple; callers
// that must distinguish "missing" from "store failure" should use the
// internal hasAdminPermission directly.
func (k Keeper) HasPermission(ctx context.Context, address string, permission types.Permission, licenseTypeID string) bool {
	ok, _ := k.hasAdminPermission(ctx, address, licenseTypeID, permission)
	return ok
}

// InitGenesis initializes the module's state from a genesis state. It runs the
// full GenesisState.Validate up front so direct keeper callers (tests, future
// migrations) get the same invariant enforcement as the JSON ValidateGenesis
// path on the AppModule.
func (k *Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) error {
	if err := data.Validate(); err != nil {
		return err
	}

	if err := k.Params.Set(ctx, data.Params); err != nil {
		return err
	}

	for _, lt := range data.LicenseTypes {
		if err := k.LicenseTypes.Set(ctx, lt.Id, lt); err != nil {
			return err
		}
	}

	// The id sequence is genesis state in its own right; it is never derived
	// from the stats counters.
	for _, lc := range data.LicenseCounts {
		if err := k.LicenseCounts.Set(ctx, lc.LicenseTypeId, lc.Count); err != nil {
			return err
		}
	}

	for _, license := range data.Licenses {
		if err := k.Licenses.Set(ctx, collections.Join(license.Type, license.Id), license); err != nil {
			return err
		}
		// The holder index tracks active licenses only.
		if license.Status == types.StatusActive {
			if err := k.ActiveLicensesByHolder.Set(ctx, collections.Join3(license.Holder, license.Type, license.Id)); err != nil {
				return err
			}
		}
	}

	for _, ak := range data.Permissions {
		for _, g := range ak.Grants {
			for _, lt := range g.LicenseTypes {
				if err := k.Permissions.Set(ctx, collections.Join3(ak.Address, int32(g.Permission), lt)); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// ExportGenesis exports the module's state to a genesis state.
func (k *Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params, err := k.Params.Get(ctx)
	if err != nil {
		panic(err)
	}

	var licenseTypes []types.LicenseType
	if err := k.LicenseTypes.Walk(ctx, nil, func(_ string, lt types.LicenseType) (bool, error) {
		licenseTypes = append(licenseTypes, lt)
		return false, nil
	}); err != nil {
		panic(err)
	}

	var licenses []types.License
	if err := k.Licenses.Walk(ctx, nil, func(_ collections.Pair[string, uint64], l types.License) (bool, error) {
		licenses = append(licenses, l)
		return false, nil
	}); err != nil {
		panic(err)
	}

	allPerms, err := k.GetAllPermissions(ctx)
	if err != nil {
		panic(err)
	}

	var licenseCounts []types.LicenseCount
	if err := k.LicenseCounts.Walk(ctx, nil, func(typeID string, count uint64) (bool, error) {
		licenseCounts = append(licenseCounts, types.LicenseCount{LicenseTypeId: typeID, Count: count})
		return false, nil
	}); err != nil {
		panic(err)
	}

	return &types.GenesisState{
		Params:        params,
		LicenseTypes:  licenseTypes,
		Licenses:      licenses,
		Permissions:   allPerms,
		LicenseCounts: licenseCounts,
	}
}

// nextLicenseID returns the next license ID for a given type and increments the counter.
func (k Keeper) nextLicenseID(ctx context.Context, typeID string) (uint64, error) {
	count, err := k.LicenseCounts.Get(ctx, typeID)
	if err != nil {
		count = 0
	}
	count++
	if err := k.LicenseCounts.Set(ctx, typeID, count); err != nil {
		return 0, err
	}
	return count, nil
}

// hasAdminPermission checks if an address has a specific permission for a
// license type. A missing grant returns (false, nil) so callers can treat it
// as a normal "not authorised" case; a store error is surfaced so the caller
// can fail the tx instead of silently denying the action.
func (k Keeper) hasAdminPermission(ctx context.Context, address string, licenseTypeID string, permission types.Permission) (bool, error) {
	return k.Permissions.Has(ctx, collections.Join3(address, int32(permission), licenseTypeID))
}

// appendGrantPair folds one (permission, license_type_id) pair into a grouped
// grants slice. Pairs must arrive in ascending (permission, license_type_id)
// order — which is exactly the Permissions key order — so the resulting
// grouped view is deterministic without any sorting.
func appendGrantPair(grants []types.PermissionGrant, permission types.Permission, licenseTypeID string) []types.PermissionGrant {
	if n := len(grants); n > 0 && grants[n-1].Permission == permission {
		grants[n-1].LicenseTypes = append(grants[n-1].LicenseTypes, licenseTypeID)
		return grants
	}
	return append(grants, types.PermissionGrant{Permission: permission, LicenseTypes: []string{licenseTypeID}})
}

// GetPermissionsByAddress reconstructs the grouped AddressPermissions view for an address from the
// flat Permissions keyset. Returns found=false when the address has no grants.
func (k Keeper) GetPermissionsByAddress(ctx context.Context, address string) (types.AddressPermissions, bool, error) {
	var grants []types.PermissionGrant
	rng := collections.NewPrefixedTripleRange[string, int32, string](address)
	err := k.Permissions.Walk(ctx, rng, func(key collections.Triple[string, int32, string]) (bool, error) {
		grants = appendGrantPair(grants, types.Permission(key.K2()), key.K3())
		return false, nil
	})
	if err != nil {
		return types.AddressPermissions{}, false, err
	}
	if len(grants) == 0 {
		return types.AddressPermissions{}, false, nil
	}
	return types.AddressPermissions{Address: address, Grants: grants}, true, nil
}

// GetAllPermissions reconstructs the grouped AddressPermissions view for every address with
// at least one grant, in ascending address order.
func (k Keeper) GetAllPermissions(ctx context.Context) ([]types.AddressPermissions, error) {
	var allPerms []types.AddressPermissions
	err := k.Permissions.Walk(ctx, nil, func(key collections.Triple[string, int32, string]) (bool, error) {
		addr := key.K1()
		if n := len(allPerms); n == 0 || allPerms[n-1].Address != addr {
			allPerms = append(allPerms, types.AddressPermissions{Address: addr})
		}
		ak := &allPerms[len(allPerms)-1]
		ak.Grants = appendGrantPair(ak.Grants, types.Permission(key.K2()), key.K3())
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return allPerms, nil
}

// isOwner checks if the sender is the module owner.
func (k Keeper) isOwner(ctx context.Context, sender string) (bool, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return false, err
	}
	return params.Owner == sender, nil
}
