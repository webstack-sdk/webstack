package keeper

import (
	"context"
	"testing"
	"time"

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
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/webstack-sdk/webstack/testutil/sample"
	license "github.com/webstack-sdk/webstack/x/license"
	licensekeeper "github.com/webstack-sdk/webstack/x/license/keeper"
	licensetypes "github.com/webstack-sdk/webstack/x/license/types"
	network "github.com/webstack-sdk/webstack/x/network"
	networkkeeper "github.com/webstack-sdk/webstack/x/network/keeper"
	networktypes "github.com/webstack-sdk/webstack/x/network/types"
	permissionkeeper "github.com/webstack-sdk/webstack/x/permission/keeper"
	permissiontypes "github.com/webstack-sdk/webstack/x/permission/types"
)

// FakeAccountKeeper is a map-backed types.AccountKeeper for keeper tests.
type FakeAccountKeeper struct {
	accounts map[string]sdk.AccountI
	nextNum  uint64
}

func NewFakeAccountKeeper() *FakeAccountKeeper {
	return &FakeAccountKeeper{accounts: make(map[string]sdk.AccountI)}
}

func (f *FakeAccountKeeper) GetAccount(_ context.Context, addr sdk.AccAddress) sdk.AccountI {
	return f.accounts[addr.String()]
}

func (f *FakeAccountKeeper) HasAccount(_ context.Context, addr sdk.AccAddress) bool {
	_, ok := f.accounts[addr.String()]
	return ok
}

func (f *FakeAccountKeeper) NewAccountWithAddress(_ context.Context, addr sdk.AccAddress) sdk.AccountI {
	acc := authtypes.NewBaseAccountWithAddress(addr)
	if err := acc.SetAccountNumber(f.nextNum); err != nil {
		panic(err)
	}
	f.nextNum++
	return acc
}

func (f *FakeAccountKeeper) SetAccount(_ context.Context, acc sdk.AccountI) {
	f.accounts[acc.GetAddress().String()] = acc
}

// FakeBankKeeper records module transfers for keeper tests.
type FakeBankKeeper struct {
	// Transfers records each SendCoinsFromAccountToModule call.
	Transfers []FakeTransfer
	// FailWith, when set, makes transfers fail (e.g. to simulate an operator
	// that cannot pay the deauthorize fee).
	FailWith error
}

// FakeTransfer is one recorded SendCoinsFromAccountToModule call.
type FakeTransfer struct {
	From   string
	Module string
	Amount sdk.Coins
}

func (f *FakeBankKeeper) SendCoinsFromAccountToModule(_ context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	if f.FailWith != nil {
		return f.FailWith
	}
	f.Transfers = append(f.Transfers, FakeTransfer{From: senderAddr.String(), Module: recipientModule, Amount: amt})
	return nil
}

// NetworkFixture bundles a network keeper with the license and permission
// keepers it consumes, sharing one CommitMultiStore, plus fake account/bank
// keepers. The license namespace is owned by Owner; network params are the
// module defaults with LicenseTypes seeded to [LicenseType].
type NetworkFixture struct {
	Keeper           networkkeeper.Keeper
	LicenseKeeper    licensekeeper.Keeper
	PermissionKeeper permissionkeeper.Keeper
	AccountKeeper    *FakeAccountKeeper
	BankKeeper       *FakeBankKeeper
	Ctx              sdk.Context
	Owner            string

	// LicenseType is the counted license type seeded into the network params.
	LicenseType string
}

// NetworkFixtureBlockTime is the fixture context's initial block time; tests
// that roll days advance from here via WithBlockTime.
var NetworkFixtureBlockTime = time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

