package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/collections"
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
					{Permission: types.PermissionIssue, LicenseTypes: []string{"t1"}},
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
					{Permission: types.PermissionIssue, LicenseTypes: []string{"t1"}},
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
					{Permission: types.Permission(99), LicenseTypes: []string{"t1"}},
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
					{Permission: types.PermissionIssue, LicenseTypes: []string{}},
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
					{Permission: types.PermissionIssue, LicenseTypes: []string{"t1", "t2"}},
					{Permission: types.PermissionRevoke, LicenseTypes: []string{"t1"}},
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
				require.True(t, k.HasPermission(ctx, adminAddr, types.PermissionIssue, "t1"))
				require.True(t, k.HasPermission(ctx, adminAddr, types.PermissionIssue, "t2"))
				require.True(t, k.HasPermission(ctx, adminAddr, types.PermissionRevoke, "t1"))
				require.False(t, k.HasPermission(ctx, adminAddr, types.PermissionRevoke, "t2"))
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
		Grants: []types.AdminKeyGrant{{Permission: types.PermissionIssue, LicenseTypes: []string{"t1"}}},
	})
	require.NoError(t, err)

	// Adding a new permission must not drop the previous one.
	_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: adminAddr,
		Grants: []types.AdminKeyGrant{{Permission: types.PermissionRevoke, LicenseTypes: []string{"t1"}}},
	})
	require.NoError(t, err)

	// Extending an existing permission with a new license type must union, not replace.
	_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: adminAddr,
		Grants: []types.AdminKeyGrant{{Permission: types.PermissionIssue, LicenseTypes: []string{"t2", "t3"}}},
	})
	require.NoError(t, err)

	// Re-granting the same (permission, license type) pair must be a no-op (dedup).
	_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: adminAddr,
		Grants: []types.AdminKeyGrant{{Permission: types.PermissionIssue, LicenseTypes: []string{"t1"}}},
	})
	require.NoError(t, err)

	require.True(t, k.HasPermission(ctx, adminAddr, types.PermissionIssue, "t1"))
	require.True(t, k.HasPermission(ctx, adminAddr, types.PermissionIssue, "t2"))
	require.True(t, k.HasPermission(ctx, adminAddr, types.PermissionIssue, "t3"))
	require.True(t, k.HasPermission(ctx, adminAddr, types.PermissionRevoke, "t1"))
	require.False(t, k.HasPermission(ctx, adminAddr, types.PermissionRevoke, "t2"))

	// State should be deterministic: grants sorted by permission, license types sorted within each grant,
	// with no duplicates.
	ak, found, err := k.GetAdminKey(ctx, adminAddr)
	require.NoError(t, err)
	require.True(t, found)
	require.Len(t, ak.Grants, 2)
	require.Equal(t, types.PermissionIssue, ak.Grants[0].Permission)
	require.Equal(t, []string{"t1", "t2", "t3"}, ak.Grants[0].LicenseTypes)
	require.Equal(t, types.PermissionRevoke, ak.Grants[1].Permission)
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
		require.NoError(t, k.AdminGrants.Clear(ctx, collections.NewPrefixedTripleRange[string, int32, string](adminAddr)))
		_, err := ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
			Owner: owner, Address: adminAddr,
			Grants: []types.AdminKeyGrant{
				{Permission: types.PermissionIssue, LicenseTypes: []string{"t1", "t2"}},
				{Permission: types.PermissionRevoke, LicenseTypes: []string{"t1"}},
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
				{LicenseTypeId: "t1", Permission: types.PermissionIssue},
			},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not the module owner")
		// state is unchanged
		require.True(t, k.HasPermission(ctx, adminAddr, types.PermissionIssue, "t1"))
	})

	t.Run("removes matching pair, leaves others", func(t *testing.T) {
		seed(t)
		_, err := ms.RevokeAdminKeyPermissions(ctx, &types.MsgRevokeAdminKeyPermissions{
			Owner: owner, Address: adminAddr,
			Permissions: []types.AdminKeyPermission{
				{LicenseTypeId: "t1", Permission: types.PermissionIssue},
			},
		})
		require.NoError(t, err)
		require.False(t, k.HasPermission(ctx, adminAddr, types.PermissionIssue, "t1"))
		require.True(t, k.HasPermission(ctx, adminAddr, types.PermissionIssue, "t2"))
		require.True(t, k.HasPermission(ctx, adminAddr, types.PermissionRevoke, "t1"))
	})

	t.Run("dropping last license type drops the grant", func(t *testing.T) {
		seed(t)
		_, err := ms.RevokeAdminKeyPermissions(ctx, &types.MsgRevokeAdminKeyPermissions{
			Owner: owner, Address: adminAddr,
			Permissions: []types.AdminKeyPermission{
				{LicenseTypeId: "t1", Permission: types.PermissionRevoke},
			},
		})
		require.NoError(t, err)
		ak, found, err := k.GetAdminKey(ctx, adminAddr)
		require.NoError(t, err)
		require.True(t, found)
		// Only the "issue" grant should remain — "revoke" had only t1, now empty.
		require.Len(t, ak.Grants, 1)
		require.Equal(t, types.PermissionIssue, ak.Grants[0].Permission)
	})

	t.Run("removing every pair deletes the admin key entry", func(t *testing.T) {
		seed(t)
		_, err := ms.RevokeAdminKeyPermissions(ctx, &types.MsgRevokeAdminKeyPermissions{
			Owner: owner, Address: adminAddr,
			Permissions: []types.AdminKeyPermission{
				{LicenseTypeId: "t1", Permission: types.PermissionIssue},
				{LicenseTypeId: "t2", Permission: types.PermissionIssue},
				{LicenseTypeId: "t1", Permission: types.PermissionRevoke},
			},
		})
		require.NoError(t, err)
		_, found, err := k.GetAdminKey(ctx, adminAddr)
		require.NoError(t, err)
		require.False(t, found, "admin key entry should be gone when no grants remain")
	})

	t.Run("unknown pairs are silently ignored", func(t *testing.T) {
		seed(t)
		_, err := ms.RevokeAdminKeyPermissions(ctx, &types.MsgRevokeAdminKeyPermissions{
			Owner: owner, Address: adminAddr,
			Permissions: []types.AdminKeyPermission{
				{LicenseTypeId: "does-not-exist", Permission: types.PermissionIssue},
				{LicenseTypeId: "t1", Permission: types.Permission(99)},
				{LicenseTypeId: "t2", Permission: types.PermissionIssue}, // this one matches
			},
		})
		require.NoError(t, err)
		require.True(t, k.HasPermission(ctx, adminAddr, types.PermissionIssue, "t1"))
		require.False(t, k.HasPermission(ctx, adminAddr, types.PermissionIssue, "t2"))
		require.True(t, k.HasPermission(ctx, adminAddr, types.PermissionRevoke, "t1"))
	})

	t.Run("revoke on missing admin key is a no-op", func(t *testing.T) {
		other := sample.AccAddress()
		_, err := ms.RevokeAdminKeyPermissions(ctx, &types.MsgRevokeAdminKeyPermissions{
			Owner: owner, Address: other,
			Permissions: []types.AdminKeyPermission{
				{LicenseTypeId: "t1", Permission: types.PermissionIssue},
			},
		})
		require.NoError(t, err)
	})

	t.Run("re-grant after full removal works", func(t *testing.T) {
		seed(t)
		_, err := ms.RevokeAdminKeyPermissions(ctx, &types.MsgRevokeAdminKeyPermissions{
			Owner: owner, Address: adminAddr,
			Permissions: []types.AdminKeyPermission{
				{LicenseTypeId: "t1", Permission: types.PermissionIssue},
				{LicenseTypeId: "t2", Permission: types.PermissionIssue},
				{LicenseTypeId: "t1", Permission: types.PermissionRevoke},
			},
		})
		require.NoError(t, err)

		_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
			Owner: owner, Address: adminAddr,
			Grants: []types.AdminKeyGrant{{Permission: types.PermissionRevoke, LicenseTypes: []string{"t2"}}},
		})
		require.NoError(t, err)
		require.True(t, k.HasPermission(ctx, adminAddr, types.PermissionRevoke, "t2"))
		require.False(t, k.HasPermission(ctx, adminAddr, types.PermissionIssue, "t1"))
	})
}

