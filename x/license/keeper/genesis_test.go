package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	keepertest "github.com/webstack-sdk/webstack/testutil/keeper"
	"github.com/webstack-sdk/webstack/testutil/sample"
	"github.com/webstack-sdk/webstack/x/license/keeper"
	"github.com/webstack-sdk/webstack/x/license/types"
)

// TestInitGenesisRunsFullValidation verifies that Keeper.InitGenesis itself
// runs GenesisState.Validate — so direct keeper callers (tests, future
// migrations) get the same invariant enforcement as the JSON path through
// AppModule.ValidateGenesis.
func TestInitGenesisRunsFullValidation(t *testing.T) {
	k, ctx := keepertest.LicenseKeeper(t)

	bad := &types.GenesisState{
		LicenseTypes: []types.LicenseType{
			{
				Id:           "neg",
				MaxSupply:    math.NewInt(-1),
				IssuedCount:  math.ZeroInt(),
				ActiveCount:  math.ZeroInt(),
				RevokedCount: math.ZeroInt(),
			},
		},
	}

	err := k.InitGenesis(ctx, bad)
	require.Error(t, err)
	require.Contains(t, err.Error(), "max_supply must not be negative")
}

// TestGenesisRoundTripActiveIndex verifies that the holder index is rebuilt
// for active licenses only, and that revoked licenses keep their status and
// revoked_date through an export/import cycle.
func TestGenesisRoundTripActiveIndex(t *testing.T) {
	src := keepertest.NewLicenseFixture(t)
	// Revocation stamps revoked_date with the block date; use a realistic
	// block time so the exported dates are meaningful.
	srcCtx := src.Ctx.WithBlockTime(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	owner := src.Owner
	holder := sample.AccAddress()
	ms := keeper.NewMsgServerImpl(src.Keeper)

	_, err := ms.CreateLicenseType(srcCtx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "node", MaxSupply: math.ZeroInt(),
	})
	require.NoError(t, err)
	src.Grant(t, owner, types.PermissionIssue, "node")
	src.Grant(t, owner, types.PermissionRevoke, "node")

	resp, err := ms.IssueLicenses(srcCtx, &types.MsgIssueLicenses{
		Issuer: owner, Entries: []types.IssueLicenseEntry{
			{LicenseTypeId: "node", Holder: holder, StartDate: "2026-01-01", Count: 3},
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.Ids, 3)

	revResp, err := ms.RevokeLicenses(srcCtx, &types.MsgRevokeLicenses{
		Revoker: owner, LicenseTypeId: "node", Holder: holder, Count: 1,
	})
	require.NoError(t, err)
	require.Equal(t, []uint64{3}, revResp.Ids, "most recently issued license is revoked first")

	exported := src.Keeper.ExportGenesis(srcCtx)

	dst, dstCtx := keepertest.LicenseKeeper(t)
	require.NoError(t, dst.InitGenesis(dstCtx, exported))

	// The holder index is rebuilt for active licenses only.
	q := setupQuerier(dst)
	byHolder, err := q.LicensesByHolder(dstCtx, &types.QueryLicensesByHolderRequest{Holder: holder})
	require.NoError(t, err)
	require.Len(t, byHolder.Licenses, 2)

	// The revoked license itself survives with its status and revoked_date,
	// and its end_date is untouched by revocation.
	l, found, err := dst.GetLicense(dstCtx, "node", 3)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, types.StatusRevoked, l.Status)
	require.Equal(t, "2026-07-01", l.RevokedDate)
	require.Empty(t, l.EndDate)
}

// TestGenesisRoundTripPreservesExplicitCounter verifies that the id sequence
// is genesis state in its own right: a counter deliberately larger than
// issued_count must survive export/import unchanged instead of being
// re-derived from the stats counter.
func TestGenesisRoundTripPreservesExplicitCounter(t *testing.T) {
	src := keepertest.NewLicenseFixture(t)
	srcCtx := src.Ctx
	owner := src.Owner
	holder := sample.AccAddress()
	ms := keeper.NewMsgServerImpl(src.Keeper)

	_, err := ms.CreateLicenseType(srcCtx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "node", MaxSupply: math.ZeroInt(),
	})
	require.NoError(t, err)
	src.Grant(t, owner, types.PermissionIssue, "node")
	_, err = ms.IssueLicenses(srcCtx, &types.MsgIssueLicenses{
		Issuer: owner, Entries: []types.IssueLicenseEntry{
			{LicenseTypeId: "node", Holder: holder, StartDate: "2026-01-01", Count: 2},
		},
	})
	require.NoError(t, err)

	// Bump the sequence past issued_count (2) to simulate the concepts
	// diverging.
	require.NoError(t, src.Keeper.LicenseCounts.Set(srcCtx, "node", 10))

	exported := src.Keeper.ExportGenesis(srcCtx)
	require.Equal(t, []types.LicenseCount{{LicenseTypeId: "node", Count: 10}}, exported.LicenseCounts)

	dst := keepertest.NewLicenseFixture(t)
	require.NoError(t, dst.Keeper.InitGenesis(dst.Ctx, exported))
	// License grants live in the permission module, not license genesis; the
	// destination chain needs its own grant for the issuer.
	dst.Grant(t, owner, types.PermissionIssue, "node")

	// The next issued id continues from the explicit counter, not from
	// issued_count.
	dstMs := keeper.NewMsgServerImpl(dst.Keeper)
	resp, err := dstMs.IssueLicenses(dst.Ctx, &types.MsgIssueLicenses{
		Issuer: owner, Entries: []types.IssueLicenseEntry{
			{LicenseTypeId: "node", Holder: holder, StartDate: "2026-02-01", Count: 1},
		},
	})
	require.NoError(t, err)
	require.Equal(t, []uint64{11}, resp.Ids)
}

