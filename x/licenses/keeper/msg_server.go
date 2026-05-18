package keeper

import (
	"context"
	"errors"
	"fmt"
	"sort"
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

	for _, grant := range msg.Grants {
		if !types.IsValidPermission(grant.Permission) {
			return nil, fmt.Errorf("invalid permission %q: must be one of issue, revoke, update", grant.Permission)
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

	existing, err := ms.k.AdminKeys.Get(ctx, msg.Address)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return nil, err
	}

	permToTypes := make(map[string]map[string]struct{})
	addGrants := func(grants []types.AdminKeyGrant) {
		for _, g := range grants {
			set, ok := permToTypes[g.Permission]
			if !ok {
				set = make(map[string]struct{})
				permToTypes[g.Permission] = set
			}
			for _, lt := range g.LicenseTypes {
				set[lt] = struct{}{}
			}
		}
	}
	addGrants(existing.Grants)
	addGrants(msg.Grants)

	ak := types.AdminKey{
		Address: msg.Address,
		Grants:  sortedGrants(permToTypes),
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
		types.EventTypeGrantAdminPermissions,
		sdk.NewAttribute(types.AttributeKeyAddress, msg.Address),
		sdk.NewAttribute(types.AttributeKeyPermissions, strings.Join(perms, ",")),
		sdk.NewAttribute(types.AttributeKeyGrantTypes, strings.Join(grantTypes, ";")),
	))

	return &types.MsgGrantAdminPermissionsResponse{}, nil
}