// ---------------------------------------------------------------------------
// IssueLicenses
// ---------------------------------------------------------------------------

func TestIssueLicenses(t *testing.T) {
	k, ms, ctx, owner := setupWithOwner(t)
	issuer := sample.AccAddress()
	holder := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "node", MaxSupply: math.NewInt(10),
	})
	require.NoError(t, err)

	_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: issuer,
		Grants: []types.AdminKeyGrant{{Permission: types.PermissionIssue, LicenseTypes: []string{"node"}}},
	})
	require.NoError(t, err)

	tests := []struct {
		name      string
		input     *types.MsgIssueLicenses
		expErr    bool
		expErrMsg string
		expCount  int
	}{
		{
			name: "empty entries",
			input: &types.MsgIssueLicenses{
				Issuer: issuer, Entries: []types.IssueLicenseEntry{},
			},
			expErr:    true,
			expErrMsg: "must not be empty",
		},
		{
			name: "no permission",
			input: &types.MsgIssueLicenses{
				Issuer: sample.AccAddress(), Entries: []types.IssueLicenseEntry{
					{LicenseTypeId: "node", Holder: holder, StartDate: "2026-01-01", Count: 1},
				},
			},
			expErr:    true,
			expErrMsg: "does not have issue permission",
		},
		{
			name: "invalid holder",
			input: &types.MsgIssueLicenses{
				Issuer: issuer, Entries: []types.IssueLicenseEntry{
					{LicenseTypeId: "node", Holder: "bad", StartDate: "2026-01-01", Count: 1},
				},
			},
			expErr:    true,
			expErrMsg: "invalid holder address",
		},
		{
			name: "missing start_date",
			input: &types.MsgIssueLicenses{
				Issuer: issuer, Entries: []types.IssueLicenseEntry{
					{LicenseTypeId: "node", Holder: holder, StartDate: "", Count: 1},
				},
			},
			expErr:    true,
			expErrMsg: "start_date is required",
		},
		{
			name: "bad date format",
			input: &types.MsgIssueLicenses{
				Issuer: issuer, Entries: []types.IssueLicenseEntry{
					{LicenseTypeId: "node", Holder: holder, StartDate: "01-01-2026", Count: 1},
				},
			},
			expErr:    true,
			expErrMsg: "YYYY-MM-DD",
		},
		{
			name: "end_date before start_date",
			input: &types.MsgIssueLicenses{
				Issuer: issuer, Entries: []types.IssueLicenseEntry{
					{LicenseTypeId: "node", Holder: holder, StartDate: "2026-06-01", EndDate: "2026-01-01", Count: 1},
				},
			},
			expErr:    true,
			expErrMsg: "must not be before",
		},
		{
			name: "count zero",
			input: &types.MsgIssueLicenses{
				Issuer: issuer, Entries: []types.IssueLicenseEntry{
					{LicenseTypeId: "node", Holder: holder, StartDate: "2026-01-01"},
				},
			},
			expErr:    true,
			expErrMsg: "count must be greater than zero",
		},
		{
			name: "valid single",
			input: &types.MsgIssueLicenses{
				Issuer: issuer, Entries: []types.IssueLicenseEntry{
					{LicenseTypeId: "node", Holder: holder, StartDate: "2026-01-01", EndDate: "2027-01-01", Count: 1},
				},
			},
			expErr:   false,
			expCount: 1,
		},
		{
			name: "valid with count=3",
			input: &types.MsgIssueLicenses{
				Issuer: issuer, Entries: []types.IssueLicenseEntry{
					{LicenseTypeId: "node", Holder: holder, StartDate: "2026-01-01", Count: 3},
				},
			},
			expErr:   false,
			expCount: 3,
		},
		{
			name: "max supply exceeded",
			input: &types.MsgIssueLicenses{
				Issuer: issuer, Entries: []types.IssueLicenseEntry{
					{LicenseTypeId: "node", Holder: holder, StartDate: "2026-01-01", Count: 10},
				},
			},
			expErr:    true,
			expErrMsg: "exceed max supply",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := ms.IssueLicenses(ctx, tc.input)
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

// TestIssueLicensesMultipleEntries covers the multi-entry behavior: entries
// can target different holders and license types, per-entry counts accumulate
// against the supply cap, and the signer needs the "issue" grant for every
// referenced type.
func TestIssueLicensesMultipleEntries(t *testing.T) {
	k, ms, ctx, owner := setupWithOwner(t)
	issuer := sample.AccAddress()
	holder1 := sample.AccAddress()
	holder2 := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "capped", MaxSupply: math.NewInt(5),
	})
	require.NoError(t, err)
	_, err = ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "open", MaxSupply: math.ZeroInt(),
	})
	require.NoError(t, err)
	_, err = ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "ungranted", MaxSupply: math.ZeroInt(),
	})
	require.NoError(t, err)

	_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: issuer,
		Grants: []types.AdminKeyGrant{{Permission: types.PermissionIssue, LicenseTypes: []string{"capped", "open"}}},
	})
	require.NoError(t, err)

	// Signer must hold the issue grant for every type referenced by the entries.
	_, err = ms.IssueLicenses(ctx, &types.MsgIssueLicenses{
		Issuer: issuer, Entries: []types.IssueLicenseEntry{
			{LicenseTypeId: "open", Holder: holder1, StartDate: "2026-01-01", Count: 1},
			{LicenseTypeId: "ungranted", Holder: holder1, StartDate: "2026-01-01", Count: 1},
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not have issue permission for license type ungranted")
	// Nothing was issued for the granted entry either.
	lt, _, err := k.GetLicenseType(ctx, "open")
	require.NoError(t, err)
	require.True(t, lt.IssuedCount.IsZero())

	// Counts for entries referencing the same type accumulate against the cap.
	_, err = ms.IssueLicenses(ctx, &types.MsgIssueLicenses{
		Issuer: issuer, Entries: []types.IssueLicenseEntry{
			{LicenseTypeId: "capped", Holder: holder1, StartDate: "2026-01-01", Count: 3},
			{LicenseTypeId: "capped", Holder: holder2, StartDate: "2026-01-01", Count: 3},
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceed max supply")
	lt, _, err = k.GetLicenseType(ctx, "capped")
	require.NoError(t, err)
	require.True(t, lt.IssuedCount.IsZero(), "failed batch must not issue anything")

	// Valid mixed batch across types and holders.
	resp, err := ms.IssueLicenses(ctx, &types.MsgIssueLicenses{
		Issuer: issuer, Entries: []types.IssueLicenseEntry{
			{LicenseTypeId: "capped", Holder: holder1, StartDate: "2026-01-01", EndDate: "2027-01-01", Count: 2},
			{LicenseTypeId: "capped", Holder: holder2, StartDate: "2026-02-01", Count: 3},
			{LicenseTypeId: "open", Holder: holder2, StartDate: "2026-03-01", Count: 1},
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.Ids, 6, "ids are flattened in entry order")

	// Per-type counters reflect the aggregate issuance.
	lt, _, err = k.GetLicenseType(ctx, "capped")
	require.NoError(t, err)
	require.Equal(t, math.NewInt(5), lt.IssuedCount)
	lt, _, err = k.GetLicenseType(ctx, "open")
	require.NoError(t, err)
	require.Equal(t, math.NewInt(1), lt.IssuedCount)

	// Each holder got the licenses from their entries.
	l, found, err := k.GetLicense(ctx, "capped", resp.Ids[0])
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, holder1, l.Holder)
	l, found, err = k.GetLicense(ctx, "capped", resp.Ids[2])
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, holder2, l.Holder)
	l, found, err = k.GetLicense(ctx, "open", resp.Ids[5])
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, holder2, l.Holder)
}

// ---------------------------------------------------------------------------
// RevokeLicenses
// ---------------------------------------------------------------------------

func TestRevokeLicenses(t *testing.T) {
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
		Grants: []types.AdminKeyGrant{{Permission: types.PermissionIssue, LicenseTypes: []string{"rev"}}},
	})
	require.NoError(t, err)
	_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: revoker,
		Grants: []types.AdminKeyGrant{{Permission: types.PermissionRevoke, LicenseTypes: []string{"rev"}}},
	})
	require.NoError(t, err)

	// Issue 3 licenses to the same holder.
	resp, err := ms.IssueLicenses(ctx, &types.MsgIssueLicenses{
		Issuer: issuer, Entries: []types.IssueLicenseEntry{
			{LicenseTypeId: "rev", Holder: holder, StartDate: "2026-01-01", Count: 3},
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.Ids, 3)

	tests := []struct {
		name      string
		input     *types.MsgRevokeLicenses
		expErr    bool
		expErrMsg string
	}{
		{
			name:      "no permission",
			input:     &types.MsgRevokeLicenses{Revoker: sample.AccAddress(), LicenseTypeId: "rev", Holder: holder, Count: 1},
			expErr:    true,
			expErrMsg: "does not have revoke permission",
		},
		{
			name:      "not enough active licenses",
			input:     &types.MsgRevokeLicenses{Revoker: revoker, LicenseTypeId: "rev", Holder: holder, Count: 10},
			expErr:    true,
			expErrMsg: "has 3 active license(s)",
		},
		{
			name:   "revoke 2 — most recent first",
			input:  &types.MsgRevokeLicenses{Revoker: revoker, LicenseTypeId: "rev", Holder: holder, Count: 2},
			expErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			revokeResp, err := ms.RevokeLicenses(ctx, tc.input)
			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.Len(t, revokeResp.Ids, 2)
				// Most recently issued (id=3) should be revoked first, then id=2.
				require.Equal(t, resp.Ids[2], revokeResp.Ids[0])
				require.Equal(t, resp.Ids[1], revokeResp.Ids[1])

				// Verify revoked licenses record revoked_date and keep their
				// issued end_date (empty here — none was set at issuance).
				for _, id := range revokeResp.Ids {
					license, found, _ := k.GetLicense(ctx, "rev", id)
					require.True(t, found)
					require.Equal(t, types.StatusRevoked, license.Status)
					require.NotEmpty(t, license.RevokedDate)
					require.Empty(t, license.EndDate, "end_date must not be overwritten by revocation")
				}

				// Verify the remaining license is still active.
				license, found, _ := k.GetLicense(ctx, "rev", resp.Ids[0])
				require.True(t, found)
				require.Equal(t, types.StatusActive, license.Status)

				// Verify counters.
				lt, _, _ := k.GetLicenseType(ctx, "rev")
				require.Equal(t, math.NewInt(3), lt.IssuedCount)
				require.Equal(t, math.NewInt(1), lt.ActiveCount)
				require.Equal(t, math.NewInt(2), lt.RevokedCount)
			}
		})
	}
}