// NewNetworkFixture returns a network keeper wired to license, permission,
// and fake account/bank keepers for testing.
func NewNetworkFixture(t testing.TB) *NetworkFixture {
	t.Helper()

	licenseStoreKey := storetypes.NewKVStoreKey(licensetypes.StoreKey)
	permissionStoreKey := storetypes.NewKVStoreKey(permissiontypes.StoreKey)
	networkStoreKey := storetypes.NewKVStoreKey(networktypes.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(licenseStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(permissionStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(networkStoreKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	licensetypes.RegisterInterfaces(registry)
	permissiontypes.RegisterInterfaces(registry)
	networktypes.RegisterInterfaces(registry)
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
	bk := &FakeBankKeeper{}

	lk := licensekeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(licenseStoreKey),
		log.NewNopLogger(),
		authority,
		pk,
		ak,
	)

	nk := networkkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(networkStoreKey),
		log.NewNopLogger(),
		authority,
		lk,
		pk,
		ak,
		bk,
	)

	license.RegisterNamespace(pk, lk)
	network.RegisterNamespace(pk, nk)

	ctx := sdk.NewContext(stateStore, tmproto.Header{Height: 1, Time: NetworkFixtureBlockTime}, false, log.NewNopLogger())

	require.NoError(t, pk.Namespaces.Set(ctx, licensetypes.ModuleName, permissiontypes.Namespace{
		Module: licensetypes.ModuleName,
		Owner:  owner,
	}))
	require.NoError(t, pk.Namespaces.Set(ctx, networktypes.ModuleName, permissiontypes.Namespace{
		Module: networktypes.ModuleName,
		Owner:  owner,
	}))

	const licenseType = "node.license"
	params := networktypes.DefaultParams()
	params.LicenseTypes = []string{licenseType}
	require.NoError(t, nk.Params.Set(ctx, params))

	return &NetworkFixture{
		Keeper:           nk,
		LicenseKeeper:    lk,
		PermissionKeeper: pk,
		AccountKeeper:    ak,
		BankKeeper:       bk,
		Ctx:              ctx,
		Owner:            owner,
		LicenseType:      licenseType,
	}
}

// GrantNetwork writes a module-wide network-namespace grant directly into
// the permission keyset, bypassing the owner-gated msg path.
func (f *NetworkFixture) GrantNetwork(t testing.TB, grantee, permission string) {
	t.Helper()
	require.NoError(t, f.PermissionKeeper.Grants.Set(f.Ctx,
		collections.Join4(networktypes.ModuleName, grantee, permission, "")))
}

// IssueLicenses creates the fixture's counted license type on first use and
// issues count active licenses of it to holder, directly through license
// state (the license msg path is covered by the license module's own tests).
func (f *NetworkFixture) IssueLicenses(t testing.TB, holder string, count uint64) {
	t.Helper()
	f.IssueLicensesOfType(t, holder, f.LicenseType, count)
}

// IssueLicensesOfType is IssueLicenses for an arbitrary license type id.
func (f *NetworkFixture) IssueLicensesOfType(t testing.TB, holder, typeID string, count uint64) {
	t.Helper()

	lt, found, err := f.LicenseKeeper.GetLicenseType(f.Ctx, typeID)
	require.NoError(t, err)
	if !found {
		lt = licensetypes.LicenseType{
			Id:            typeID,
			Transferrable: false,
			MaxSupply:     math.ZeroInt(),
			IssuedCount:   math.ZeroInt(),
			ActiveCount:   math.ZeroInt(),
			RevokedCount:  math.ZeroInt(),
		}
	}

	for i := uint64(0); i < count; i++ {
		id, err := f.LicenseKeeper.LicenseCounts.Get(f.Ctx, typeID)
		if err != nil {
			id = 0
		}
		id++
		require.NoError(t, f.LicenseKeeper.LicenseCounts.Set(f.Ctx, typeID, id))
		require.NoError(t, f.LicenseKeeper.Licenses.Set(f.Ctx, collections.Join(typeID, id), licensetypes.License{
			Id:        id,
			Type:      typeID,
			Holder:    holder,
			StartDate: "2025-01-01",
			Status:    licensetypes.StatusActive,
		}))
		require.NoError(t, f.LicenseKeeper.ActiveLicensesByHolder.Set(f.Ctx, collections.Join3(holder, typeID, id)))
		lt.IssuedCount = lt.IssuedCount.AddRaw(1)
		lt.ActiveCount = lt.ActiveCount.AddRaw(1)
	}
	require.NoError(t, f.LicenseKeeper.LicenseTypes.Set(f.Ctx, typeID, lt))
}

// RevokeLicenses revokes count of holder's active licenses of the fixture's
// counted type through the license msg server, exercising the real
// revocation path (holder-index removal included).
func (f *NetworkFixture) RevokeLicenses(t testing.TB, holder string, count uint64) {
	t.Helper()

	revoker := sample.AccAddress()
	require.NoError(t, f.PermissionKeeper.Grants.Set(f.Ctx,
		collections.Join4(licensetypes.ModuleName, revoker, licensetypes.PermissionRevoke, f.LicenseType)))

	_, err := licensekeeper.NewMsgServerImpl(f.LicenseKeeper).RevokeLicenses(f.Ctx, &licensetypes.MsgRevokeLicenses{
		Revoker:       revoker,
		LicenseTypeId: f.LicenseType,
		Holder:        holder,
		Count:         count,
	})
	require.NoError(t, err)
}

// WithBlockTime returns the fixture context advanced to the given block time
// and the next height.
func (f *NetworkFixture) WithBlockTime(t time.Time) sdk.Context {
	header := f.Ctx.BlockHeader()
	header.Time = t
	header.Height++
	return f.Ctx.WithBlockHeader(header)
}
