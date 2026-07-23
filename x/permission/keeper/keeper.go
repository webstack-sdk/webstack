package keeper

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/webstack-sdk/webstack/x/permission/types"
)

type Keeper struct {
	cdc    codec.BinaryCodec
	logger log.Logger

	// state management
	Schema collections.Schema

	// Namespaces maps a consuming module's name to its namespace, whose owner
	// controls grants within it.
	Namespaces collections.Map[string, types.Namespace]

	// Grants is the flat set of (module, grantee, permission, scope) grant
	// keys, so a permission check is a single point-read. The flat Grant views
	// served by queries and genesis are read straight off this keyset in key
	// order.
	Grants collections.KeySet[collections.Quad[string, string, string, string]]

	// registry holds the in-process namespace specs consuming modules register
	// at wiring time. The map is shared by every value copy of the keeper.
	registry map[string]types.NamespaceSpec

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

		Namespaces: collections.NewMap(sb, types.NamespacePrefix, "namespaces", collections.StringKey, codec.CollValue[types.Namespace](cdc)),
		Grants:     collections.NewKeySet(sb, types.GrantPrefix, "grants", collections.QuadKeyCodec(collections.StringKey, collections.StringKey, collections.StringKey, collections.StringKey)),

		registry: make(map[string]types.NamespaceSpec),

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

// RegisterNamespace records a consuming module's permission vocabulary and
// scope validator. It must be called during app wiring, before any state
// access; registering an invalid spec or the same module twice is a wiring
// bug, so both panic.
func (k Keeper) RegisterNamespace(module string, spec types.NamespaceSpec) {
	if err := types.ValidateName("module", module); err != nil {
		panic(err)
	}
	if err := spec.Validate(); err != nil {
		panic(fmt.Errorf("namespace spec for module %q: %w", module, err))
	}
	if _, exists := k.registry[module]; exists {
		panic(fmt.Errorf("namespace for module %q is already registered", module))
	}
	k.registry[module] = spec
}

// Spec returns the registered namespace spec for a module.
func (k Keeper) Spec(module string) (types.NamespaceSpec, bool) {
	spec, ok := k.registry[module]
	return spec, ok
}

// GetNamespace returns a namespace by module name and whether it was found.
func (k Keeper) GetNamespace(ctx context.Context, module string) (types.Namespace, bool, error) {
	ns, err := k.Namespaces.Get(ctx, module)
	if err != nil {
		return types.Namespace{}, false, nil
	}
	return ns, true, nil
}

// IsOwner reports whether addr owns the module's namespace. A missing
// namespace is surfaced as ErrNamespaceNotFound so callers can distinguish
// "not the owner" from "namespace does not exist".
func (k Keeper) IsOwner(ctx context.Context, module, addr string) (bool, error) {
	ns, found, err := k.GetNamespace(ctx, module)
	if err != nil {
		return false, err
	}
	if !found {
		return false, types.ErrNamespaceNotFound.Wrapf("namespace for module %q not found", module)
	}
	return ns.Owner == addr, nil
}

// Has reports whether grantee holds the (permission, scope) grant within the
// module's namespace. A missing grant returns (false, nil); a store error is
// surfaced so the caller can fail the tx instead of silently denying.
func (k Keeper) Has(ctx context.Context, module, grantee, permission, scope string) (bool, error) {
	return k.Grants.Has(ctx, collections.Join4(module, grantee, permission, scope))
}

// HasPermission is the yes/no convenience form of Has: any underlying store
// error is treated as "no permission". Callers that must distinguish "missing"
// from "store failure" should use Has.
func (k Keeper) HasPermission(ctx context.Context, module, grantee, permission, scope string) bool {
	ok, _ := k.Has(ctx, module, grantee, permission, scope)
	return ok
}

// validateGrantPair checks a (permission, scope) pair against the module's
// registered spec: the permission must be in the vocabulary and the scope must
// satisfy the module's scope rules.
func (k Keeper) validateGrantPair(ctx context.Context, module string, spec types.NamespaceSpec, permission, scope string) error {
	if !spec.HasPermission(permission) {
		return types.ErrInvalidPermission.Wrapf("permission %q is not registered for module %q", permission, module)
	}
	if spec.ScopeExists == nil {
		return nil
	}
	if scope == "" {
		return types.ErrInvalidScope.Wrapf("module %q scopes its permissions: scope must not be empty", module)
	}
	exists, err := spec.ScopeExists(ctx, scope)
	if err != nil {
		return err
	}
	if !exists {
		return types.ErrInvalidScope.Wrapf("scope %q does not exist in module %q", scope, module)
	}
	return nil
}

// InitGenesis initializes the module's state from a genesis state. On top of
// the stateless GenesisState.Validate, every namespace must be registered in
// this binary and every grant must satisfy the registered spec — the same
// invariants the msg handlers enforce.
func (k *Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) error {
	if err := data.Validate(); err != nil {
		return err
	}

	for _, ns := range data.Namespaces {
		if _, registered := k.registry[ns.Module]; !registered {
			return types.ErrModuleNotRegistered.Wrapf("genesis namespace %q is not registered in this binary", ns.Module)
		}
		if err := k.Namespaces.Set(ctx, ns.Module, ns); err != nil {
			return err
		}
	}

	for _, g := range data.Grants {
		spec := k.registry[g.Module]
		if err := k.validateGrantPair(ctx, g.Module, spec, g.Permission, g.Scope); err != nil {
			return err
		}
		if err := k.Grants.Set(ctx, collections.Join4(g.Module, g.Grantee, g.Permission, g.Scope)); err != nil {
			return err
		}
	}

	return nil
}

// ExportGenesis exports the module's state to a genesis state. Both lists come
// straight off their collections in key order, so the export is deterministic.
func (k *Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	var namespaces []types.Namespace
	if err := k.Namespaces.Walk(ctx, nil, func(_ string, ns types.Namespace) (bool, error) {
		namespaces = append(namespaces, ns)
		return false, nil
	}); err != nil {
		panic(err)
	}

	var grants []types.Grant
	if err := k.Grants.Walk(ctx, nil, func(key collections.Quad[string, string, string, string]) (bool, error) {
		grants = append(grants, types.Grant{
			Module:     key.K1(),
			Grantee:    key.K2(),
			Permission: key.K3(),
			Scope:      key.K4(),
		})
		return false, nil
	}); err != nil {
		panic(err)
	}

	return &types.GenesisState{
		Namespaces: namespaces,
		Grants:     grants,
	}
}
