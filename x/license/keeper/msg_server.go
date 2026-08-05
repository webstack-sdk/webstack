package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/webstack-sdk/webstack/x/license/types"
)

type msgServer struct {
	k Keeper
}

var _ types.MsgServer = msgServer{}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{k: keeper}
}

// CreateLicenseType registers a new license type. The signer must hold the
// module-wide "type.create" grant. Namespace ownership alone does not
// authorize creation: as with "issue" and "revoke", the owner's role is to
// grant the right, including to itself, not to exercise it implicitly.
func (ms msgServer) CreateLicenseType(ctx context.Context, msg *types.MsgCreateLicenseType) (*types.MsgCreateLicenseTypeResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// The empty scope is the only key form a module-wide grant is stored
	// under; x/permission rejects any other at grant time.
	hasPerm, err := ms.k.hasPermission(ctx, msg.Creator, types.PermissionCreateType, "")
	if err != nil {
		return nil, err
	}
	if !hasPerm {
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "%s does not have %s permission", msg.Creator, types.PermissionCreateType)
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

	// The signer is not recorded: the "type.create" grant is the whole
	// authorization, and downstream modules gate on their own grants rather
	// than on who created a license type.
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
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "signer %s is not the license namespace owner", msg.Owner)
	}

	lt, err := ms.k.LicenseTypes.Get(ctx, msg.Id)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrLicenseTypeNotFound, "license type %s not found", msg.Id)
	}

	// Only transferrability is updatable; max_supply is fixed at creation.
	lt.Transferrable = msg.Transferrable

	if err := ms.k.LicenseTypes.Set(ctx, msg.Id, lt); err != nil {
		return nil, err
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeUpdateLicenseType,
		sdk.NewAttribute(types.AttributeKeyLicenseTypeID, msg.Id),
	))

	return &types.MsgUpdateLicenseTypeResponse{}, nil
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

		hasPerm, err := ms.k.hasPermission(ctx, msg.Issuer, types.PermissionIssue, entry.LicenseTypeId)
		if err != nil {
			return nil, err
		}
		if !hasPerm {
			return nil, errorsmod.Wrapf(types.ErrUnauthorized, "%s does not have issue permission for license type %s", msg.Issuer, entry.LicenseTypeId)
		}
	}

	// Check supply caps up front, aggregating requested counts per license
	// type, so no licenses are issued if any entry would exceed a cap.
	//
	// max_supply bounds licenses *outstanding*, not licenses ever issued: the
	// check is against active_count, which revocation decrements. Revoking
	// therefore returns a slot to the pool, which is what a chargeback on a
	// purchased license needs — the sale is undone, so it must stop consuming
	// supply. issued_count keeps counting lifetime issuance for audit and can
	// exceed max_supply over time; it is deliberately not the cap.
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

		if !lt.MaxSupply.IsZero() && lt.ActiveCount.Add(total).GT(lt.MaxSupply) {
			return nil, errorsmod.Wrapf(types.ErrMaxSupplyReached, "entry %d: license type %s: issuing %d would exceed max supply of %s (outstanding: %s)", i, entry.LicenseTypeId, entry.Count, lt.MaxSupply.String(), lt.ActiveCount.String())
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

		// Issuance creates the holder's account if absent, so a wallet
		// holding only a license can sign its first transaction.
		holderAddr, err := sdk.AccAddressFromBech32(entry.Holder)
		if err != nil {
			return nil, fmt.Errorf("invalid holder address %q: %w", entry.Holder, err)
		}
		ms.k.createAccountIfNotExists(ctx, holderAddr)

		for j := uint64(0); j < entry.Count; j++ {
			id, err := ms.k.nextLicenseID(ctx)
			if err != nil {
				return nil, err
			}

			license := types.License{
				Id:        id,
				Type:      entry.LicenseTypeId,
				Holder:    entry.Holder,
				StartDate: entry.StartDate,
				EndDate:   entry.EndDate,
				Status:    types.StatusActive,
			}

			if err := ms.k.Licenses.Set(ctx, id, license); err != nil {
				return nil, err
			}
			// Written once here and never moved: a license's type is fixed,
			// and this index covers revoked licenses too.
			if err := ms.k.LicensesByType.Set(ctx, collections.Join(entry.LicenseTypeId, id)); err != nil {
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

	hasPerm, err := ms.k.hasPermission(ctx, msg.Revoker, types.PermissionRevoke, msg.LicenseTypeId)
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
	// Do not preallocate from count: it is raw user input and may vastly
	// exceed the holder's actual holdings.
	var activeIDs []uint64
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

	revokedDate := sdkCtx.BlockTime().Format("2006-01-02")
	revokedIDs := make([]uint64, 0, count)

	for _, id := range activeIDs {
		license, err := ms.k.Licenses.Get(ctx, id)
		if err != nil {
			return nil, err
		}

		// EndDate keeps its issued value; the revocation date is recorded
		// separately.
		license.Status = types.StatusRevoked
		license.RevokedDate = revokedDate

		// LicensesByType is left alone: it lists revoked licenses too.
		if err := ms.k.Licenses.Set(ctx, id, license); err != nil {
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
