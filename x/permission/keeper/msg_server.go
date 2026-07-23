package keeper

import (
	"context"
	"fmt"
	"strings"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/webstack-sdk/webstack/x/permission/types"
)

type msgServer struct {
	k Keeper
}

var _ types.MsgServer = msgServer{}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{k: keeper}
}

// CreateNamespace creates the grant namespace for a module and sets its owner.
// The module must be registered in this binary, so a namespace can never exist
// without a permission vocabulary to validate grants against.
func (ms msgServer) CreateNamespace(ctx context.Context, msg *types.MsgCreateNamespace) (*types.MsgCreateNamespaceResponse, error) {
	if ms.k.authority != msg.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.k.authority, msg.Authority)
	}

	if _, registered := ms.k.registry[msg.Module]; !registered {
		return nil, types.ErrModuleNotRegistered.Wrapf("module %q is not registered in this binary", msg.Module)
	}

	if _, found, err := ms.k.GetNamespace(ctx, msg.Module); err != nil {
		return nil, err
	} else if found {
		return nil, types.ErrNamespaceExists.Wrapf("namespace for module %q already exists", msg.Module)
	}

	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return nil, fmt.Errorf("invalid owner address %q: %w", msg.Owner, err)
	}

	ns := types.Namespace{Module: msg.Module, Owner: msg.Owner}
	if err := ms.k.Namespaces.Set(ctx, msg.Module, ns); err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCreateNamespace,
		sdk.NewAttribute(types.AttributeKeyModule, msg.Module),
		sdk.NewAttribute(types.AttributeKeyOwner, msg.Owner),
	))

	return &types.MsgCreateNamespaceResponse{}, nil
}

// UpdateNamespaceOwner rotates a namespace's owner via governance, so a lost
// or compromised owner key can be recovered without its cooperation.
func (ms msgServer) UpdateNamespaceOwner(ctx context.Context, msg *types.MsgUpdateNamespaceOwner) (*types.MsgUpdateNamespaceOwnerResponse, error) {
	if ms.k.authority != msg.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.k.authority, msg.Authority)
	}

	ns, found, err := ms.k.GetNamespace(ctx, msg.Module)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, types.ErrNamespaceNotFound.Wrapf("namespace for module %q not found", msg.Module)
	}

	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return nil, fmt.Errorf("invalid owner address %q: %w", msg.Owner, err)
	}

	ns.Owner = msg.Owner
	if err := ms.k.Namespaces.Set(ctx, msg.Module, ns); err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeUpdateNamespaceOwner,
		sdk.NewAttribute(types.AttributeKeyModule, msg.Module),
		sdk.NewAttribute(types.AttributeKeyOwner, msg.Owner),
	))

	return &types.MsgUpdateNamespaceOwnerResponse{}, nil
}

// TransferOwnership hands a namespace to a new owner. The current owner signs.
func (ms msgServer) TransferOwnership(ctx context.Context, msg *types.MsgTransferOwnership) (*types.MsgTransferOwnershipResponse, error) {
	ns, found, err := ms.k.GetNamespace(ctx, msg.Module)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, types.ErrNamespaceNotFound.Wrapf("namespace for module %q not found", msg.Module)
	}

	if ns.Owner != msg.Owner {
		return nil, types.ErrUnauthorized.Wrapf("signer %s is not the owner %s of namespace %q", msg.Owner, ns.Owner, msg.Module)
	}

	if _, err := sdk.AccAddressFromBech32(msg.NewOwner); err != nil {
		return nil, fmt.Errorf("invalid new owner address %q: %w", msg.NewOwner, err)
	}

	ns.Owner = msg.NewOwner
	if err := ms.k.Namespaces.Set(ctx, msg.Module, ns); err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeTransferOwnership,
		sdk.NewAttribute(types.AttributeKeyModule, msg.Module),
		sdk.NewAttribute(types.AttributeKeyOwner, msg.NewOwner),
	))

	return &types.MsgTransferOwnershipResponse{}, nil
}

