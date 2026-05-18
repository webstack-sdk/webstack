package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/webstack-sdk/webstack/testutil/keeper"
	"github.com/webstack-sdk/webstack/testutil/sample"
	"github.com/webstack-sdk/webstack/x/licenses/keeper"
	"github.com/webstack-sdk/webstack/x/licenses/types"
)

func setupMsgServer(t testing.TB) (keeper.Keeper, types.MsgServer, context.Context) {
	k, ctx := keepertest.LicensesKeeper(t)
	return k, keeper.NewMsgServerImpl(k), ctx
}

func setupWithOwner(t testing.TB) (keeper.Keeper, types.MsgServer, sdk.Context, string) {
	k, ms, goCtx := setupMsgServer(t)
	ctx := sdk.UnwrapSDKContext(goCtx)
	owner := k.GetParams(ctx).Owner
	return k, ms, ctx, owner
}

func TestMsgServer(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
}

// ---------------------------------------------------------------------------
// UpdateParams
// ---------------------------------------------------------------------------

func TestUpdateParams(t *testing.T) {
	k, ms, ctx, _ := setupWithOwner(t)
	newOwner := sample.AccAddress()

	tests := []struct {
		name      string
		input     *types.MsgUpdateParams
		expErr    bool
		expErrMsg string
	}{
		{
			name: "invalid authority",
			input: &types.MsgUpdateParams{
				Authority: "invalid",
				Params:    types.Params{Owner: newOwner},
			},
			expErr:    true,
			expErrMsg: "invalid authority",
		},
		{
			name: "valid",
			input: &types.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    types.Params{Owner: newOwner},
			},
			expErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.UpdateParams(ctx, tc.input)
			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, newOwner, k.GetParams(ctx).Owner)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CreateLicenseType
// ---------------------------------------------------------------------------

func TestCreateLicenseType(t *testing.T) {
	_, ms, ctx, owner := setupWithOwner(t)

	tests := []struct {
		name      string
		input     *types.MsgCreateLicenseType
		expErr    bool
		expErrMsg string
	}{
		{
			name: "non-owner",
			input: &types.MsgCreateLicenseType{
				Owner: sample.AccAddress(),
				Id:    "test.type",
			},
			expErr:    true,
			expErrMsg: "not the module owner",
		},
		{
			name: "empty id",
			input: &types.MsgCreateLicenseType{
				Owner: owner,
				Id:    "",
			},
			expErr:    true,
			expErrMsg: "cannot be empty",
		},
		{
			name: "valid",
			input: &types.MsgCreateLicenseType{
				Owner:         owner,
				Id:            "test.type",
				Transferrable: true,
				MaxSupply:     math.NewInt(100),
			},
			expErr: false,
		},
		{
			name: "duplicate",
			input: &types.MsgCreateLicenseType{
				Owner:     owner,
				Id:        "test.type",
				MaxSupply: math.ZeroInt(),
			},
			expErr:    true,
			expErrMsg: "already exists",
		},
		{
			name: "negative max_supply",
			input: &types.MsgCreateLicenseType{
				Owner:     owner,
				Id:        "neg.type",
				MaxSupply: math.NewInt(-1),
			},
			expErr:    true,
			expErrMsg: "max_supply must not be negative",
		},
		{
			name: "nil max_supply",
			input: &types.MsgCreateLicenseType{
				Owner: owner,
				Id:    "nil.type",
			},
			expErr:    true,
			expErrMsg: "max_supply must be set",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.CreateLicenseType(ctx, tc.input)
			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GrantAdminPermissions
// ---------------------------------------------------------------------------

func TestGrantAdminPermissions(t *testing.T) {
	k, ms, ctx, owner := setupWithOwner(t)
	adminAddr := sample.AccAddress()

	// Create license types referenced by the grants
	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{Owner: owner, Id: "t1", MaxSupply: math.ZeroInt()})
	require.NoError(t, err)
	_, err = ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{Owner: owner, Id: "t2", MaxSupply: math.ZeroInt()})
	require.NoError(t, err)

	tests := []struct {
		name      string
		input     *types.MsgGrantAdminPermissions
		expErr    bool
		expErrMsg string
	}{
		{
			name: "non-owner",
			input: &types.MsgGrantAdminPermissions{
				Owner:   sample.AccAddress(),
				Address: adminAddr,
				Grants: []types.AdminKeyGrant{
					{Permission: "issue", LicenseTypes: []string{"t1"}},
				},
			},
			expErr:    true,
			expErrMsg: "not the module owner",
		},
		{
			name: "invalid address",
			input: &types.MsgGrantAdminPermissions{
				Owner:   owner,
				Address: "bad",
				Grants: []types.AdminKeyGrant{
					{Permission: "issue", LicenseTypes: []string{"t1"}},
				},
			},
			expErr:    true,
			expErrMsg: "invalid address",
		},
		{
			name: "invalid permission",
			input: &types.MsgGrantAdminPermissions{
				Owner:   owner,
				Address: adminAddr,
				Grants: []types.AdminKeyGrant{
					{Permission: "destroy", LicenseTypes: []string{"t1"}},
				},
			},
			expErr:    true,
			expErrMsg: "invalid permission",
		},
		{
			name: "empty license types",
			input: &types.MsgGrantAdminPermissions{
				Owner:   owner,
				Address: adminAddr,
				Grants: []types.AdminKeyGrant{
					{Permission: "issue", LicenseTypes: []string{}},
				},
			},
			expErr:    true,
			expErrMsg: "at least one license type",
		},
		{
			name: "valid",
			input: &types.MsgGrantAdminPermissions{
				Owner:   owner,
				Address: adminAddr,
				Grants: []types.AdminKeyGrant{
					{Permission: "issue", LicenseTypes: []string{"t1", "t2"}},
					{Permission: "revoke", LicenseTypes: []string{"t1"}},
				},
			},
			expErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.GrantAdminPermissions(ctx, tc.input)
			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.True(t, k.HasPermission(ctx, adminAddr, "issue", "t1"))
				require.True(t, k.HasPermission(ctx, adminAddr, "issue", "t2"))
				require.True(t, k.HasPermission(ctx, adminAddr, "revoke", "t1"))
				require.False(t, k.HasPermission(ctx, adminAddr, "revoke", "t2"))
			}
		})
	}
}

