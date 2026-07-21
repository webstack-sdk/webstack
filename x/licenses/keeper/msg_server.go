package keeper

import (
	"context"
	"fmt"
	"strings"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/webstack-sdk/webstack/x/licenses/types"
)

type msgServer struct {
	k Keeper
}

var _ types.MsgServer = msgServer{}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{k: keeper}
}

func (ms msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if ms.k.authority != msg.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.k.authority, msg.Authority)
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if err := ms.k.Params.Set(ctx, msg.Params); err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeUpdateParams,
		sdk.NewAttribute(types.AttributeKeyOwner, msg.Params.Owner),
	))

	return &types.MsgUpdateParamsResponse{}, nil
}

func (ms msgServer) CreateLicenseType(ctx context.Context, msg *types.MsgCreateLicenseType) (*types.MsgCreateLicenseTypeResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	isOwner, err := ms.k.isOwner(ctx, msg.Owner)
	if err != nil {
		return nil, err
	}
	if !isOwner {
		params, _ := ms.k.Params.Get(ctx)
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "signer %s is not the module owner %s", msg.Owner, params.Owner)
	}

	if msg.Id == "" {
		return nil, errorsmod.Wrap(types.ErrLicenseTypeNotFound, "license type id cannot be empty")
	}

	if err := types.ValidateMaxSupply(msg.MaxSupply); err != nil {
		return nil, err
	}

	_, err = ms.k.LicenseTypes.Get(ctx, msg.Id)
	if err == nil {
		return nil, errorsmod.Wrapf(types.ErrLicenseTypeExists, "license type %s already exists", msg.Id)
	}

	lt := types.LicenseType{
		Id:            msg.Id,
		Transferrable: msg.Transferrable,
		MaxSupply:     msg.MaxSupply,
		IssuedCount:   math.ZeroInt(),
		ActiveCount:   math.ZeroInt(),
		RevokedCount:  math.ZeroInt(),
	}

	if err := ms.k.LicenseTypes.Set(ctx, msg.Id, lt); err != nil {
		return nil, err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeCreateLicenseType,
		sdk.NewAttribute(types.AttributeKeyLicenseTypeID, msg.Id),
	))

	return &types.MsgCreateLicenseTypeResponse{}, nil
}

func (ms msgServer) UpdateLicenseType(ctx context.Context, msg *types.MsgUpdateLicenseType) (*types.MsgUpdateLicenseTypeResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	isOwner, err := ms.k.isOwner(ctx, msg.Owner)
	if err != nil {
		return nil, err
	}
	if !isOwner {
		params, _ := ms.k.Params.Get(ctx)
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "signer %s is not the module owner %s", msg.Owner, params.Owner)
	}

	if err := types.ValidateMaxSupply(msg.MaxSupply); err != nil {
		return nil, err
	}

	lt, err := ms.k.LicenseTypes.Get(ctx, msg.Id)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLicenseTypeNotFound, "license type %s not found", msg.Id)
	}

	if !msg.MaxSupply.IsZero() && lt.IssuedCount.GT(msg.MaxSupply) {
		return nil, errorsmod.Wrapf(types.ErrMaxSupplyReached, "cannot set max_supply to %s: %s licenses already issued", msg.MaxSupply.String(), lt.IssuedCount.String())
	}

	lt.Transferrable = msg.Transferrable
	lt.MaxSupply = msg.MaxSupply

	if err := ms.k.LicenseTypes.Set(ctx, msg.Id, lt); err != nil {
		return nil, err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeUpdateLicenseType,
		sdk.NewAttribute(types.AttributeKeyLicenseTypeID, msg.Id),
	))

	return &types.MsgUpdateLicenseTypeResponse{}, nil
}