// GrantPermissions merges the incoming grants with any existing grants for the
// grantee. (permission, scope) pairs that already exist are deduped; nothing
// is ever removed by this message. Use MsgRevokePermissions to remove specific
// pairs.
func (ms msgServer) GrantPermissions(ctx context.Context, msg *types.MsgGrantPermissions) (*types.MsgGrantPermissionsResponse, error) {
	ns, found, err := ms.k.GetNamespace(ctx, msg.Module)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, types.ErrNamespaceNotFound.Wrapf("namespace for module %q not found", msg.Module)
	}
	if ns.Owner != msg.Owner {
		return nil, types.ErrUnauthorized.Wrapf("signer %s is not the owner %s of namespace %q", msg.Owner, ns.Owner, msg.Module)
	}

	spec, registered := ms.k.registry[msg.Module]
	if !registered {
		return nil, types.ErrModuleNotRegistered.Wrapf("module %q is not registered in this binary", msg.Module)
	}

	if _, err := sdk.AccAddressFromBech32(msg.Grantee); err != nil {
		return nil, fmt.Errorf("invalid grantee address %q: %w", msg.Grantee, err)
	}

	if len(msg.Grants) > types.MaxGrants {
		return nil, fmt.Errorf("grants length %d exceeds max %d", len(msg.Grants), types.MaxGrants)
	}

	// Resolve and validate every (permission, scope) pair before writing
	// anything, so a partially-invalid message grants nothing.
	type pair struct{ permission, scope string }
	var pairs []pair
	for i, grant := range msg.Grants {
		if len(grant.Scopes) > types.MaxGrants {
			return nil, fmt.Errorf("grant %d scopes length %d exceeds max %d", i, len(grant.Scopes), types.MaxGrants)
		}
		// An empty scope list is the module-wide grant form; it is rejected by
		// validateGrantPair when the module scopes its permissions.
		scopes := grant.Scopes
		if len(scopes) == 0 {
			scopes = []string{""}
		}
		for _, scope := range scopes {
			if err := ms.k.validateGrantPair(ctx, msg.Module, spec, grant.Permission, scope); err != nil {
				return nil, fmt.Errorf("grant %d: %w", i, err)
			}
			pairs = append(pairs, pair{permission: grant.Permission, scope: scope})
		}
	}

	// Grants are unioned into the flat keyset: existing pairs are untouched
	// and re-granted pairs are idempotent overwrites.
	for _, p := range pairs {
		if err := ms.k.Grants.Set(ctx, collections.Join4(msg.Module, msg.Grantee, p.permission, p.scope)); err != nil {
			return nil, err
		}
	}

	var perms []string
	var scopeLists []string
	for _, grant := range msg.Grants {
		perms = append(perms, grant.Permission)
		scopeLists = append(scopeLists, strings.Join(grant.Scopes, ","))
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeGrantPermissions,
		sdk.NewAttribute(types.AttributeKeyModule, msg.Module),
		sdk.NewAttribute(types.AttributeKeyGrantee, msg.Grantee),
		sdk.NewAttribute(types.AttributeKeyPermissions, strings.Join(perms, ",")),
		sdk.NewAttribute(types.AttributeKeyScopes, strings.Join(scopeLists, ";")),
	))

	return &types.MsgGrantPermissionsResponse{}, nil
}

// RevokePermissions removes specific (permission, scope) pairs from a grantee
// within a namespace. Pairs that are not currently present are silently
// ignored (Remove is idempotent) — the caller can safely re-send the same
// revoke.
func (ms msgServer) RevokePermissions(ctx context.Context, msg *types.MsgRevokePermissions) (*types.MsgRevokePermissionsResponse, error) {
	ns, found, err := ms.k.GetNamespace(ctx, msg.Module)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, types.ErrNamespaceNotFound.Wrapf("namespace for module %q not found", msg.Module)
	}
	if ns.Owner != msg.Owner {
		return nil, types.ErrUnauthorized.Wrapf("signer %s is not the owner %s of namespace %q", msg.Owner, ns.Owner, msg.Module)
	}

	if len(msg.Permissions) > types.MaxGrants {
		return nil, fmt.Errorf("permissions length %d exceeds max %d", len(msg.Permissions), types.MaxGrants)
	}

	for _, p := range msg.Permissions {
		if err := ms.k.Grants.Remove(ctx, collections.Join4(msg.Module, msg.Grantee, p.Permission, p.Scope)); err != nil {
			return nil, err
		}
	}

	var revokedPerms []string
	var revokedScopes []string
	for _, p := range msg.Permissions {
		revokedPerms = append(revokedPerms, p.Permission)
		revokedScopes = append(revokedScopes, p.Scope)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeRevokePermissions,
		sdk.NewAttribute(types.AttributeKeyModule, msg.Module),
		sdk.NewAttribute(types.AttributeKeyGrantee, msg.Grantee),
		sdk.NewAttribute(types.AttributeKeyPermissions, strings.Join(revokedPerms, ",")),
		sdk.NewAttribute(types.AttributeKeyScopes, strings.Join(revokedScopes, ",")),
	))

	return &types.MsgRevokePermissionsResponse{}, nil
}