// TestGrantAdminPermissionsMerges verifies that repeated GrantAdminPermissions
// calls for the same address accumulate grants rather than overwriting them.
func TestGrantAdminPermissionsMerges(t *testing.T) {
	k, ms, ctx, owner := setupWithOwner(t)
	adminAddr := sample.AccAddress()

	for _, id := range []string{"t1", "t2", "t3"} {
		_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{Owner: owner, Id: id, MaxSupply: math.ZeroInt()})
		require.NoError(t, err)
	}

	_, err := ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: adminAddr,
		Grants: []types.AdminKeyGrant{{Permission: "issue", LicenseTypes: []string{"t1"}}},
	})
	require.NoError(t, err)

	// Adding a new permission must not drop the previous one.
	_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: adminAddr,
		Grants: []types.AdminKeyGrant{{Permission: "revoke", LicenseTypes: []string{"t1"}}},
	})
	require.NoError(t, err)

	// Extending an existing permission with a new license type must union, not replace.
	_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: adminAddr,
		Grants: []types.AdminKeyGrant{{Permission: "issue", LicenseTypes: []string{"t2", "t3"}}},
	})
	require.NoError(t, err)

	// Re-granting the same (permission, license type) pair must be a no-op (dedup).
	_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: adminAddr,
		Grants: []types.AdminKeyGrant{{Permission: "issue", LicenseTypes: []string{"t1"}}},
	})
	require.NoError(t, err)

	require.True(t, k.HasPermission(ctx, adminAddr, "issue", "t1"))
	require.True(t, k.HasPermission(ctx, adminAddr, "issue", "t2"))
	require.True(t, k.HasPermission(ctx, adminAddr, "issue", "t3"))
	require.True(t, k.HasPermission(ctx, adminAddr, "revoke", "t1"))
	require.False(t, k.HasPermission(ctx, adminAddr, "revoke", "t2"))

	// State should be deterministic: grants sorted by permission, license types sorted within each grant,
	// with no duplicates.
	ak, err := k.AdminKeys.Get(ctx, adminAddr)
	require.NoError(t, err)
	require.Len(t, ak.Grants, 2)
	require.Equal(t, "issue", ak.Grants[0].Permission)
	require.Equal(t, []string{"t1", "t2", "t3"}, ak.Grants[0].LicenseTypes)
	require.Equal(t, "revoke", ak.Grants[1].Permission)
	require.Equal(t, []string{"t1"}, ak.Grants[1].LicenseTypes)
}