// GrantAdminPermissions merges the incoming grants with any existing grants for the
// given address. (permission, license type) pairs that already exist are deduped;
// nothing is ever removed by this message. Use MsgRevokeAdminKeyPermissions to remove
// specific pairs.
func (ms msgServer) GrantAdminPermissions(ctx context.Context, msg *types.MsgGrantAdminPermissions) (*types.MsgGrantAdminPermissionsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	isOwner, err := ms.k.isOwner(ctx, msg.Owner)
	if err != nil {
		return nil, err
	}
	if !isOwner {
		params, _ := ms.k.Params.Get(ctx)
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "signer %s is not the module owner %s", msg.Owner, params.Owner)
	}

	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return nil, fmt.Errorf("invalid address %q: %w", msg.Address, err)
	}

	if len(msg.Grants) > types.MaxAdminGrants {
		return nil, fmt.Errorf("grants length %d exceeds max %d", len(msg.Grants), types.MaxAdminGrants)
	}

	for i, grant := range msg.Grants {
		if len(grant.LicenseTypes) > types.MaxAdminGrants {
			return nil, fmt.Errorf("grant %d license_types length %d exceeds max %d", i, len(grant.LicenseTypes), types.MaxAdminGrants)
		}
		if !types.IsValidPermission(grant.Permission) {
			return nil, fmt.Errorf("invalid permission %q: must be one of issue, revoke", grant.Permission)
		}
		if len(grant.LicenseTypes) == 0 {
			return nil, fmt.Errorf("grant for permission %q must include at least one license type", grant.Permission)
		}
		for _, lt := range grant.LicenseTypes {
			if _, found, err := ms.k.GetLicenseType(ctx, lt); err != nil {
				return nil, err
			} else if !found {
				return nil, errorsmod.Wrapf(types.ErrLicenseTypeNotFound, "license type %q in grant for permission %q does not exist", lt, grant.Permission)
			}
		}
	}

	// Grants are unioned into the flat keyset: existing pairs are untouched and
	// re-granted pairs are idempotent overwrites.
	for _, grant := range msg.Grants {
		for _, lt := range grant.LicenseTypes {
			if err := ms.k.AdminGrants.Set(ctx, collections.Join3(msg.Address, grant.Permission, lt)); err != nil {
				return nil, err
			}
		}
	}

	var perms []string
	var grantTypes []string
	for _, grant := range msg.Grants {
		perms = append(perms, grant.Permission)
		grantTypes = append(grantTypes, strings.Join(grant.LicenseTypes, ","))
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeGrantAdminPermissions,
		sdk.NewAttribute(types.AttributeKeyAddress, msg.Address),
		sdk.NewAttribute(types.AttributeKeyPermissions, strings.Join(perms, ",")),
		sdk.NewAttribute(types.AttributeKeyGrantTypes, strings.Join(grantTypes, ";")),
	))

	return &types.MsgGrantAdminPermissionsResponse{}, nil
}

// RevokeAdminKeyPermissions removes specific (license type, permission) pairs
// from an admin key. Pairs that are not currently present are silently ignored
// (Remove is idempotent) — the caller can safely re-send the same revoke.
func (ms msgServer) RevokeAdminKeyPermissions(ctx context.Context, msg *types.MsgRevokeAdminKeyPermissions) (*types.MsgRevokeAdminKeyPermissionsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	isOwner, err := ms.k.isOwner(ctx, msg.Owner)
	if err != nil {
		return nil, err
	}
	if !isOwner {
		params, _ := ms.k.Params.Get(ctx)
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "signer %s is not the module owner %s", msg.Owner, params.Owner)
	}

	if len(msg.Permissions) > types.MaxAdminGrants {
		return nil, fmt.Errorf("permissions length %d exceeds max %d", len(msg.Permissions), types.MaxAdminGrants)
	}

	for _, p := range msg.Permissions {
		if err := ms.k.AdminGrants.Remove(ctx, collections.Join3(msg.Address, p.Permission, p.LicenseTypeId)); err != nil {
			return nil, err
		}
	}

	var revokedPerms []string
	var revokedTypes []string
	for _, p := range msg.Permissions {
		revokedPerms = append(revokedPerms, p.Permission)
		revokedTypes = append(revokedTypes, p.LicenseTypeId)
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeRevokeAdminKeyPermissions,
		sdk.NewAttribute(types.AttributeKeyAddress, msg.Address),
		sdk.NewAttribute(types.AttributeKeyPermissions, strings.Join(revokedPerms, ",")),
		sdk.NewAttribute(types.AttributeKeyGrantTypes, strings.Join(revokedTypes, ",")),
	))

	return &types.MsgRevokeAdminKeyPermissionsResponse{}, nil
}

