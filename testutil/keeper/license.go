package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"
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
// consumes. The license namespace is registered and created with Owner as its
// namespace owner.
type LicenseFixture struct {
	Keeper           keeper.Keeper
	PermissionKeeper permissionkeeper.Keeper
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

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(licenseStoreKey),
		log.NewNopLogger(),
		authority,
		pk,
	)

	license.RegisterNamespace(pk, k)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	require.NoError(t, pk.Namespaces.Set(ctx, types.ModuleName, permissiontypes.Namespace{
		Module: types.ModuleName,
		Owner:  owner,
	}))

	return &LicenseFixture{
		Keeper:           k,
		PermissionKeeper: pk,
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

// LicenseKeeper returns a licenses keeper and context for testing.
func LicenseKeeper(t testing.TB) (keeper.Keeper, sdk.Context) {
	t.Helper()
	f := NewLicenseFixture(t)
	return f.Keeper, f.Ctx
}
