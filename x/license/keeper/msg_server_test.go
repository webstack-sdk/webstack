package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/webstack-sdk/webstack/testutil/keeper"
	"github.com/webstack-sdk/webstack/testutil/sample"
	"github.com/webstack-sdk/webstack/x/license/keeper"
	"github.com/webstack-sdk/webstack/x/license/types"
)

// setupWithOwner returns a license fixture, its msg server, context, and the
// license namespace owner. Permission grants are written through
// LicenseFixture.Grant — the owner-gated grant path itself lives in (and is
// tested by) the x/permission module.
func setupWithOwner(t testing.TB) (*keepertest.LicenseFixture, types.MsgServer, sdk.Context, string) {
	t.Helper()
	f := keepertest.NewLicenseFixture(t)
	return f, keeper.NewMsgServerImpl(f.Keeper), f.Ctx, f.Owner
}

func TestMsgServer(t *testing.T) {
	f, ms, ctx, _ := setupWithOwner(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotEmpty(t, f.Keeper)
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
			expErrMsg: "not the license namespace owner",
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
// IssueLicenses
// ---------------------------------------------------------------------------

func TestIssueLicenses(t *testing.T) {
	f, ms, ctx, owner := setupWithOwner(t)
	issuer := sample.AccAddress()
	holder := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "node", MaxSupply: math.NewInt(10),
	})
	require.NoError(t, err)

	f.Grant(t, issuer, types.PermissionIssue, "node")

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

	lt, found, err := f.Keeper.GetLicenseType(ctx, "node")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, math.NewInt(4), lt.IssuedCount)
}

// TestIssueLicensesMultipleEntries covers the multi-entry behavior: entries
// can target different holders and license types, per-entry counts accumulate
// against the supply cap, and the signer needs the "issue" grant for every
// referenced type.
func TestIssueLicensesMultipleEntries(t *testing.T) {
	f, ms, ctx, owner := setupWithOwner(t)
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

	f.Grant(t, issuer, types.PermissionIssue, "capped")
	f.Grant(t, issuer, types.PermissionIssue, "open")

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
	lt, _, err := f.Keeper.GetLicenseType(ctx, "open")
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
	lt, _, err = f.Keeper.GetLicenseType(ctx, "capped")
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
	lt, _, err = f.Keeper.GetLicenseType(ctx, "capped")
	require.NoError(t, err)
	require.Equal(t, math.NewInt(5), lt.IssuedCount)
	lt, _, err = f.Keeper.GetLicenseType(ctx, "open")
	require.NoError(t, err)
	require.Equal(t, math.NewInt(1), lt.IssuedCount)

	// Each holder got the licenses from their entries.
	l, found, err := f.Keeper.GetLicense(ctx, "capped", resp.Ids[0])
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, holder1, l.Holder)
	l, found, err = f.Keeper.GetLicense(ctx, "capped", resp.Ids[2])
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, holder2, l.Holder)
	l, found, err = f.Keeper.GetLicense(ctx, "open", resp.Ids[5])
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, holder2, l.Holder)
}

// ---------------------------------------------------------------------------
// RevokeLicenses
// ---------------------------------------------------------------------------

func TestRevokeLicenses(t *testing.T) {
	f, ms, ctx, owner := setupWithOwner(t)
	issuer := sample.AccAddress()
	revoker := sample.AccAddress()
	holder := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "rev", MaxSupply: math.ZeroInt(),
	})
	require.NoError(t, err)
	f.Grant(t, issuer, types.PermissionIssue, "rev")
	f.Grant(t, revoker, types.PermissionRevoke, "rev")

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
					license, found, _ := f.Keeper.GetLicense(ctx, "rev", id)
					require.True(t, found)
					require.Equal(t, types.StatusRevoked, license.Status)
					require.NotEmpty(t, license.RevokedDate)
					require.Empty(t, license.EndDate, "end_date must not be overwritten by revocation")
				}

				// Verify the remaining license is still active.
				license, found, _ := f.Keeper.GetLicense(ctx, "rev", resp.Ids[0])
				require.True(t, found)
				require.Equal(t, types.StatusActive, license.Status)

				// Verify counters.
				lt, _, _ := f.Keeper.GetLicenseType(ctx, "rev")
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
	f, ms, ctx, owner := setupWithOwner(t)
	admin := sample.AccAddress()
	holder := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "ed", MaxSupply: math.ZeroInt(),
	})
	require.NoError(t, err)
	f.Grant(t, admin, types.PermissionIssue, "ed")
	f.Grant(t, admin, types.PermissionRevoke, "ed")

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

	l, found, err := f.Keeper.GetLicense(ctx, "ed", resp.Ids[0])
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
	f, ms, ctx, owner := setupWithOwner(t)
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

	f.Grant(t, issuer, types.PermissionIssue, "xfer")
	f.Grant(t, issuer, types.PermissionIssue, "noxfer")

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
				l, found, _ := f.Keeper.GetLicense(ctx, "xfer", xferID)
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
	f, ms, ctx, owner := setupWithOwner(t)
	issuer := sample.AccAddress()
	holder := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "lim", Transferrable: false, MaxSupply: math.NewInt(100),
	})
	require.NoError(t, err)
	f.Grant(t, issuer, types.PermissionIssue, "lim")

	// 1<<63 is the smallest uint64 value that wraps to a negative int64.
	_, err = ms.IssueLicenses(ctx, &types.MsgIssueLicenses{
		Issuer: issuer, Entries: []types.IssueLicenseEntry{
			{LicenseTypeId: "lim", Holder: holder, StartDate: "2026-01-01", Count: 1 << 63},
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceed max supply")
}

// TestIssueLicensesEntriesCap ensures IssueLicenses rejects entry lists
// larger than MaxIssueBatchSize.
func TestIssueLicensesEntriesCap(t *testing.T) {
	f, ms, ctx, owner := setupWithOwner(t)
	issuer := sample.AccAddress()
	holder := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "cap", Transferrable: false, MaxSupply: math.ZeroInt(),
	})
	require.NoError(t, err)
	f.Grant(t, issuer, types.PermissionIssue, "cap")

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
// transferable, even though the license entry still exists under the
// original holder.
func TestTransferLicenseRejectsRevoked(t *testing.T) {
	f, ms, ctx, owner := setupWithOwner(t)
	issuer := sample.AccAddress()
	holder := sample.AccAddress()
	recipient := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "xfer", Transferrable: true, MaxSupply: math.ZeroInt(),
	})
	require.NoError(t, err)

	f.Grant(t, issuer, types.PermissionIssue, "xfer")
	f.Grant(t, issuer, types.PermissionRevoke, "xfer")

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
	l, found, err := f.Keeper.GetLicense(ctx, "xfer", id)
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
	l, _, err = f.Keeper.GetLicense(ctx, "xfer", id)
	require.NoError(t, err)
	require.Equal(t, holder, l.Holder)
}

// ---------------------------------------------------------------------------
// UpdateLicenseType
// ---------------------------------------------------------------------------

func TestUpdateLicenseType(t *testing.T) {
	f, ms, ctx, owner := setupWithOwner(t)
	issuer := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "lt1", Transferrable: false, MaxSupply: math.NewInt(100),
	})
	require.NoError(t, err)
	f.Grant(t, issuer, types.PermissionIssue, "lt1")
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
			expErrMsg: "not the license namespace owner",
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