// IssueLicenses issues licenses for each entry in the message. Each entry
// carries its own license type, holder, dates, and count; the signer must hold
// the "issue" grant for every referenced license type. All entries are
// validated before any license is issued, and the returned ids are flattened
// in entry order.
func (ms msgServer) IssueLicenses(ctx context.Context, msg *types.MsgIssueLicenses) (*types.MsgIssueLicensesResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if len(msg.Entries) == 0 {
		return nil, errorsmod.Wrap(types.ErrEmptyBatchEntries, "entries must not be empty")
	}
	if len(msg.Entries) > types.MaxIssueBatchSize {
		return nil, fmt.Errorf("entries length %d exceeds max batch size %d", len(msg.Entries), types.MaxIssueBatchSize)
	}

	for i, entry := range msg.Entries {
		if _, err := sdk.AccAddressFromBech32(entry.Holder); err != nil {
			return nil, fmt.Errorf("entry %d: invalid holder address %q: %w", i, entry.Holder, err)
		}
		if err := types.ValidateDates(entry.StartDate, entry.EndDate); err != nil {
			return nil, fmt.Errorf("entry %d: %w", i, err)
		}
		if entry.Count == 0 {
			return nil, errorsmod.Wrapf(types.ErrInvalidCount, "entry %d: count must be greater than zero", i)
		}

		hasPerm, err := ms.k.hasAdminPermission(ctx, msg.Issuer, entry.LicenseTypeId, "issue")
		if err != nil {
			return nil, err
		}
		if !hasPerm {
			return nil, errorsmod.Wrapf(types.ErrUnauthorized, "%s does not have issue permission for license type %s", msg.Issuer, entry.LicenseTypeId)
		}
	}

	// Check supply caps up front, aggregating requested counts per license
	// type, so no licenses are issued if any entry would exceed a cap.
	totals := make(map[string]math.Int)
	for i, entry := range msg.Entries {
		lt, err := ms.k.LicenseTypes.Get(ctx, entry.LicenseTypeId)
		if err != nil {
			return nil, errorsmod.Wrapf(types.ErrLicenseTypeNotFound, "entry %d: license type %s not found", i, entry.LicenseTypeId)
		}

		total, ok := totals[entry.LicenseTypeId]
		if !ok {
			total = math.ZeroInt()
		}
		total = total.Add(math.NewIntFromUint64(entry.Count))
		totals[entry.LicenseTypeId] = total

		if !lt.MaxSupply.IsZero() && lt.IssuedCount.Add(total).GT(lt.MaxSupply) {
			return nil, errorsmod.Wrapf(types.ErrMaxSupplyReached, "entry %d: license type %s: issuing %d would exceed max supply of %s (current: %s)", i, entry.LicenseTypeId, entry.Count, lt.MaxSupply.String(), lt.IssuedCount.String())
		}
	}

	ids := make([]uint64, 0, len(msg.Entries))
	for _, entry := range msg.Entries {
		// Re-read the license type each entry so counts accumulate correctly
		// when multiple entries reference the same type.
		lt, err := ms.k.LicenseTypes.Get(ctx, entry.LicenseTypeId)
		if err != nil {
			return nil, err
		}
		countInt := math.NewIntFromUint64(entry.Count)

		for j := uint64(0); j < entry.Count; j++ {
			id, err := ms.k.nextLicenseID(ctx, entry.LicenseTypeId)
			if err != nil {
				return nil, err
			}

			license := types.License{
				Id:        id,
				Type:      entry.LicenseTypeId,
				Holder:    entry.Holder,
				StartDate: entry.StartDate,
				EndDate:   entry.EndDate,
				Status:    "active",
			}

			if err := ms.k.Licenses.Set(ctx, collections.Join(entry.LicenseTypeId, id), license); err != nil {
				return nil, err
			}
			if err := ms.k.ActiveLicensesByHolder.Set(ctx, collections.Join3(entry.Holder, entry.LicenseTypeId, id)); err != nil {
				return nil, err
			}

			ids = append(ids, id)
		}

		lt.IssuedCount = lt.IssuedCount.Add(countInt)
		lt.ActiveCount = lt.ActiveCount.Add(countInt)
		if err := ms.k.LicenseTypes.Set(ctx, entry.LicenseTypeId, lt); err != nil {
			return nil, err
		}

		sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeIssueLicenses,
			sdk.NewAttribute(types.AttributeKeyLicenseTypeID, entry.LicenseTypeId),
			sdk.NewAttribute(types.AttributeKeyHolder, entry.Holder),
			sdk.NewAttribute("count", fmt.Sprintf("%d", entry.Count)),
		))
	}

	return &types.MsgIssueLicensesResponse{Ids: ids}, nil
}

