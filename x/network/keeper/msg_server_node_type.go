package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/webstack-sdk/webstack/x/network/types"
)

// CreateNodeType registers a node type and binds it to a license type.
//
// Registration is gated twice, and the two gates answer different questions.
// The module-wide "nodetype.create" grant decides *whether* the signer may
// define node types at all, and stays revocable on its own. The creator match
// against the license type decides *which* license types they may define node
// types against: a license type is only ever extended by the address that
// created it, so one tenant cannot attach node types to another's licenses
// even while holding the grant.
//
// The binding is one-to-one: a node type names exactly one license type, and a
// license type backs at most one node type. It is also fixed at creation, and
// node type records are never removed — activation resolves a node's type
// through this registry, so a type that vanished or re-pointed would strand
// every node already carrying it.
func (ms msgServer) CreateNodeType(ctx context.Context, msg *types.MsgCreateNodeType) (*types.MsgCreateNodeTypeResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	hasPerm, err := ms.k.hasPermission(ctx, msg.Creator, types.PermissionNodeTypeCreate)
	if err != nil {
		return nil, err
	}
	if !hasPerm {
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "%s does not have the %s permission", msg.Creator, types.PermissionNodeTypeCreate)
	}

	// Node type ids are single-use: the existing record is what reserves the
	// id, so a duplicate is rejected rather than overwriting a live binding.
	exists, err := ms.k.NodeTypes.Has(ctx, msg.Id)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errorsmod.Wrapf(types.ErrNodeTypeExists, "node type %s", msg.Id)
	}

	// The binding is one-to-one: a license type backs at most one node type.
	// Checked before the creator match so the caller learns the license type is
	// taken rather than being told they are unauthorized for it.
	bound, err := ms.k.NodeTypeByLicenseType.Get(ctx, msg.LicenseTypeId)
	switch {
	case err == nil:
		return nil, errorsmod.Wrapf(types.ErrLicenseTypeBound, "license type %s already backs node type %s", msg.LicenseTypeId, bound)
	case !errors.Is(err, collections.ErrNotFound):
		return nil, err
	}

	creator, found, err := ms.k.licenseKeeper.LicenseTypeCreator(ctx, msg.LicenseTypeId)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errorsmod.Wrapf(types.ErrLicenseTypeNotFound, "license type %s", msg.LicenseTypeId)
	}
	if creator != msg.Creator {
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "%s did not create license type %s", msg.Creator, msg.LicenseTypeId)
	}

	nodeType := types.NodeType{
		Id:            msg.Id,
		Creator:       msg.Creator,
		LicenseTypeId: msg.LicenseTypeId,
	}
	if err := ms.k.NodeTypes.Set(ctx, msg.Id, nodeType); err != nil {
		return nil, err
	}
	// Written once and never moved: the binding is immutable, so this map
	// cannot drift from the record it mirrors.
	if err := ms.k.NodeTypeByLicenseType.Set(ctx, msg.LicenseTypeId, msg.Id); err != nil {
		return nil, err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCreateNodeType,
		sdk.NewAttribute(types.AttributeKeyNodeType, msg.Id),
		sdk.NewAttribute(types.AttributeKeyCreator, msg.Creator),
		sdk.NewAttribute(types.AttributeKeyLicenseTypeID, msg.LicenseTypeId),
	))

	return &types.MsgCreateNodeTypeResponse{}, nil
}