// TestRevokeLicensesPreservesEndDate: revocation records revoked_date and
// leaves the issued end_date intact.
func TestRevokeLicensesPreservesEndDate(t *testing.T) {
	k, ms, ctx, owner := setupWithOwner(t)
	admin := sample.AccAddress()
	holder := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "ed", MaxSupply: math.ZeroInt(),
	})
	require.NoError(t, err)
	_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: admin,
		Grants: []types.AdminKeyGrant{
			{Permission: types.PermissionIssue, LicenseTypes: []string{"ed"}},
			{Permission: types.PermissionRevoke, LicenseTypes: []string{"ed"}},
		},
	})
	require.NoError(t, err)

	resp, err := ms.IssueLicenses(ctx, &types.MsgIssueLicenses{
		Issuer: admin, Entries: []types.IssueLicenseEntry{
			{LicenseTypeId: "ed", Holder: holder, StartDate: "2026-01-01", EndDate: "2027-01-01", Count: 1},
		},
	})
	require.NoError(t, err)

	_, err = ms.RevokeLicenses(ctx, &types.MsgRevokeLicenses{
		Revoker: admin, LicenseTypeId: "ed", Holder: holder, Count: 1,
	})
	require.NoError(t, err)

	l, found, err := k.GetLicense(ctx, "ed", resp.Ids[0])
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, types.StatusRevoked, l.Status)
	require.Equal(t, "2027-01-01", l.EndDate, "issued end_date must survive revocation")
	require.NotEmpty(t, l.RevokedDate)
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
		Grants: []types.AdminKeyGrant{{Permission: types.PermissionIssue, LicenseTypes: []string{"xfer", "noxfer"}}},
	})
	require.NoError(t, err)

	resp, err := ms.IssueLicenses(ctx, &types.MsgIssueLicenses{
		Issuer: issuer, Entries: []types.IssueLicenseEntry{
			{LicenseTypeId: "xfer", Holder: holder, StartDate: "2026-01-01", Count: 1},
		},
	})
	require.NoError(t, err)
	xferID := resp.Ids[0]

	resp, err = ms.IssueLicenses(ctx, &types.MsgIssueLicenses{
		Issuer: issuer, Entries: []types.IssueLicenseEntry{
			{LicenseTypeId: "noxfer", Holder: holder, StartDate: "2026-01-01", Count: 1},
		},
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

// TestIssueLicensesSupplyCheckIsUnsigned guards the supply-cap arithmetic:
// a count with the high bit set must not wrap negative and silently bypass
// the MaxSupply check.
func TestIssueLicensesSupplyCheckIsUnsigned(t *testing.T) {
	_, ms, ctx, owner := setupWithOwner(t)
	issuer := sample.AccAddress()
	holder := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "lim", Transferrable: false, MaxSupply: math.NewInt(100),
	})
	require.NoError(t, err)
	_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: issuer,
		Grants: []types.AdminKeyGrant{{Permission: types.PermissionIssue, LicenseTypes: []string{"lim"}}},
	})
	require.NoError(t, err)

	// 1<<63 is the smallest uint64 value that wraps to a negative int64.
	_, err = ms.IssueLicenses(ctx, &types.MsgIssueLicenses{
		Issuer: issuer, Entries: []types.IssueLicenseEntry{
			{LicenseTypeId: "lim", Holder: holder, StartDate: "2026-01-01", Count: 1 << 63},
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceed max supply")
}

// TestGrantAdminPermissionsGrantsCap rejects an over-cap Grants slice.
func TestGrantAdminPermissionsGrantsCap(t *testing.T) {
	_, ms, ctx, owner := setupWithOwner(t)
	adminAddr := sample.AccAddress()

	grants := make([]types.AdminKeyGrant, types.MaxAdminGrants+1)
	for i := range grants {
		grants[i] = types.AdminKeyGrant{Permission: types.PermissionIssue, LicenseTypes: []string{"t1"}}
	}

	_, err := ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: adminAddr, Grants: grants,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "grants length")
}

// TestGrantAdminPermissionsLicenseTypesCap rejects an over-cap inner
// LicenseTypes slice within a single grant.
func TestGrantAdminPermissionsLicenseTypesCap(t *testing.T) {
	_, ms, ctx, owner := setupWithOwner(t)
	adminAddr := sample.AccAddress()

	lts := make([]string, types.MaxAdminGrants+1)
	for i := range lts {
		lts[i] = "t1"
	}

	_, err := ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner:   owner,
		Address: adminAddr,
		Grants:  []types.AdminKeyGrant{{Permission: types.PermissionIssue, LicenseTypes: lts}},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "license_types length")
}

