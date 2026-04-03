package keeper

import (
	"context"
	"fmt"
	"strings"
	"time"

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

func isValidPermission(p string) bool {
	for _, vp := range types.Permissions {
		if vp == p {
			return true
		}
	}
	return false
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

	_, err = ms.k.LicenseTypes.Get(ctx, msg.Id)
	if err == nil {
		return nil, errorsmod.Wrapf(types.ErrLicenseTypeExists, "license type %s already exists", msg.Id)
	}

	lt := types.LicenseType{
		Id:            msg.Id,
		Transferrable: msg.Transferrable,
		MaxSupply:     msg.MaxSupply,
		IssuedCount:   math.ZeroInt(),
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

func (ms msgServer) SetAdminKey(ctx context.Context, msg *types.MsgSetAdminKey) (*types.MsgSetAdminKeyResponse, error) {
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

	for _, grant := range msg.Grants {
		if !isValidPermission(grant.Permission) {
			return nil, fmt.Errorf("invalid permission %q: must be one of issue, revoke, update", grant.Permission)
		}
		if len(grant.LicenseTypes) == 0 {
			return nil, fmt.Errorf("grant for permission %q must include at least one license type", grant.Permission)
		}
	}

	ak := types.AdminKey{
		Address: msg.Address,
		Grants:  msg.Grants,
	}

	if err := ms.k.AdminKeys.Set(ctx, msg.Address, ak); err != nil {
		return nil, err
	}

	var perms []string
	var grantTypes []string
	for _, grant := range msg.Grants {
		perms = append(perms, grant.Permission)
		grantTypes = append(grantTypes, strings.Join(grant.LicenseTypes, ","))
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeSetAdminKey,
		sdk.NewAttribute(types.AttributeKeyAddress, msg.Address),
		sdk.NewAttribute(types.AttributeKeyPermissions, strings.Join(perms, ",")),
		sdk.NewAttribute(types.AttributeKeyGrantTypes, strings.Join(grantTypes, ";")),
	))

	return &types.MsgSetAdminKeyResponse{}, nil
}

func (ms msgServer) RemoveAdminKey(ctx context.Context, msg *types.MsgRemoveAdminKey) (*types.MsgRemoveAdminKeyResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	isOwner, err := ms.k.isOwner(ctx, msg.Owner)
	if err != nil {
		return nil, err
	}
	if !isOwner {
		params, _ := ms.k.Params.Get(ctx)
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "signer %s is not the module owner %s", msg.Owner, params.Owner)
	}

	if err := ms.k.AdminKeys.Remove(ctx, msg.Address); err != nil {
		return nil, err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeRemoveAdminKey,
		sdk.NewAttribute(types.AttributeKeyAddress, msg.Address),
	))

	return &types.MsgRemoveAdminKeyResponse{}, nil
}

func (ms msgServer) IssueLicense(ctx context.Context, msg *types.MsgIssueLicense) (*types.MsgIssueLicenseResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if _, err := sdk.AccAddressFromBech32(msg.Holder); err != nil {
		return nil, fmt.Errorf("invalid holder address %q: %w", msg.Holder, err)
	}

	if err := validateDates(msg.StartDate, msg.EndDate); err != nil {
		return nil, err
	}

	if hasPerm, _ := ms.k.hasAdminPermission(ctx, msg.Issuer, msg.LicenseTypeId, "issue"); !hasPerm {
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "%s does not have issue permission for license type %s", msg.Issuer, msg.LicenseTypeId)
	}

	lt, err := ms.k.LicenseTypes.Get(ctx, msg.LicenseTypeId)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLicenseTypeNotFound, "license type %s not found", msg.LicenseTypeId)
	}

	count := msg.Count
	if count == 0 {
		count = 1
	}

	if !lt.MaxSupply.IsZero() && lt.IssuedCount.AddRaw(int64(count)).GT(lt.MaxSupply) {
		return nil, errorsmod.Wrapf(types.ErrMaxSupplyReached, "license type %s: issuing %d would exceed max supply of %s (current: %s)", msg.LicenseTypeId, count, lt.MaxSupply.String(), lt.IssuedCount.String())
	}

	ids := make([]uint64, 0, count)
	for i := uint64(0); i < count; i++ {
		id, err := ms.k.nextLicenseID(ctx, msg.LicenseTypeId)
		if err != nil {
			return nil, err
		}

		license := types.License{
			Id:        id,
			Type:      msg.LicenseTypeId,
			Holder:    msg.Holder,
			StartDate: msg.StartDate,
			EndDate:   msg.EndDate,
			Status:    "active",
		}

		if err := ms.k.Licenses.Set(ctx, collections.Join(msg.LicenseTypeId, id), license); err != nil {
			return nil, err
		}
		if err := ms.k.LicenseByHolder.Set(ctx, collections.Join3(msg.Holder, msg.LicenseTypeId, id), id); err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	lt.IssuedCount = lt.IssuedCount.AddRaw(int64(count))
	if err := ms.k.LicenseTypes.Set(ctx, msg.LicenseTypeId, lt); err != nil {
		return nil, err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeIssueLicense,
		sdk.NewAttribute(types.AttributeKeyLicenseTypeID, msg.LicenseTypeId),
		sdk.NewAttribute(types.AttributeKeyHolder, msg.Holder),
		sdk.NewAttribute("count", fmt.Sprintf("%d", count)),
	))

	return &types.MsgIssueLicenseResponse{Ids: ids}, nil
}

