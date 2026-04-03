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
// SetAdminKey
// ---------------------------------------------------------------------------

func TestSetAdminKey(t *testing.T) {
	k, ms, ctx, owner := setupWithOwner(t)
	adminAddr := sample.AccAddress()

	tests := []struct {
		name      string
		input     *types.MsgSetAdminKey
		expErr    bool
		expErrMsg string
	}{
		{
			name: "non-owner",
			input: &types.MsgSetAdminKey{
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
			input: &types.MsgSetAdminKey{
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
			input: &types.MsgSetAdminKey{
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
			input: &types.MsgSetAdminKey{
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
			input: &types.MsgSetAdminKey{
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
			_, err := ms.SetAdminKey(ctx, tc.input)
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

// ---------------------------------------------------------------------------
// RemoveAdminKey
// ---------------------------------------------------------------------------

func TestRemoveAdminKey(t *testing.T) {
	k, ms, ctx, owner := setupWithOwner(t)
	adminAddr := sample.AccAddress()

	_, err := ms.SetAdminKey(ctx, &types.MsgSetAdminKey{
		Owner:   owner,
		Address: adminAddr,
		Grants:  []types.AdminKeyGrant{{Permission: "issue", LicenseTypes: []string{"t1"}}},
	})
	require.NoError(t, err)
	require.True(t, k.HasPermission(ctx, adminAddr, "issue", "t1"))

	tests := []struct {
		name      string
		input     *types.MsgRemoveAdminKey
		expErr    bool
		expErrMsg string
	}{
		{
			name: "non-owner",
			input: &types.MsgRemoveAdminKey{
				Owner:   sample.AccAddress(),
				Address: adminAddr,
			},
			expErr:    true,
			expErrMsg: "not the module owner",
		},
		{
			name: "valid",
			input: &types.MsgRemoveAdminKey{
				Owner:   owner,
				Address: adminAddr,
			},
			expErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.RemoveAdminKey(ctx, tc.input)
			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.False(t, k.HasPermission(ctx, adminAddr, "issue", "t1"))
			}
		})
	}
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

	_, err = ms.SetAdminKey(ctx, &types.MsgSetAdminKey{
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

	_, err = ms.SetAdminKey(ctx, &types.MsgSetAdminKey{
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
	_, err = ms.SetAdminKey(ctx, &types.MsgSetAdminKey{
		Owner: owner, Address: issuer,
		Grants: []types.AdminKeyGrant{{Permission: "issue", LicenseTypes: []string{"rev"}}},
	})
	require.NoError(t, err)
	_, err = ms.SetAdminKey(ctx, &types.MsgSetAdminKey{
		Owner: owner, Address: revoker,
		Grants: []types.AdminKeyGrant{{Permission: "revoke", LicenseTypes: []string{"rev"}}},
	})
	require.NoError(t, err)

	resp, err := ms.IssueLicense(ctx, &types.MsgIssueLicense{
		Issuer: issuer, LicenseTypeId: "rev", Holder: holder, StartDate: "2026-01-01",
	})
	require.NoError(t, err)
	licenseID := resp.Ids[0]

	tests := []struct {
		name      string
		input     *types.MsgRevokeLicense
		expErr    bool
		expErrMsg string
	}{
		{
			name:      "not found",
			input:     &types.MsgRevokeLicense{Revoker: revoker, LicenseTypeId: "rev", Id: 999},
			expErr:    true,
			expErrMsg: "not found",
		},
		{
			name:      "no permission",
			input:     &types.MsgRevokeLicense{Revoker: sample.AccAddress(), LicenseTypeId: "rev", Id: licenseID},
			expErr:    true,
			expErrMsg: "does not have revoke permission",
		},
		{
			name:   "valid",
			input:  &types.MsgRevokeLicense{Revoker: revoker, LicenseTypeId: "rev", Id: licenseID},
			expErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.RevokeLicense(ctx, tc.input)
			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				_, found, _ := k.GetLicense(ctx, "rev", licenseID)
				require.False(t, found)
				lt, _, _ := k.GetLicenseType(ctx, "rev")
				require.True(t, lt.IssuedCount.IsZero())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateLicense
// ---------------------------------------------------------------------------

func TestUpdateLicense(t *testing.T) {
	k, ms, ctx, owner := setupWithOwner(t)
	issuer := sample.AccAddress()
	updater := sample.AccAddress()
	holder := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "upd", MaxSupply: math.ZeroInt(),
	})
	require.NoError(t, err)
	_, err = ms.SetAdminKey(ctx, &types.MsgSetAdminKey{
		Owner: owner, Address: issuer,
		Grants: []types.AdminKeyGrant{{Permission: "issue", LicenseTypes: []string{"upd"}}},
	})
	require.NoError(t, err)
	_, err = ms.SetAdminKey(ctx, &types.MsgSetAdminKey{
		Owner: owner, Address: updater,
		Grants: []types.AdminKeyGrant{{Permission: "update", LicenseTypes: []string{"upd"}}},
	})
	require.NoError(t, err)

	resp, err := ms.IssueLicense(ctx, &types.MsgIssueLicense{
		Issuer: issuer, LicenseTypeId: "upd", Holder: holder, StartDate: "2026-01-01",
	})
	require.NoError(t, err)
	licenseID := resp.Ids[0]

	tests := []struct {
		name      string
		input     *types.MsgUpdateLicense
		expErr    bool
		expErrMsg string
	}{
		{
			name:      "not found",
			input:     &types.MsgUpdateLicense{Updater: updater, LicenseTypeId: "upd", Id: 999, Status: "revoked"},
			expErr:    true,
			expErrMsg: "not found",
		},
		{
			name:      "no permission",
			input:     &types.MsgUpdateLicense{Updater: sample.AccAddress(), LicenseTypeId: "upd", Id: licenseID, Status: "revoked"},
			expErr:    true,
			expErrMsg: "does not have update permission",
		},
		{
			name:      "invalid status",
			input:     &types.MsgUpdateLicense{Updater: updater, LicenseTypeId: "upd", Id: licenseID, Status: "suspended"},
			expErr:    true,
			expErrMsg: "invalid status",
		},
		{
			name:   "valid",
			input:  &types.MsgUpdateLicense{Updater: updater, LicenseTypeId: "upd", Id: licenseID, Status: "revoked"},
			expErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.UpdateLicense(ctx, tc.input)
			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				l, found, _ := k.GetLicense(ctx, "upd", licenseID)
				require.True(t, found)
				require.Equal(t, "revoked", l.Status)
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

	_, err = ms.SetAdminKey(ctx, &types.MsgSetAdminKey{
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
	_, err = ms.SetAdminKey(ctx, &types.MsgSetAdminKey{
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