// ---------------------------------------------------------------------------
// RevokeAdminKeyPermissions
// ---------------------------------------------------------------------------

func TestRevokeAdminKeyPermissions(t *testing.T) {
	k, ms, ctx, owner := setupWithOwner(t)
	adminAddr := sample.AccAddress()

	for _, id := range []string{"t1", "t2"} {
		_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{Owner: owner, Id: id, MaxSupply: math.ZeroInt()})
		require.NoError(t, err)
	}

	seed := func(t *testing.T) {
		t.Helper()
		// Reset to a known set of grants for each subtest.
		_ = k.AdminKeys.Remove(ctx, adminAddr)
		_, err := ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
			Owner: owner, Address: adminAddr,
			Grants: []types.AdminKeyGrant{
				{Permission: "issue", LicenseTypes: []string{"t1", "t2"}},
				{Permission: "revoke", LicenseTypes: []string{"t1"}},
			},
		})
		require.NoError(t, err)
	}

	t.Run("non-owner is rejected", func(t *testing.T) {
		seed(t)
		_, err := ms.RevokeAdminKeyPermissions(ctx, &types.MsgRevokeAdminKeyPermissions{
			Owner:   sample.AccAddress(),
			Address: adminAddr,
			Permissions: []types.AdminKeyPermission{
				{LicenseTypeId: "t1", Permission: "issue"},
			},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not the module owner")
		// state is unchanged
		require.True(t, k.HasPermission(ctx, adminAddr, "issue", "t1"))
	})

	t.Run("removes matching pair, leaves others", func(t *testing.T) {
		seed(t)
		_, err := ms.RevokeAdminKeyPermissions(ctx, &types.MsgRevokeAdminKeyPermissions{
			Owner: owner, Address: adminAddr,
			Permissions: []types.AdminKeyPermission{
				{LicenseTypeId: "t1", Permission: "issue"},
			},
		})
		require.NoError(t, err)
		require.False(t, k.HasPermission(ctx, adminAddr, "issue", "t1"))
		require.True(t, k.HasPermission(ctx, adminAddr, "issue", "t2"))
		require.True(t, k.HasPermission(ctx, adminAddr, "revoke", "t1"))
	})

	t.Run("dropping last license type drops the grant", func(t *testing.T) {
		seed(t)
		_, err := ms.RevokeAdminKeyPermissions(ctx, &types.MsgRevokeAdminKeyPermissions{
			Owner: owner, Address: adminAddr,
			Permissions: []types.AdminKeyPermission{
				{LicenseTypeId: "t1", Permission: "revoke"},
			},
		})
		require.NoError(t, err)
		ak, err := k.AdminKeys.Get(ctx, adminAddr)
		require.NoError(t, err)
		// Only the "issue" grant should remain — "revoke" had only t1, now empty.
		require.Len(t, ak.Grants, 1)
		require.Equal(t, "issue", ak.Grants[0].Permission)
	})

	t.Run("removing every pair deletes the admin key entry", func(t *testing.T) {
		seed(t)
		_, err := ms.RevokeAdminKeyPermissions(ctx, &types.MsgRevokeAdminKeyPermissions{
			Owner: owner, Address: adminAddr,
			Permissions: []types.AdminKeyPermission{
				{LicenseTypeId: "t1", Permission: "issue"},
				{LicenseTypeId: "t2", Permission: "issue"},
				{LicenseTypeId: "t1", Permission: "revoke"},
			},
		})
		require.NoError(t, err)
		_, err = k.AdminKeys.Get(ctx, adminAddr)
		require.Error(t, err, "admin key entry should be deleted when no grants remain")
	})

	t.Run("unknown pairs are silently ignored", func(t *testing.T) {
		seed(t)
		_, err := ms.RevokeAdminKeyPermissions(ctx, &types.MsgRevokeAdminKeyPermissions{
			Owner: owner, Address: adminAddr,
			Permissions: []types.AdminKeyPermission{
				{LicenseTypeId: "does-not-exist", Permission: "issue"},
				{LicenseTypeId: "t1", Permission: "update"},
				{LicenseTypeId: "t2", Permission: "issue"}, // this one matches
			},
		})
		require.NoError(t, err)
		require.True(t, k.HasPermission(ctx, adminAddr, "issue", "t1"))
		require.False(t, k.HasPermission(ctx, adminAddr, "issue", "t2"))
		require.True(t, k.HasPermission(ctx, adminAddr, "revoke", "t1"))
	})

	t.Run("revoke on missing admin key is a no-op", func(t *testing.T) {
		other := sample.AccAddress()
		_, err := ms.RevokeAdminKeyPermissions(ctx, &types.MsgRevokeAdminKeyPermissions{
			Owner: owner, Address: other,
			Permissions: []types.AdminKeyPermission{
				{LicenseTypeId: "t1", Permission: "issue"},
			},
		})
		require.NoError(t, err)
	})
}

// ---------------------------------------------------------------------------
// IssueLicense
// ---------------------------------------------------------------------------

func TestIssueLicense(t *testing.T) {
	k, ms, ctx, owner := setupWithOwner(t)
	issuer := sample.AccAddress()
	holder := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "node", MaxSupply: math.NewInt(10),
	})
	require.NoError(t, err)

	_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: issuer,
		Grants: []types.AdminKeyGrant{{Permission: "issue", LicenseTypes: []string{"node"}}},
	})
	require.NoError(t, err)

	tests := []struct {
		name      string
		input     *types.MsgIssueLicense
		expErr    bool
		expErrMsg string
		expCount  int
	}{
		{
			name: "no permission",
			input: &types.MsgIssueLicense{
				Issuer: sample.AccAddress(), LicenseTypeId: "node",
				Holder: holder, StartDate: "2026-01-01",
			},
			expErr:    true,
			expErrMsg: "does not have issue permission",
		},
		{
			name: "invalid holder",
			input: &types.MsgIssueLicense{
				Issuer: issuer, LicenseTypeId: "node",
				Holder: "bad", StartDate: "2026-01-01",
			},
			expErr:    true,
			expErrMsg: "invalid holder address",
		},
		{
			name: "missing start_date",
			input: &types.MsgIssueLicense{
				Issuer: issuer, LicenseTypeId: "node",
				Holder: holder, StartDate: "",
			},
			expErr:    true,
			expErrMsg: "start_date is required",
		},
		{
			name: "bad date format",
			input: &types.MsgIssueLicense{
				Issuer: issuer, LicenseTypeId: "node",
				Holder: holder, StartDate: "01-01-2026",
			},
			expErr:    true,
			expErrMsg: "YYYY-MM-DD",
		},
		{
			name: "end_date before start_date",
			input: &types.MsgIssueLicense{
				Issuer: issuer, LicenseTypeId: "node",
				Holder: holder, StartDate: "2026-06-01", EndDate: "2026-01-01",
			},
			expErr:    true,
			expErrMsg: "must not be before",
		},
		{
			name: "valid single",
			input: &types.MsgIssueLicense{
				Issuer: issuer, LicenseTypeId: "node",
				Holder: holder, StartDate: "2026-01-01", EndDate: "2027-01-01",
			},
			expErr:   false,
			expCount: 1,
		},
		{
			name: "valid with count=3",
			input: &types.MsgIssueLicense{
				Issuer: issuer, LicenseTypeId: "node",
				Holder: holder, StartDate: "2026-01-01", Count: 3,
			},
			expErr:   false,
			expCount: 3,
		},
		{
			name: "max supply exceeded",
			input: &types.MsgIssueLicense{
				Issuer: issuer, LicenseTypeId: "node",
				Holder: holder, StartDate: "2026-01-01", Count: 10,
			},
			expErr:    true,
			expErrMsg: "exceed max supply",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.IssueLicense(ctx, tc.input)
			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.Len(t, resp.Ids, tc.expCount)
			}
		})
	}

	lt, found, err := k.GetLicenseType(ctx, "node")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, math.NewInt(4), lt.IssuedCount)
}