// TestGenesisRoundTripPreservesLicenseIDs covers the LicenseCounts genesis
// export/import path: after exporting and re-importing genesis into a fresh
// keeper, newly issued license IDs must not collide with pre-existing ones.
func TestGenesisRoundTripPreservesLicenseIDs(t *testing.T) {
	// Source keeper: create a license type, issue a handful of licenses,
	// then export the genesis state.
	src := keepertest.NewLicenseFixture(t)
	srcCtx := src.Ctx
	owner := src.Owner
	holder := sample.AccAddress()

	ms := keeper.NewMsgServerImpl(src.Keeper)

	_, err := ms.CreateLicenseType(srcCtx, &types.MsgCreateLicenseType{
		Owner:         owner,
		Id:            "node",
		Transferrable: false,
		MaxSupply:     math.NewInt(100),
	})
	require.NoError(t, err)

	src.Grant(t, owner, types.PermissionIssue, "node")

	resp, err := ms.IssueLicenses(srcCtx, &types.MsgIssueLicenses{
		Issuer: owner,
		Entries: []types.IssueLicenseEntry{
			{LicenseTypeId: "node", Holder: holder, StartDate: "2026-01-01", Count: 5},
		},
	})
	require.NoError(t, err)
	require.Equal(t, []uint64{1, 2, 3, 4, 5}, resp.Ids)

	exported := src.Keeper.ExportGenesis(srcCtx)

	// Destination keeper: a fresh store, then InitGenesis with the exported
	// state. The genesis must include the existing license type and its
	// counters, so the next issued ID is 6, not 1.
	dst := keepertest.NewLicenseFixture(t)
	dstCtx := dst.Ctx
	require.NoError(t, dst.Keeper.InitGenesis(dstCtx, exported))
	dst.Grant(t, owner, types.PermissionIssue, "node")

	// Sanity: the imported license type's IssuedCount survived.
	lt, found, err := dst.Keeper.GetLicenseType(dstCtx, "node")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, math.NewInt(5), lt.IssuedCount, "IssuedCount must survive genesis import")

	// Sanity: the imported licenses survived.
	for _, id := range resp.Ids {
		l, ok, err := dst.Keeper.GetLicense(dstCtx, "node", id)
		require.NoError(t, err)
		require.True(t, ok, "license id %d must exist after import", id)
		require.Equal(t, holder, l.Holder)
	}

	// The bug: nextLicenseID resets to 0 because LicenseCounts isn't
	// restored, so issuing returns id=1 and overwrites the existing one.
	dstMs := keeper.NewMsgServerImpl(dst.Keeper)
	newHolder := sample.AccAddress()
	issueResp, err := dstMs.IssueLicenses(dstCtx, &types.MsgIssueLicenses{
		Issuer: owner,
		Entries: []types.IssueLicenseEntry{
			{LicenseTypeId: "node", Holder: newHolder, StartDate: "2026-02-01", Count: 1},
		},
	})
	require.NoError(t, err)
	require.Len(t, issueResp.Ids, 1)

	newID := issueResp.Ids[0]
	require.Greater(t, newID, uint64(5), "new license id must not collide with imported ids")

	// And the imported holder's licenses must still all be theirs (i.e., the
	// new issuance didn't overwrite license 1).
	l1, ok, err := dst.Keeper.GetLicense(dstCtx, "node", 1)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, holder, l1.Holder, "imported license 1 must not have been overwritten")
}
