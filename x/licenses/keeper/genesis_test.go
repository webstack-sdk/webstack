package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	keepertest "github.com/webstack-sdk/webstack/testutil/keeper"
	"github.com/webstack-sdk/webstack/testutil/sample"
	"github.com/webstack-sdk/webstack/x/licenses/keeper"
	"github.com/webstack-sdk/webstack/x/licenses/types"
)

// TestInitGenesisRunsFullValidation verifies that Keeper.InitGenesis itself
// runs GenesisState.Validate — so direct keeper callers (tests, future
// migrations) get the same invariant enforcement as the JSON path through
// AppModule.ValidateGenesis.
func TestInitGenesisRunsFullValidation(t *testing.T) {
	k, ctx := keepertest.LicensesKeeper(t)
	owner := k.GetParams(ctx).Owner

	bad := &types.GenesisState{
		Params: types.Params{Owner: owner},
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

// TestGenesisRoundTripPreservesLicenseIDs covers the LicenseCounts genesis
// export/import path: after exporting and re-importing genesis into a fresh
// keeper, newly issued license IDs must not collide with pre-existing ones.
func TestGenesisRoundTripPreservesLicenseIDs(t *testing.T) {
	// Source keeper: create a license type, issue a handful of licenses,
	// then export the genesis state.
	src, srcCtx := keepertest.LicensesKeeper(t)
	owner := src.GetParams(srcCtx).Owner
	holder := sample.AccAddress()

	ms := keeper.NewMsgServerImpl(src)

	_, err := ms.CreateLicenseType(srcCtx, &types.MsgCreateLicenseType{
		Owner:         owner,
		Id:            "node",
		Transferrable: false,
		MaxSupply:     math.NewInt(100),
	})
	require.NoError(t, err)

	_, err = ms.GrantAdminPermissions(srcCtx, &types.MsgGrantAdminPermissions{
		Owner:   owner,
		Address: owner,
		Grants: []types.AdminKeyGrant{
			{Permission: "issue", LicenseTypes: []string{"node"}},
		},
	})
	require.NoError(t, err)

	resp, err := ms.IssueLicenses(srcCtx, &types.MsgIssueLicenses{
		Issuer: owner,
		Entries: []types.IssueLicenseEntry{
			{LicenseTypeId: "node", Holder: holder, StartDate: "2026-01-01", Count: 5},
		},
	})
	require.NoError(t, err)
	require.Equal(t, []uint64{1, 2, 3, 4, 5}, resp.Ids)

	exported := src.ExportGenesis(srcCtx)

	// Destination keeper: a fresh store, then InitGenesis with the exported
	// state. The genesis must include the existing license type and its
	// counters, so the next issued ID is 6, not 1.
	dst, dstCtx := keepertest.LicensesKeeper(t)
	require.NoError(t, dst.InitGenesis(dstCtx, exported))

	// Sanity: the imported license type's IssuedCount survived.
	lt, found, err := dst.GetLicenseType(dstCtx, "node")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, math.NewInt(5), lt.IssuedCount, "IssuedCount must survive genesis import")

	// Sanity: the imported licenses survived.
	for _, id := range resp.Ids {
		l, ok, err := dst.GetLicense(dstCtx, "node", id)
		require.NoError(t, err)
		require.True(t, ok, "license id %d must exist after import", id)
		require.Equal(t, holder, l.Holder)
	}

	// The bug: nextLicenseID resets to 0 because LicenseCounts isn't
	// restored, so issuing returns id=1 and overwrites the existing one.
	dstMs := keeper.NewMsgServerImpl(dst)
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
	l1, ok, err := dst.GetLicense(dstCtx, "node", 1)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, holder, l1.Holder, "imported license 1 must not have been overwritten")
}