// ---------------------------------------------------------------------------
// BatchIssueLicense
// ---------------------------------------------------------------------------

func TestBatchIssueLicense(t *testing.T) {
	k, ms, ctx, owner := setupWithOwner(t)
	issuer := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "batch", MaxSupply: math.NewInt(5),
	})
	require.NoError(t, err)

	_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: issuer,
		Grants: []types.AdminKeyGrant{{Permission: "issue", LicenseTypes: []string{"batch"}}},
	})
	require.NoError(t, err)

	tests := []struct {
		name      string
		input     *types.MsgBatchIssueLicense
		expErr    bool
		expErrMsg string
		expCount  int
	}{
		{
			name: "empty entries",
			input: &types.MsgBatchIssueLicense{
				Issuer: issuer, LicenseTypeId: "batch",
				Entries: []types.BatchIssueLicenseEntry{},
			},
			expErr:    true,
			expErrMsg: "must not be empty",
		},
		{
			name: "no permission",
			input: &types.MsgBatchIssueLicense{
				Issuer: sample.AccAddress(), LicenseTypeId: "batch",
				Entries: []types.BatchIssueLicenseEntry{
					{Holder: sample.AccAddress(), StartDate: "2026-01-01"},
				},
			},
			expErr:    true,
			expErrMsg: "does not have issue permission",
		},
		{
			name: "invalid holder in entry",
			input: &types.MsgBatchIssueLicense{
				Issuer: issuer, LicenseTypeId: "batch",
				Entries: []types.BatchIssueLicenseEntry{
					{Holder: "bad", StartDate: "2026-01-01"},
				},
			},
			expErr:    true,
			expErrMsg: "invalid holder address",
		},
		{
			name: "invalid date in entry",
			input: &types.MsgBatchIssueLicense{
				Issuer: issuer, LicenseTypeId: "batch",
				Entries: []types.BatchIssueLicenseEntry{
					{Holder: sample.AccAddress(), StartDate: "bad-date"},
				},
			},
			expErr:    true,
			expErrMsg: "YYYY-MM-DD",
		},
		{
			name: "valid 3 entries",
			input: &types.MsgBatchIssueLicense{
				Issuer: issuer, LicenseTypeId: "batch",
				Entries: []types.BatchIssueLicenseEntry{
					{Holder: sample.AccAddress(), StartDate: "2026-01-01", EndDate: "2027-01-01"},
					{Holder: sample.AccAddress(), StartDate: "2026-02-01"},
					{Holder: sample.AccAddress(), StartDate: "2026-03-01"},
				},
			},
			expErr:   false,
			expCount: 3,
		},
		{
			name: "max supply exceeded",
			input: &types.MsgBatchIssueLicense{
				Issuer: issuer, LicenseTypeId: "batch",
				Entries: []types.BatchIssueLicenseEntry{
					{Holder: sample.AccAddress(), StartDate: "2026-01-01"},
					{Holder: sample.AccAddress(), StartDate: "2026-01-01"},
					{Holder: sample.AccAddress(), StartDate: "2026-01-01"},
				},
			},
			expErr:    true,
			expErrMsg: "exceed max supply",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.BatchIssueLicense(ctx, tc.input)
			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.Len(t, resp.Ids, tc.expCount)
			}
		})
	}

	lt, found, err := k.GetLicenseType(ctx, "batch")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, math.NewInt(3), lt.IssuedCount)
}

