package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/webstack-sdk/webstack/x/license/types"
)

type Keeper struct {
	cdc    codec.BinaryCodec
	logger log.Logger

	// state management
	Schema        collections.Schema
	LicenseTypes  collections.Map[string, types.LicenseType]
	Licenses      collections.Map[collections.Pair[string, uint64], types.License]
	LicenseCounts collections.Map[string, uint64]

	// ActiveLicensesByHolder indexes (holder, license_type_id, license_id) for
	// active licenses only: entries are added on issue, moved on transfer, and
	// removed on revoke. Revoked licenses remain in Licenses.
	ActiveLicensesByHolder collections.KeySet[collections.Triple[string, string, uint64]]

	// permissionKeeper holds ownership and (permission, license type) grants
	// for the license module under the "license" namespace.
	permissionKeeper types.PermissionKeeper

	authority string
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	logger log.Logger,
	authority string,
	permissionKeeper types.PermissionKeeper,
) Keeper {
	logger = logger.With(log.ModuleKey, "x/"+types.ModuleName)

	sb := collections.NewSchemaBuilder(storeService)

	if authority == "" {
		authority = authtypes.NewModuleAddress(govtypes.ModuleName).String()
	}

	k := Keeper{
		cdc:    cdc,
		logger: logger,

		LicenseTypes:  collections.NewMap(sb, types.LicenseTypePrefix, "license_types", collections.StringKey, codec.CollValue[types.LicenseType](cdc)),
		Licenses:      collections.NewMap(sb, types.LicensePrefix, "licenses", collections.PairKeyCodec(collections.StringKey, collections.Uint64Key), codec.CollValue[types.License](cdc)),
		LicenseCounts: collections.NewMap(sb, types.LicenseCountPrefix, "license_counts", collections.StringKey, collections.Uint64Value),

		ActiveLicensesByHolder: collections.NewKeySet(sb, types.ActiveLicensesByHolderPrefix, "active_licenses_by_holder", collections.TripleKeyCodec(collections.StringKey, collections.StringKey, collections.Uint64Key)),

		permissionKeeper: permissionKeeper,

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

// hasPermission checks whether an address holds a permission ("issue",
// "revoke") for a license type, via the x/permission module's "license"
// namespace. A missing grant returns (false, nil); a store error is surfaced
// so the caller can fail the tx instead of silently denying the action.
func (k Keeper) hasPermission(ctx context.Context, address, permission, licenseTypeID string) (bool, error) {
	return k.permissionKeeper.Has(ctx, types.ModuleName, address, permission, licenseTypeID)
}

// isOwner checks whether the sender owns the license namespace in the
// x/permission module.
func (k Keeper) isOwner(ctx context.Context, sender string) (bool, error) {
	return k.permissionKeeper.IsOwner(ctx, types.ModuleName, sender)
}

// InitGenesis initializes the module's state from a genesis state. It runs the
// full GenesisState.Validate up front so direct keeper callers (tests, future
// migrations) get the same invariant enforcement as the JSON ValidateGenesis
// path on the AppModule.
func (k *Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) error {
	if err := data.Validate(); err != nil {
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

	return nil
}

// ExportGenesis exports the module's state to a genesis state.
func (k *Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
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

	var licenseCounts []types.LicenseCount
	if err := k.LicenseCounts.Walk(ctx, nil, func(typeID string, count uint64) (bool, error) {
		licenseCounts = append(licenseCounts, types.LicenseCount{LicenseTypeId: typeID, Count: count})
		return false, nil
	}); err != nil {
		panic(err)
	}

	return &types.GenesisState{
		LicenseTypes:  licenseTypes,
		Licenses:      licenses,
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
