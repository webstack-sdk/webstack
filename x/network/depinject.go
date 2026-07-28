package network

import (
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"

	modulev1 "github.com/webstack-sdk/webstack/api/network/module/v1"
	licensekeeper "github.com/webstack-sdk/webstack/x/license/keeper"
	"github.com/webstack-sdk/webstack/x/network/keeper"
	"github.com/webstack-sdk/webstack/x/network/types"
	permissionkeeper "github.com/webstack-sdk/webstack/x/permission/keeper"
	permissiontypes "github.com/webstack-sdk/webstack/x/permission/types"
)

var _ appmodule.AppModule = AppModule{}

func init() {
	appmodule.Register(
		&modulev1.Module{},
		appmodule.Provide(ProvideModule),
	)
}

type ModuleInputs struct {
	depinject.In

	Cdc              codec.Codec
	StoreService     store.KVStoreService
	LicenseKeeper    licensekeeper.Keeper
	PermissionKeeper permissionkeeper.Keeper
	AccountKeeper    authkeeper.AccountKeeper
	BankKeeper       bankkeeper.Keeper
}

type ModuleOutputs struct {
	depinject.Out

	Module appmodule.AppModule
	Keeper keeper.Keeper
}

func ProvideModule(in ModuleInputs) ModuleOutputs {
	govAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	k := keeper.NewKeeper(in.Cdc, in.StoreService, log.NewLogger(os.Stderr), govAddr, in.LicenseKeeper, in.PermissionKeeper, in.AccountKeeper, in.BankKeeper)
	RegisterNamespace(in.PermissionKeeper, k)
	m := NewAppModule(in.Cdc, k)

	return ModuleOutputs{Module: m, Keeper: k}
}

// RegisterNamespace registers the network module's permission vocabulary
// with the x/permission keeper. It must be called exactly once during app
// wiring, after the network keeper is constructed. Grants are module-wide:
// the namespace has no scope function, and grant scopes are empty strings.
func RegisterNamespace(pk permissionkeeper.Keeper, _ keeper.Keeper) {
	pk.RegisterNamespace(types.ModuleName, permissiontypes.NamespaceSpec{
		Permissions: types.ValidPermissions,
	})
}