// ---------------------------------------------------------------------------
// RevokeLicense
// ---------------------------------------------------------------------------

func TestRevokeLicense(t *testing.T) {
	k, ms, ctx, owner := setupWithOwner(t)
	issuer := sample.AccAddress()
	revoker := sample.AccAddress()
	holder := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "rev", MaxSupply: math.ZeroInt(),
	})
	require.NoError(t, err)
	_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: issuer,
		Grants: []types.AdminKeyGrant{{Permission: "issue", LicenseTypes: []string{"rev"}}},
	})
	require.NoError(t, err)
	_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: revoker,
		Grants: []types.AdminKeyGrant{{Permission: "revoke", LicenseTypes: []string{"rev"}}},
	})
	require.NoError(t, err)

	// Issue 3 licenses to the same holder.
	resp, err := ms.IssueLicense(ctx, &types.MsgIssueLicense{
		Issuer: issuer, LicenseTypeId: "rev", Holder: holder, StartDate: "2026-01-01", Count: 3,
	})
	require.NoError(t, err)
	require.Len(t, resp.Ids, 3)

	tests := []struct {
		name      string
		input     *types.MsgRevokeLicense
		expErr    bool
		expErrMsg string
	}{
		{
			name:      "no permission",
			input:     &types.MsgRevokeLicense{Revoker: sample.AccAddress(), LicenseTypeId: "rev", Holder: holder, Count: 1},
			expErr:    true,
			expErrMsg: "does not have revoke permission",
		},
		{
			name:      "not enough active licenses",
			input:     &types.MsgRevokeLicense{Revoker: revoker, LicenseTypeId: "rev", Holder: holder, Count: 10},
			expErr:    true,
			expErrMsg: "has 3 active license(s)",
		},
		{
			name:   "revoke 2 — most recent first",
			input:  &types.MsgRevokeLicense{Revoker: revoker, LicenseTypeId: "rev", Holder: holder, Count: 2},
			expErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			revokeResp, err := ms.RevokeLicense(ctx, tc.input)
			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.Len(t, revokeResp.Ids, 2)
				// Most recently issued (id=3) should be revoked first, then id=2.
				require.Equal(t, resp.Ids[2], revokeResp.Ids[0])
				require.Equal(t, resp.Ids[1], revokeResp.Ids[1])

				// Verify revoked licenses have status and end_date set.
				for _, id := range revokeResp.Ids {
					license, found, _ := k.GetLicense(ctx, "rev", id)
					require.True(t, found)
					require.Equal(t, "revoked", license.Status)
					require.NotEmpty(t, license.EndDate)
				}

				// Verify the remaining license is still active.
				license, found, _ := k.GetLicense(ctx, "rev", resp.Ids[0])
				require.True(t, found)
				require.Equal(t, "active", license.Status)

				// Verify counters.
				lt, _, _ := k.GetLicenseType(ctx, "rev")
				require.Equal(t, math.NewInt(3), lt.IssuedCount)
				require.Equal(t, math.NewInt(1), lt.ActiveCount)
				require.Equal(t, math.NewInt(2), lt.RevokedCount)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TransferLicense
// ---------------------------------------------------------------------------

func TestTransferLicense(t *testing.T) {
	k, ms, ctx, owner := setupWithOwner(t)
	issuer := sample.AccAddress()
	holder := sample.AccAddress()
	recipient := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "xfer", Transferrable: true, MaxSupply: math.ZeroInt(),
	})
	require.NoError(t, err)

	_, err = ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "noxfer", Transferrable: false, MaxSupply: math.ZeroInt(),
	})
	require.NoError(t, err)

	_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: issuer,
		Grants: []types.AdminKeyGrant{{Permission: "issue", LicenseTypes: []string{"xfer", "noxfer"}}},
	})
	require.NoError(t, err)

	resp, err := ms.IssueLicense(ctx, &types.MsgIssueLicense{
		Issuer: issuer, LicenseTypeId: "xfer", Holder: holder, StartDate: "2026-01-01",
	})
	require.NoError(t, err)
	xferID := resp.Ids[0]

	resp, err = ms.IssueLicense(ctx, &types.MsgIssueLicense{
		Issuer: issuer, LicenseTypeId: "noxfer", Holder: holder, StartDate: "2026-01-01",
	})
	require.NoError(t, err)
	noxferID := resp.Ids[0]

	tests := []struct {
		name      string
		input     *types.MsgTransferLicense
		expErr    bool
		expErrMsg string
	}{
		{
			name:      "invalid recipient",
			input:     &types.MsgTransferLicense{Holder: holder, LicenseTypeId: "xfer", Id: xferID, Recipient: "bad"},
			expErr:    true,
			expErrMsg: "invalid recipient address",
		},
		{
			name:      "transfer to self",
			input:     &types.MsgTransferLicense{Holder: holder, LicenseTypeId: "xfer", Id: xferID, Recipient: holder},
			expErr:    true,
			expErrMsg: "cannot transfer license to the current holder",
		},
		{
			name:      "not found",
			input:     &types.MsgTransferLicense{Holder: holder, LicenseTypeId: "xfer", Id: 999, Recipient: recipient},
			expErr:    true,
			expErrMsg: "not found",
		},
		{
			name:      "not holder",
			input:     &types.MsgTransferLicense{Holder: sample.AccAddress(), LicenseTypeId: "xfer", Id: xferID, Recipient: recipient},
			expErr:    true,
			expErrMsg: "not the holder",
		},
		{
			name:      "not transferrable",
			input:     &types.MsgTransferLicense{Holder: holder, LicenseTypeId: "noxfer", Id: noxferID, Recipient: recipient},
			expErr:    true,
			expErrMsg: "not transferrable",
		},
		{
			name:   "valid",
			input:  &types.MsgTransferLicense{Holder: holder, LicenseTypeId: "xfer", Id: xferID, Recipient: recipient},
			expErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.TransferLicense(ctx, tc.input)
			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				l, found, _ := k.GetLicense(ctx, "xfer", xferID)
				require.True(t, found)
				require.Equal(t, recipient, l.Holder)
			}
		})
	}
}