func (ms msgServer) RevokeLicense(ctx context.Context, msg *types.MsgRevokeLicense) (*types.MsgRevokeLicenseResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	license, err := ms.k.Licenses.Get(ctx, collections.Join(msg.LicenseTypeId, msg.Id))
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLicenseNotFound, "license (type=%s, id=%d) not found", msg.LicenseTypeId, msg.Id)
	}

	if hasPerm, _ := ms.k.hasAdminPermission(ctx, msg.Revoker, license.Type, "revoke"); !hasPerm {
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "%s does not have revoke permission for license type %s", msg.Revoker, license.Type)
	}

	// Remove holder index
	if err := ms.k.LicenseByHolder.Remove(ctx, collections.Join3(license.Holder, msg.LicenseTypeId, msg.Id)); err != nil {
		return nil, err
	}

	// Remove license
	if err := ms.k.Licenses.Remove(ctx, collections.Join(msg.LicenseTypeId, msg.Id)); err != nil {
		return nil, err
	}

	// Decrement issued count
	lt, err := ms.k.LicenseTypes.Get(ctx, license.Type)
	if err == nil && !lt.IssuedCount.IsZero() {
		lt.IssuedCount = lt.IssuedCount.SubRaw(1)
		_ = ms.k.LicenseTypes.Set(ctx, license.Type, lt)
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeRevokeLicense,
		sdk.NewAttribute(types.AttributeKeyLicenseTypeID, msg.LicenseTypeId),
		sdk.NewAttribute(types.AttributeKeyLicenseID, fmt.Sprintf("%d", msg.Id)),
	))

	return &types.MsgRevokeLicenseResponse{}, nil
}

