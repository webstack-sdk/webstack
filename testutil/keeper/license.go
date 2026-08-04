package keeper

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/webstack-sdk/webstack/testutil/sample"
	license "github.com/webstack-sdk/webstack/x/license"
	"github.com/webstack-sdk/webstack/x/license/keeper"
	"github.com/webstack-sdk/webstack/x/license/types"
	permissionkeeper "github.com/webstack-sdk/webstack/x/permission/keeper"
	permissiontypes "github.com/webstack-sdk/webstack/x/permission/types"
)

// LicenseFixture bundles a license keeper with the permission keeper it
// consumes and a fake account keeper recording holder-account creation. The
// license namespace is registered and created with Owner as its namespace
// owner.
type LicenseFixture struct {
	Keeper           keeper.Keeper
	PermissionKeeper permissionkeeper.Keeper
	AccountKeeper    *FakeAccountKeeper
	Ctx              sdk.Context
	Owner            string
}

// NewLicenseFixture returns a license keeper wired to a permission keeper for
// testing, with the "license" namespace created under a fresh owner address.
func NewLicenseFixture(t testing.TB) *LicenseFixture {
	t.Helper()

	licenseStoreKey := storetypes.NewKVStoreKey(types.StoreKey)
	permissionStoreKey := storetypes.NewKVStoreKey(permissiontypes.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(licenseStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(permissionStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	permissiontypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	owner := sample.AccAddress()
	authority := sample.AccAddress()

	pk := permissionkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(permissionStoreKey),
		log.NewNopLogger(),
		authority,
	)

	ak := NewFakeAccountKeeper()

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(licenseStoreKey),
		log.NewNopLogger(),
		authority,
		pk,
		ak,
	)

	license.RegisterNamespace(pk, k)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	require.NoError(t, pk.Namespaces.Set(ctx, types.ModuleName, permissiontypes.Namespace{
		Module: types.ModuleName,
		Owner:  owner,
	}))

	// Creating license types is grant-gated, not owner-gated, so the owner
	// grants it to itself — the step a real chain takes once the namespace has
	// an owner. Tests that exercise the authorization rule itself call Ungrant
	// to take it back.
	require.NoError(t, pk.Grants.Set(ctx,
		collections.Join4(types.ModuleName, owner, types.PermissionCreateType, "")))

	return &LicenseFixture{
		Keeper:           k,
		PermissionKeeper: pk,
		AccountKeeper:    ak,
		Ctx:              ctx,
		Owner:            owner,
	}
}

// Grant writes a (grantee, permission, licenseType) grant directly into the
// permission keyset, bypassing the owner-gated msg path.
func (f *LicenseFixture) Grant(t testing.TB, grantee, permission, licenseType string) {
	t.Helper()
	require.NoError(t, f.PermissionKeeper.Grants.Set(f.Ctx,
		collections.Join4(types.ModuleName, grantee, permission, licenseType)))
}

// Ungrant removes a (grantee, permission, licenseType) grant, the inverse of
// Grant. Removal is idempotent, so revoking an absent grant is a no-op.
func (f *LicenseFixture) Ungrant(t testing.TB, grantee, permission, licenseType string) {
	t.Helper()
	require.NoError(t, f.PermissionKeeper.Grants.Remove(f.Ctx,
		collections.Join4(types.ModuleName, grantee, permission, licenseType)))
}

// SeedActiveLicenses writes count active licenses of typeID for holder
// straight into license state, creating the license type on first use and
// keeping every index and counter consistent.
//
// It takes the keeper rather than a fixture so both the license and network
// fixtures can share it. This is the ONLY place test code allocates license
// ids: the id scheme and the set of indexes a license must appear in live in
// one place, so they cannot drift apart from the keeper's own issuance path.
func SeedActiveLicenses(t testing.TB, k keeper.Keeper, ctx sdk.Context, holder, typeID string, count uint64) {
	t.Helper()

	lt, found, err := k.GetLicenseType(ctx, typeID)
	require.NoError(t, err)
	if !found {
		lt = types.LicenseType{
			Id:           typeID,
			MaxSupply:    math.ZeroInt(),
			IssuedCount:  math.ZeroInt(),
			ActiveCount:  math.ZeroInt(),
			RevokedCount: math.ZeroInt(),
		}
	}

	for i := uint64(0); i < count; i++ {
		id, err := k.NextLicenseID.Get(ctx)
		if errors.Is(err, collections.ErrNotFound) {
			id = types.FirstLicenseID
		} else {
			require.NoError(t, err)
		}
		require.NoError(t, k.NextLicenseID.Set(ctx, id+1))

		require.NoError(t, k.Licenses.Set(ctx, id, types.License{
			Id:        id,
			Type:      typeID,
			Holder:    holder,
			StartDate: "2025-01-01",
			Status:    types.StatusActive,
		}))
		require.NoError(t, k.LicensesByType.Set(ctx, collections.Join(typeID, id)))
		require.NoError(t, k.ActiveLicensesByHolder.Set(ctx, collections.Join3(holder, typeID, id)))
		lt.IssuedCount = lt.IssuedCount.AddRaw(1)
		lt.ActiveCount = lt.ActiveCount.AddRaw(1)
	}
	require.NoError(t, k.LicenseTypes.Set(ctx, typeID, lt))
}

// LicenseKeeper returns a licenses keeper and context for testing.
func LicenseKeeper(t testing.TB) (keeper.Keeper, sdk.Context) {
	t.Helper()
	f := NewLicenseFixture(t)
	return f.Keeper, f.Ctx
}