// TestIssueLicenseSupplyCheckIsUnsigned guards the supply-cap arithmetic:
// a count with the high bit set must not wrap negative and silently bypass
// the MaxSupply check.
func TestIssueLicenseSupplyCheckIsUnsigned(t *testing.T) {
	_, ms, ctx, owner := setupWithOwner(t)
	issuer := sample.AccAddress()
	holder := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "lim", Transferrable: false, MaxSupply: math.NewInt(100),
	})
	require.NoError(t, err)
	_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: issuer,
		Grants: []types.AdminKeyGrant{{Permission: "issue", LicenseTypes: []string{"lim"}}},
	})
	require.NoError(t, err)

	// 1<<63 is the smallest uint64 value that wraps to a negative int64.
	_, err = ms.IssueLicense(ctx, &types.MsgIssueLicense{
		Issuer: issuer, LicenseTypeId: "lim",
		Holder: holder, StartDate: "2026-01-01",
		Count:  1 << 63,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceed max supply")
}

// TestBatchIssueLicenseEntriesCap ensures BatchIssueLicense rejects entry
// lists larger than MaxIssueBatchSize.
func TestBatchIssueLicenseEntriesCap(t *testing.T) {
	_, ms, ctx, owner := setupWithOwner(t)
	issuer := sample.AccAddress()
	holder := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "cap", Transferrable: false, MaxSupply: math.ZeroInt(),
	})
	require.NoError(t, err)
	_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: issuer,
		Grants: []types.AdminKeyGrant{{Permission: "issue", LicenseTypes: []string{"cap"}}},
	})
	require.NoError(t, err)

	entries := make([]types.BatchIssueLicenseEntry, types.MaxIssueBatchSize+1)
	for i := range entries {
		entries[i] = types.BatchIssueLicenseEntry{Holder: holder, StartDate: "2026-01-01"}
	}

	_, err = ms.BatchIssueLicense(ctx, &types.MsgBatchIssueLicense{
		Issuer: issuer, LicenseTypeId: "cap", Entries: entries,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds max batch size")
}

// TestTransferLicenseRejectsRevoked: a revoked license must not be
// transferable, even though the license entry and LicenseByHolder index
// still exist under the original holder.
func TestTransferLicenseRejectsRevoked(t *testing.T) {
	k, ms, ctx, owner := setupWithOwner(t)
	issuer := sample.AccAddress()
	holder := sample.AccAddress()
	recipient := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "xfer", Transferrable: true, MaxSupply: math.ZeroInt(),
	})
	require.NoError(t, err)

	_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: issuer,
		Grants: []types.AdminKeyGrant{
			{Permission: "issue", LicenseTypes: []string{"xfer"}},
			{Permission: "revoke", LicenseTypes: []string{"xfer"}},
		},
	})
	require.NoError(t, err)

	resp, err := ms.IssueLicense(ctx, &types.MsgIssueLicense{
		Issuer: issuer, LicenseTypeId: "xfer", Holder: holder, StartDate: "2026-01-01",
	})
	require.NoError(t, err)
	id := resp.Ids[0]

	_, err = ms.RevokeLicense(ctx, &types.MsgRevokeLicense{
		Revoker: issuer, LicenseTypeId: "xfer", Holder: holder, Count: 1,
	})
	require.NoError(t, err)

	// Sanity: the license entry still exists with status=revoked.
	l, found, err := k.GetLicense(ctx, "xfer", id)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "revoked", l.Status)
	require.Equal(t, holder, l.Holder, "revoked license is still indexed under the original holder")

	// Attempting to transfer the revoked license must fail.
	_, err = ms.TransferLicense(ctx, &types.MsgTransferLicense{
		Holder: holder, LicenseTypeId: "xfer", Id: id, Recipient: recipient,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "revoked")

	// And the holder must not have changed.
	l, _, err = k.GetLicense(ctx, "xfer", id)
	require.NoError(t, err)
	require.Equal(t, holder, l.Holder)
}