// RevokeAdminKeyPermissions removes specific (license type, permission) pairs from
// an admin key. Pairs that are not currently present are silently ignored — the
// caller can safely re-send the same revoke. A grant whose LicenseTypes list becomes
// empty is dropped; if no grants remain the AdminKey entry itself is deleted.
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

	existing, err := ms.k.AdminKeys.Get(ctx, msg.Address)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			// No grants exist for this address; revoke is a no-op.
			sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
				types.EventTypeRevokeAdminKeyPermissions,
				sdk.NewAttribute(types.AttributeKeyAddress, msg.Address),
			))
			return &types.MsgRevokeAdminKeyPermissionsResponse{}, nil
		}
		return nil, err
	}

	permToTypes := make(map[string]map[string]struct{})
	for _, g := range existing.Grants {
		set := make(map[string]struct{}, len(g.LicenseTypes))
		for _, lt := range g.LicenseTypes {
			set[lt] = struct{}{}
		}
		permToTypes[g.Permission] = set
	}

	for _, p := range msg.Permissions {
		if set, ok := permToTypes[p.Permission]; ok {
			delete(set, p.LicenseTypeId)
			if len(set) == 0 {
				delete(permToTypes, p.Permission)
			}
		}
	}

	if len(permToTypes) == 0 {
		if err := ms.k.AdminKeys.Remove(ctx, msg.Address); err != nil {
			return nil, err
		}
	} else {
		ak := types.AdminKey{
			Address: msg.Address,
			Grants:  sortedGrants(permToTypes),
		}
		if err := ms.k.AdminKeys.Set(ctx, msg.Address, ak); err != nil {
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

// sortedGrants flattens a (permission -> {license_type}) map into a deterministically
// ordered slice of AdminKeyGrants: permissions ascending, license types ascending
// within each grant.
func sortedGrants(permToTypes map[string]map[string]struct{}) []types.AdminKeyGrant {
	perms := make([]string, 0, len(permToTypes))
	for p := range permToTypes {
		perms = append(perms, p)
	}
	sort.Strings(perms)

	out := make([]types.AdminKeyGrant, 0, len(perms))
	for _, p := range perms {
		set := permToTypes[p]
		lts := make([]string, 0, len(set))
		for lt := range set {
			lts = append(lts, lt)
		}
		sort.Strings(lts)
		out = append(out, types.AdminKeyGrant{Permission: p, LicenseTypes: lts})
	}
	return out
}

func (ms msgServer) IssueLicense(ctx context.Context, msg *types.MsgIssueLicense) (*types.MsgIssueLicenseResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if _, err := sdk.AccAddressFromBech32(msg.Holder); err != nil {
		return nil, fmt.Errorf("invalid holder address %q: %w", msg.Holder, err)
	}

	if err := types.ValidateDates(msg.StartDate, msg.EndDate); err != nil {
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

	countInt := math.NewIntFromUint64(count)
	if !lt.MaxSupply.IsZero() && lt.IssuedCount.Add(countInt).GT(lt.MaxSupply) {
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

	lt.IssuedCount = lt.IssuedCount.Add(countInt)
	lt.ActiveCount = lt.ActiveCount.Add(countInt)
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

	if hasPerm, _ := ms.k.hasAdminPermission(ctx, msg.Revoker, msg.LicenseTypeId, "revoke"); !hasPerm {
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "%s does not have revoke permission for license type %s", msg.Revoker, msg.LicenseTypeId)
	}

	count := msg.Count
	if count == 0 {
		count = 1
	}

	// Collect active license IDs for this holder+type.
	rng := collections.NewSuperPrefixedTripleRange[string, string, uint64](msg.Holder, msg.LicenseTypeId)
	var activeIDs []uint64
	err := ms.k.LicenseByHolder.Walk(ctx, rng, func(key collections.Triple[string, string, uint64], _ uint64) (bool, error) {
		license, err := ms.k.Licenses.Get(ctx, collections.Join(key.K2(), key.K3()))
		if err != nil {
			return true, err
		}
		if license.Status == "active" {
			activeIDs = append(activeIDs, key.K3())
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	if uint64(len(activeIDs)) < count {
		return nil, errorsmod.Wrapf(types.ErrLicenseNotFound, "holder %s has %d active license(s) of type %s, but %d requested", msg.Holder, len(activeIDs), msg.LicenseTypeId, count)
	}

	// Sort descending so we revoke the most recently issued first.
	sort.Slice(activeIDs, func(i, j int) bool { return activeIDs[i] > activeIDs[j] })

	endDate := sdkCtx.BlockTime().Format("2006-01-02")
	revokedIDs := make([]uint64, 0, count)

	for _, id := range activeIDs[:count] {
		license, err := ms.k.Licenses.Get(ctx, collections.Join(msg.LicenseTypeId, id))
		if err != nil {
			return nil, err
		}

		license.Status = "revoked"
		license.EndDate = endDate

		if err := ms.k.Licenses.Set(ctx, collections.Join(msg.LicenseTypeId, id), license); err != nil {
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
		types.EventTypeRevokeLicense,
		sdk.NewAttribute(types.AttributeKeyLicenseTypeID, msg.LicenseTypeId),
		sdk.NewAttribute(types.AttributeKeyHolder, msg.Holder),
		sdk.NewAttribute("count", fmt.Sprintf("%d", count)),
	))

	return &types.MsgRevokeLicenseResponse{Ids: revokedIDs}, nil
}

func (ms msgServer) UpdateLicense(_ context.Context, _ *types.MsgUpdateLicense) (*types.MsgUpdateLicenseResponse, error) {
	return nil, fmt.Errorf("UpdateLicense is not supported")
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

	if len(msg.Entries) > types.MaxIssueBatchSize {
		return nil, fmt.Errorf("entries length %d exceeds max batch size %d", len(msg.Entries), types.MaxIssueBatchSize)
	}

	if hasPerm, _ := ms.k.hasAdminPermission(ctx, msg.Issuer, msg.LicenseTypeId, "issue"); !hasPerm {
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "%s does not have issue permission for license type %s", msg.Issuer, msg.LicenseTypeId)
	}

	lt, err := ms.k.LicenseTypes.Get(ctx, msg.LicenseTypeId)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLicenseTypeNotFound, "license type %s not found", msg.LicenseTypeId)
	}

	count := uint64(len(msg.Entries))
	countInt := math.NewIntFromUint64(count)
	if !lt.MaxSupply.IsZero() && lt.IssuedCount.Add(countInt).GT(lt.MaxSupply) {
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
		if err := types.ValidateDates(entry.StartDate, entry.EndDate); err != nil {
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

	lt.IssuedCount = lt.IssuedCount.Add(countInt)
	lt.ActiveCount = lt.ActiveCount.Add(countInt)
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

