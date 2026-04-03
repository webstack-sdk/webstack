package keeper

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/math"

	"github.com/webstack/webstack/x/licenses/types"
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
	AdminKeys     collections.Map[string, types.AdminKey]

	// indexes
	LicenseByHolder collections.Map[collections.Triple[string, string, uint64], uint64]

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
		AdminKeys:     collections.NewMap(sb, types.AdminKeyPrefix, "admin_keys", collections.StringKey, codec.CollValue[types.AdminKey](cdc)),

		LicenseByHolder: collections.NewMap(sb, types.LicenseByHolderPrefix, "license_by_holder", collections.TripleKeyCodec(collections.StringKey, collections.StringKey, collections.Uint64Key), collections.Uint64Value),

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

// HasPermission checks if an address has a specific permission for a license type.
func (k Keeper) HasPermission(ctx context.Context, address, permission, licenseTypeID string) bool {
	ok, _ := k.hasAdminPermission(ctx, address, licenseTypeID, permission)
	return ok
}

// InitGenesis initializes the module's state from a genesis state.
func (k *Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) error {
	if err := data.Params.Validate(); err != nil {
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

	for _, license := range data.Licenses {
		if err := k.Licenses.Set(ctx, collections.Join(license.Type, license.Id), license); err != nil {
			return err
		}
		if err := k.LicenseByHolder.Set(ctx, collections.Join3(license.Holder, license.Type, license.Id), license.Id); err != nil {
			return err
		}
	}

	for _, ak := range data.AdminKeys {
		if err := k.AdminKeys.Set(ctx, ak.Address, ak); err != nil {
			return err
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

	var adminKeys []types.AdminKey
	if err := k.AdminKeys.Walk(ctx, nil, func(_ string, ak types.AdminKey) (bool, error) {
		adminKeys = append(adminKeys, ak)
		return false, nil
	}); err != nil {
		panic(err)
	}

	return &types.GenesisState{
		Params:       params,
		LicenseTypes: licenseTypes,
		Licenses:     licenses,
		AdminKeys:    adminKeys,
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

// hasAdminPermission checks if an address has a specific permission for a license type.
func (k Keeper) hasAdminPermission(ctx context.Context, address string, licenseTypeID string, permission string) (bool, error) {
	ak, err := k.AdminKeys.Get(ctx, address)
	if err != nil {
		return false, nil
	}

	for _, grant := range ak.Grants {
		if grant.Permission != permission {
			continue
		}
		for _, lt := range grant.LicenseTypes {
			if lt == licenseTypeID || lt == "*" {
				return true, nil
			}
		}
	}

	return false, nil
}

// isOwner checks if the sender is the module owner.
func (k Keeper) isOwner(ctx context.Context, sender string) (bool, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return false, err
	}
	return params.Owner == sender, nil
}

// issueSingleLicense creates a single license and returns its ID.
func (k Keeper) issueSingleLicense(ctx context.Context, typeID string, lt types.LicenseType, holder string, startDate string, endDate string) (uint64, error) {
	// Check max supply
	if lt.MaxSupply.IsPositive() && lt.IssuedCount.GTE(lt.MaxSupply) {
		return 0, types.ErrMaxSupplyReached.Wrapf("type %s: issued %s / max %s", typeID, lt.IssuedCount, lt.MaxSupply)
	}

	id, err := k.nextLicenseID(ctx, typeID)
	if err != nil {
		return 0, err
	}

	license := types.License{
		Id:        id,
		Type:      typeID,
		Holder:    holder,
		StartDate: startDate,
		EndDate:   endDate,
		Status:    "active",
	}

	if err := k.Licenses.Set(ctx, collections.Join(typeID, id), license); err != nil {
		return 0, err
	}

	if err := k.LicenseByHolder.Set(ctx, collections.Join3(holder, typeID, id), id); err != nil {
		return 0, err
	}

	// Increment issued count
	lt.IssuedCount = lt.IssuedCount.Add(math.OneInt())
	if err := k.LicenseTypes.Set(ctx, typeID, lt); err != nil {
		return 0, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"license_issued",
		sdk.NewAttribute("license_type_id", typeID),
		sdk.NewAttribute("license_id", fmt.Sprintf("%d", id)),
		sdk.NewAttribute("holder", holder),
	))

	return id, nil
}
