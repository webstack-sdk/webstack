package license

import (
	"context"
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"

	modulev1 "github.com/webstack-sdk/webstack/api/license/module/v1"
	"github.com/webstack-sdk/webstack/x/license/keeper"
	"github.com/webstack-sdk/webstack/x/license/types"
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
	PermissionKeeper permissionkeeper.Keeper
}

type ModuleOutputs struct {
	depinject.Out

	Module appmodule.AppModule
	Keeper keeper.Keeper
}

func ProvideModule(in ModuleInputs) ModuleOutputs {
	govAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	k := keeper.NewKeeper(in.Cdc, in.StoreService, log.NewLogger(os.Stderr), govAddr, in.PermissionKeeper)
	RegisterNamespace(in.PermissionKeeper, k)
	m := NewAppModule(in.Cdc, k)

	return ModuleOutputs{Module: m, Keeper: k}
}

// RegisterNamespace registers the license module's permission vocabulary and
// scope validator with the x/permission keeper. It must be called exactly once
// during app wiring, after the license keeper is constructed.
func RegisterNamespace(pk permissionkeeper.Keeper, k keeper.Keeper) {
	pk.RegisterNamespace(types.ModuleName, permissiontypes.NamespaceSpec{
		Permissions: types.ValidPermissions,
		// Grant scopes are license type ids; a grant may only reference an
		// existing type.
		ScopeExists: func(ctx context.Context, scope string) (bool, error) {
			_, found, err := k.GetLicenseType(ctx, scope)
			return found, err
		},
	})
}