func (ms msgServer) UpdateLicense(ctx context.Context, msg *types.MsgUpdateLicense) (*types.MsgUpdateLicenseResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	license, err := ms.k.Licenses.Get(ctx, collections.Join(msg.LicenseTypeId, msg.Id))
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLicenseNotFound, "license (type=%s, id=%d) not found", msg.LicenseTypeId, msg.Id)
	}

	if hasPerm, _ := ms.k.hasAdminPermission(ctx, msg.Updater, license.Type, "update"); !hasPerm {
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "%s does not have update permission for license type %s", msg.Updater, license.Type)
	}

	if msg.Status != "active" && msg.Status != "revoked" {
		return nil, fmt.Errorf("invalid status %q: must be \"active\" or \"revoked\"", msg.Status)
	}

	license.Status = msg.Status

	if err := ms.k.Licenses.Set(ctx, collections.Join(msg.LicenseTypeId, msg.Id), license); err != nil {
		return nil, err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeUpdateLicense,
		sdk.NewAttribute(types.AttributeKeyLicenseTypeID, msg.LicenseTypeId),
		sdk.NewAttribute(types.AttributeKeyLicenseID, fmt.Sprintf("%d", msg.Id)),
		sdk.NewAttribute(types.AttributeKeyStatus, msg.Status),
	))

	return &types.MsgUpdateLicenseResponse{}, nil
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

	lt, err := ms.k.LicenseTypes.Get(ctx, license.Type)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLicenseTypeNotFound, "license type %s not found", license.Type)
	}
	if !lt.Transferrable {
		return nil, errorsmod.Wrapf(types.ErrLicenseNotTransferable, "license type %s is not transferrable", license.Type)
	}

	// Remove old holder index
	if err := ms.k.LicenseByHolder.Remove(ctx, collections.Join3(license.Holder, msg.LicenseTypeId, msg.Id)); err != nil {
		return nil, err
	}

	license.Holder = msg.Recipient

	if err := ms.k.Licenses.Set(ctx, collections.Join(msg.LicenseTypeId, msg.Id), license); err != nil {
		return nil, err
	}

	// Add new holder index
	if err := ms.k.LicenseByHolder.Set(ctx, collections.Join3(msg.Recipient, msg.LicenseTypeId, msg.Id), msg.Id); err != nil {
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

func (ms msgServer) BatchIssueLicense(ctx context.Context, msg *types.MsgBatchIssueLicense) (*types.MsgBatchIssueLicenseResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if len(msg.Entries) == 0 {
		return nil, fmt.Errorf("entries must not be empty")
	}

	if hasPerm, _ := ms.k.hasAdminPermission(ctx, msg.Issuer, msg.LicenseTypeId, "issue"); !hasPerm {
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "%s does not have issue permission for license type %s", msg.Issuer, msg.LicenseTypeId)
	}

	lt, err := ms.k.LicenseTypes.Get(ctx, msg.LicenseTypeId)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLicenseTypeNotFound, "license type %s not found", msg.LicenseTypeId)
	}

	count := int64(len(msg.Entries))
	if !lt.MaxSupply.IsZero() && lt.IssuedCount.AddRaw(count).GT(lt.MaxSupply) {
		return nil, errorsmod.Wrapf(types.ErrMaxSupplyReached, "license type %s: issuing %d would exceed max supply of %s (current: %s)", msg.LicenseTypeId, count, lt.MaxSupply.String(), lt.IssuedCount.String())
	}

	// Validate all entries before issuing any
	for i, entry := range msg.Entries {
		if _, err := sdk.AccAddressFromBech32(entry.Holder); err != nil {
			return nil, fmt.Errorf("entry %d: invalid holder address %q: %w", i, entry.Holder, err)
		}
		if entry.StartDate == "" {
			return nil, fmt.Errorf("entry %d: start_date is required", i)
		}
		if err := validateDates(entry.StartDate, entry.EndDate); err != nil {
			return nil, fmt.Errorf("entry %d: %w", i, err)
		}
	}

	ids := make([]uint64, 0, count)
	for _, entry := range msg.Entries {
		id, err := ms.k.nextLicenseID(ctx, msg.LicenseTypeId)
		if err != nil {
			return nil, err
		}

		license := types.License{
			Id:        id,
			Type:      msg.LicenseTypeId,
			Holder:    entry.Holder,
			StartDate: entry.StartDate,
			EndDate:   entry.EndDate,
			Status:    "active",
		}

		if err := ms.k.Licenses.Set(ctx, collections.Join(msg.LicenseTypeId, id), license); err != nil {
			return nil, err
		}
		if err := ms.k.LicenseByHolder.Set(ctx, collections.Join3(entry.Holder, msg.LicenseTypeId, id), id); err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	lt.IssuedCount = lt.IssuedCount.AddRaw(count)
	if err := ms.k.LicenseTypes.Set(ctx, msg.LicenseTypeId, lt); err != nil {
		return nil, err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeBatchIssueLicense,
		sdk.NewAttribute(types.AttributeKeyLicenseTypeID, msg.LicenseTypeId),
		sdk.NewAttribute("count", fmt.Sprintf("%d", count)),
	))

	return &types.MsgBatchIssueLicenseResponse{Ids: ids}, nil
}

// validateDates validates start_date and end_date in YYYY-MM-DD format.
func validateDates(startDate, endDate string) error {
	if startDate == "" {
		return fmt.Errorf("start_date is required")
	}
	sd, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return fmt.Errorf("invalid start_date %q: must be YYYY-MM-DD format", startDate)
	}
	if endDate != "" {
		ed, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			return fmt.Errorf("invalid end_date %q: must be YYYY-MM-DD format", endDate)
		}
		if ed.Before(sd) {
			return fmt.Errorf("end_date %s must not be before start_date %s", endDate, startDate)
		}
	}
	return nil
}