// ---------------------------------------------------------------------------
// UpdateLicenseType
// ---------------------------------------------------------------------------

func TestUpdateLicenseType(t *testing.T) {
	_, ms, ctx, owner := setupWithOwner(t)
	issuer := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "lt1", Transferrable: false, MaxSupply: math.NewInt(100),
	})
	require.NoError(t, err)
	_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: issuer,
		Grants: []types.AdminKeyGrant{{Permission: "issue", LicenseTypes: []string{"lt1"}}},
	})
	require.NoError(t, err)
	_, err = ms.IssueLicense(ctx, &types.MsgIssueLicense{
		Issuer: issuer, LicenseTypeId: "lt1", Holder: sample.AccAddress(),
		StartDate: "2026-01-01", Count: 5,
	})
	require.NoError(t, err)

	tests := []struct {
		name      string
		input     *types.MsgUpdateLicenseType
		expErr    bool
		expErrMsg string
	}{
		{
			name:      "non-owner",
			input:     &types.MsgUpdateLicenseType{Owner: sample.AccAddress(), Id: "lt1", Transferrable: true, MaxSupply: math.NewInt(200)},
			expErr:    true,
			expErrMsg: "not the module owner",
		},
		{
			name:      "not found",
			input:     &types.MsgUpdateLicenseType{Owner: owner, Id: "missing", Transferrable: true, MaxSupply: math.NewInt(200)},
			expErr:    true,
			expErrMsg: "not found",
		},
		{
			name:      "max_supply below issued_count",
			input:     &types.MsgUpdateLicenseType{Owner: owner, Id: "lt1", Transferrable: true, MaxSupply: math.NewInt(3)},
			expErr:    true,
			expErrMsg: "cannot set max_supply",
		},
		{
			name:      "negative max_supply",
			input:     &types.MsgUpdateLicenseType{Owner: owner, Id: "lt1", Transferrable: true, MaxSupply: math.NewInt(-1)},
			expErr:    true,
			expErrMsg: "max_supply must not be negative",
		},
		{
			name:      "nil max_supply",
			input:     &types.MsgUpdateLicenseType{Owner: owner, Id: "lt1", Transferrable: true},
			expErr:    true,
			expErrMsg: "max_supply must be set",
		},
		{
			name:   "valid update",
			input:  &types.MsgUpdateLicenseType{Owner: owner, Id: "lt1", Transferrable: true, MaxSupply: math.NewInt(200)},
			expErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.UpdateLicenseType(ctx, tc.input)
			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