func (ms msgServer) RevokeLicenses(ctx context.Context, msg *types.MsgRevokeLicenses) (*types.MsgRevokeLicensesResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	hasPerm, err := ms.k.hasAdminPermission(ctx, msg.Revoker, msg.LicenseTypeId, "revoke")
	if err != nil {
		return nil, err
	}
	if !hasPerm {
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "%s does not have revoke permission for license type %s", msg.Revoker, msg.LicenseTypeId)
	}

	if msg.Count == 0 {
		return nil, errorsmod.Wrap(types.ErrInvalidCount, "count must be greater than zero")
	}
	count := msg.Count

	// Walk ActiveLicensesByHolder in descending id order so we collect the
	// most recently issued licenses first, and stop as soon as we have enough.
	// The index holds active licenses only, so no per-entry status check is
	// needed.
	rng := collections.NewSuperPrefixedTripleRangeReversed[string, string, uint64](msg.Holder, msg.LicenseTypeId)
	activeIDs := make([]uint64, 0, count)
	err = ms.k.ActiveLicensesByHolder.Walk(ctx, rng, func(key collections.Triple[string, string, uint64]) (bool, error) {
		activeIDs = append(activeIDs, key.K3())
		return uint64(len(activeIDs)) >= count, nil
	})
	if err != nil {
		return nil, err
	}

	if uint64(len(activeIDs)) < count {
		return nil, errorsmod.Wrapf(types.ErrLicenseNotFound, "holder %s has %d active license(s) of type %s, but %d requested", msg.Holder, len(activeIDs), msg.LicenseTypeId, count)
	}

	endDate := sdkCtx.BlockTime().Format("2006-01-02")
	revokedIDs := make([]uint64, 0, count)

	for _, id := range activeIDs {
		license, err := ms.k.Licenses.Get(ctx, collections.Join(msg.LicenseTypeId, id))
		if err != nil {
			return nil, err
		}

		license.Status = "revoked"
		license.EndDate = endDate

		if err := ms.k.Licenses.Set(ctx, collections.Join(msg.LicenseTypeId, id), license); err != nil {
			return nil, err
		}
		if err := ms.k.ActiveLicensesByHolder.Remove(ctx, collections.Join3(msg.Holder, msg.LicenseTypeId, id)); err != nil {
			return nil, err
		}

		revokedIDs = append(revokedIDs, id)
	}

	lt, err := ms.k.LicenseTypes.Get(ctx, msg.LicenseTypeId)
	if err != nil {
		return nil, err
	}
	countInt := math.NewIntFromUint64(count)
	lt.ActiveCount = lt.ActiveCount.Sub(countInt)
	lt.RevokedCount = lt.RevokedCount.Add(countInt)
	if err := ms.k.LicenseTypes.Set(ctx, msg.LicenseTypeId, lt); err != nil {
		return nil, err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeRevokeLicenses,
		sdk.NewAttribute(types.AttributeKeyLicenseTypeID, msg.LicenseTypeId),
		sdk.NewAttribute(types.AttributeKeyHolder, msg.Holder),
		sdk.NewAttribute("count", fmt.Sprintf("%d", count)),
	))

	return &types.MsgRevokeLicensesResponse{Ids: revokedIDs}, nil
}

func (ms msgServer) TransferLicense(ctx context.Context, msg *types.MsgTransferLicense) (*types.MsgTransferLicenseResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if _, err := sdk.AccAddressFromBech32(msg.Recipient); err != nil {
		return nil, fmt.Errorf("invalid recipient address %q: %w", msg.Recipient, err)
	}

	if msg.Holder == msg.Recipient {
		return nil, fmt.Errorf("cannot transfer license to the current holder")
	}

	license, err := ms.k.Licenses.Get(ctx, collections.Join(msg.LicenseTypeId, msg.Id))
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLicenseNotFound, "license (type=%s, id=%d) not found", msg.LicenseTypeId, msg.Id)
	}

	if license.Holder != msg.Holder {
		return nil, errorsmod.Wrapf(types.ErrNotLicenseHolder, "signer %s is not the holder of license (type=%s, id=%d)", msg.Holder, msg.LicenseTypeId, msg.Id)
	}

	if license.Status != "active" {
		return nil, errorsmod.Wrapf(types.ErrLicenseRevoked, "license (type=%s, id=%d) is %s and cannot be transferred", msg.LicenseTypeId, msg.Id, license.Status)
	}

	lt, err := ms.k.LicenseTypes.Get(ctx, license.Type)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLicenseTypeNotFound, "license type %s not found", license.Type)
	}
	if !lt.Transferrable {
		return nil, errorsmod.Wrapf(types.ErrLicenseNotTransferable, "license type %s is not transferrable", license.Type)
	}

	// Remove old holder index
	if err := ms.k.ActiveLicensesByHolder.Remove(ctx, collections.Join3(license.Holder, msg.LicenseTypeId, msg.Id)); err != nil {
		return nil, err
	}

	license.Holder = msg.Recipient

	if err := ms.k.Licenses.Set(ctx, collections.Join(msg.LicenseTypeId, msg.Id), license); err != nil {
		return nil, err
	}

	// Add new holder index
	if err := ms.k.ActiveLicensesByHolder.Set(ctx, collections.Join3(msg.Recipient, msg.LicenseTypeId, msg.Id)); err != nil {
		return nil, err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeTransferLicense,
		sdk.NewAttribute(types.AttributeKeyLicenseTypeID, msg.LicenseTypeId),
		sdk.NewAttribute(types.AttributeKeyLicenseID, fmt.Sprintf("%d", msg.Id)),
		sdk.NewAttribute(types.AttributeKeyHolder, msg.Holder),
		sdk.NewAttribute(types.AttributeKeyRecipient, msg.Recipient),
	))

	return &types.MsgTransferLicenseResponse{}, nil
}