// TestRevokeAdminKeyPermissionsCap rejects an over-cap Permissions slice.
func TestRevokeAdminKeyPermissionsCap(t *testing.T) {
	_, ms, ctx, owner := setupWithOwner(t)
	adminAddr := sample.AccAddress()

	perms := make([]types.AdminKeyPermission, types.MaxAdminGrants+1)
	for i := range perms {
		perms[i] = types.AdminKeyPermission{LicenseTypeId: "t1", Permission: types.PermissionIssue}
	}

	_, err := ms.RevokeAdminKeyPermissions(ctx, &types.MsgRevokeAdminKeyPermissions{
		Owner: owner, Address: adminAddr, Permissions: perms,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "permissions length")
}

// TestIssueLicensesEntriesCap ensures IssueLicenses rejects entry lists
// larger than MaxIssueBatchSize.
func TestIssueLicensesEntriesCap(t *testing.T) {
	_, ms, ctx, owner := setupWithOwner(t)
	issuer := sample.AccAddress()
	holder := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "cap", Transferrable: false, MaxSupply: math.ZeroInt(),
	})
	require.NoError(t, err)
	_, err = ms.GrantAdminPermissions(ctx, &types.MsgGrantAdminPermissions{
		Owner: owner, Address: issuer,
		Grants: []types.AdminKeyGrant{{Permission: types.PermissionIssue, LicenseTypes: []string{"cap"}}},
	})
	require.NoError(t, err)

	entries := make([]types.IssueLicenseEntry, types.MaxIssueBatchSize+1)
	for i := range entries {
		entries[i] = types.IssueLicenseEntry{LicenseTypeId: "cap", Holder: holder, StartDate: "2026-01-01", Count: 1}
	}

	_, err = ms.IssueLicenses(ctx, &types.MsgIssueLicenses{
		Issuer: issuer, Entries: entries,
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
			{Permission: types.PermissionIssue, LicenseTypes: []string{"xfer"}},
			{Permission: types.PermissionRevoke, LicenseTypes: []string{"xfer"}},
		},
	})
	require.NoError(t, err)

	resp, err := ms.IssueLicenses(ctx, &types.MsgIssueLicenses{
		Issuer: issuer, Entries: []types.IssueLicenseEntry{
			{LicenseTypeId: "xfer", Holder: holder, StartDate: "2026-01-01", Count: 1},
		},
	})
	require.NoError(t, err)
	id := resp.Ids[0]

	_, err = ms.RevokeLicenses(ctx, &types.MsgRevokeLicenses{
		Revoker: issuer, LicenseTypeId: "xfer", Holder: holder, Count: 1,
	})
	require.NoError(t, err)

	// Sanity: the license entry still exists with status=revoked.
	l, found, err := k.GetLicense(ctx, "xfer", id)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, types.StatusRevoked, l.Status)
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
		Grants: []types.AdminKeyGrant{{Permission: types.PermissionIssue, LicenseTypes: []string{"lt1"}}},
	})
	require.NoError(t, err)
	_, err = ms.IssueLicenses(ctx, &types.MsgIssueLicenses{
		Issuer: issuer, Entries: []types.IssueLicenseEntry{
			{LicenseTypeId: "lt1", Holder: sample.AccAddress(), StartDate: "2026-01-01", Count: 5},
		},
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
