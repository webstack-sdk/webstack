package licensesprecompile

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"

	licensestypes "github.com/webstack-sdk/webstack/x/licenses/types"
)

// grantIssue adds admin grants for `address` directly via the keeper so tx tests
// can focus on the precompile-side behaviour rather than chaining grantPermissions.
func grantIssue(t *testing.T, f *testFixture, addressBech, licenseTypeID string) {
	t.Helper()
	require.NoError(t, f.keeper.Permissions.Set(f.ctx, collections.Join3(addressBech, int32(licensestypes.PermissionIssue), licenseTypeID)))
	require.NoError(t, f.keeper.Permissions.Set(f.ctx, collections.Join3(addressBech, int32(licensestypes.PermissionRevoke), licenseTypeID)))
}

// issueOne is a convenience that issues a single license through the precompile.
func issueOne(t *testing.T, f *testFixture, issuerHex common.Address, licenseTypeID string, holderHex common.Address, startDate, endDate string) []uint64 {
	t.Helper()
	method := ABI.Methods[IssueLicensesMethod]
	bz, err := f.precompile.IssueLicenses(
		f.ctx,
		f.newContract(issuerHex),
		f.stateDB,
		&method,
		[]interface{}{[]IssueLicenseEntryArg{
			{LicenseTypeId: licenseTypeID, Holder: holderHex, StartDate: startDate, EndDate: endDate, Count: 1},
		}},
	)
	require.NoError(t, err)

	out, err := method.Outputs.Unpack(bz)
	require.NoError(t, err)
	return out[0].([]uint64)
}

func TestTxCreateLicenseType(t *testing.T) {
	f := newTestFixture(t)
	method := ABI.Methods[CreateLicenseTypeMethod]

	// happy path — caller is the module owner
	bz, err := f.precompile.CreateLicenseType(
		f.ctx,
		f.newContract(f.OwnerHex),
		f.stateDB,
		&method,
		[]interface{}{"type.a", true, big.NewInt(100)},
	)
	require.NoError(t, err)

	// success bool decoded correctly
	out, err := method.Outputs.Unpack(bz)
	require.NoError(t, err)
	require.True(t, out[0].(bool))

	// keeper actually persisted the license type
	lt, err := f.keeper.LicenseTypes.Get(f.ctx, "type.a")
	require.NoError(t, err)
	require.True(t, lt.Transferrable)
	require.Equal(t, "100", lt.MaxSupply.String())

	// exactly one log emitted (LicenseTypeCreated)
	require.Len(t, f.stateDB.logs, 1)
	require.Equal(t, ABI.Events[EventTypeLicenseTypeCreated].ID, f.stateDB.logs[0].Topics[0])
	require.Equal(t, f.precompile.Address(), f.stateDB.logs[0].Address)

	// non-owner caller is rejected by the keeper
	notOwner := common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	_, err = f.precompile.CreateLicenseType(
		f.ctx,
		f.newContract(notOwner),
		f.stateDB,
		&method,
		[]interface{}{"type.b", false, big.NewInt(0)},
	)
	require.Error(t, err)
}

func TestTxUpdateLicenseType(t *testing.T) {
	f := newTestFixture(t)
	seedLicenseType(t, f, "type.a", true, 100)

	method := ABI.Methods[UpdateLicenseTypeMethod]
	bz, err := f.precompile.UpdateLicenseType(
		f.ctx,
		f.newContract(f.OwnerHex),
		f.stateDB,
		&method,
		[]interface{}{"type.a", false, big.NewInt(200)},
	)
	require.NoError(t, err)
	out, err := method.Outputs.Unpack(bz)
	require.NoError(t, err)
	require.True(t, out[0].(bool))

	lt, err := f.keeper.LicenseTypes.Get(f.ctx, "type.a")
	require.NoError(t, err)
	require.False(t, lt.Transferrable)
	require.Equal(t, "200", lt.MaxSupply.String())
}

func TestTxGrantAndRevokeAdminPermissions(t *testing.T) {
	f := newTestFixture(t)
	seedLicenseType(t, f, "type.a", true, 0)
	seedLicenseType(t, f, "type.b", true, 0)

	adminHex := common.HexToAddress("0x4444444444444444444444444444444444444444")
	adminBech := f.bechFor(t, adminHex)

	// grantPermissions --------------------------------------------
	grantM := ABI.Methods[GrantPermissionsMethod]
	grants := []PermissionGrantArg{
		{Permission: "issue", LicenseTypes: []string{"type.a", "type.b"}},
		{Permission: "revoke", LicenseTypes: []string{"type.a"}},
	}
	bz, err := f.precompile.GrantPermissions(
		f.ctx,
		f.newContract(f.OwnerHex),
		f.stateDB,
		&grantM,
		[]interface{}{adminHex, grants},
	)
	require.NoError(t, err)
	out, err := grantM.Outputs.Unpack(bz)
	require.NoError(t, err)
	require.True(t, out[0].(bool))

	ak, found, err := f.keeper.GetPermissionsByAddress(f.ctx, adminBech)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, adminBech, ak.Address)
	require.Len(t, ak.Grants, 2)

	// PermissionsGranted event emitted
	require.NotEmpty(t, f.stateDB.logs)
	require.Equal(t, ABI.Events[EventTypePermissionsGranted].ID, f.stateDB.logs[len(f.stateDB.logs)-1].Topics[0])

	// A second grant call merges rather than replaces.
	moreGrants := []PermissionGrantArg{
		{Permission: "revoke", LicenseTypes: []string{"type.b"}},
	}
	_, err = f.precompile.GrantPermissions(
		f.ctx,
		f.newContract(f.OwnerHex),
		f.stateDB,
		&grantM,
		[]interface{}{adminHex, moreGrants},
	)
	require.NoError(t, err)
	require.True(t, f.keeper.HasPermission(f.ctx, adminBech, licensestypes.PermissionIssue, "type.a"))
	require.True(t, f.keeper.HasPermission(f.ctx, adminBech, licensestypes.PermissionRevoke, "type.a"))
	require.True(t, f.keeper.HasPermission(f.ctx, adminBech, licensestypes.PermissionRevoke, "type.b"))

	// revokePermissions ----------------------------------------
	// Remove only (type.a, revoke). type.a:issue, type.b:issue, type.b:revoke stay.
	revM := ABI.Methods[RevokePermissionsMethod]
	pairs := []PermissionPairArg{
		{LicenseTypeId: "type.a", Permission: "revoke"},
	}
	bz, err = f.precompile.RevokePermissions(
		f.ctx,
		f.newContract(f.OwnerHex),
		f.stateDB,
		&revM,
		[]interface{}{adminHex, pairs},
	)
	require.NoError(t, err)
	out, err = revM.Outputs.Unpack(bz)
	require.NoError(t, err)
	require.True(t, out[0].(bool))

	require.False(t, f.keeper.HasPermission(f.ctx, adminBech, licensestypes.PermissionRevoke, "type.a"))
	require.True(t, f.keeper.HasPermission(f.ctx, adminBech, licensestypes.PermissionIssue, "type.a"))
	require.True(t, f.keeper.HasPermission(f.ctx, adminBech, licensestypes.PermissionRevoke, "type.b"))
	require.Equal(t, ABI.Events[EventTypePermissionsRevoked].ID, f.stateDB.logs[len(f.stateDB.logs)-1].Topics[0])

	// Revoking every remaining pair deletes the permissions entry.
	pairs = []PermissionPairArg{
		{LicenseTypeId: "type.a", Permission: "issue"},
		{LicenseTypeId: "type.b", Permission: "issue"},
		{LicenseTypeId: "type.b", Permission: "revoke"},
	}
	_, err = f.precompile.RevokePermissions(
		f.ctx,
		f.newContract(f.OwnerHex),
		f.stateDB,
		&revM,
		[]interface{}{adminHex, pairs},
	)
	require.NoError(t, err)

	_, found, err = f.keeper.GetPermissionsByAddress(f.ctx, adminBech)
	require.NoError(t, err)
	require.False(t, found, "permissions entry should be gone when no grants remain")
}

// TestTxGrantPermissionsUnknownPermission: the precompile parses the
// Solidity permission string at the boundary and rejects unknown values.
func TestTxGrantPermissionsUnknownPermission(t *testing.T) {
	f := newTestFixture(t)
	seedLicenseType(t, f, "type.a", true, 0)

	grantM := ABI.Methods[GrantPermissionsMethod]
	adminHex := common.HexToAddress("0x4444444444444444444444444444444444444444")
	grants := []PermissionGrantArg{{Permission: "destroy", LicenseTypes: []string{"type.a"}}}
	_, err := f.precompile.GrantPermissions(
		f.ctx,
		f.newContract(f.OwnerHex),
		f.stateDB,
		&grantM,
		[]interface{}{adminHex, grants},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid permission")
}

func TestTxIssueLicenses(t *testing.T) {
	f := newTestFixture(t)
	seedLicenseType(t, f, "type.a", true, 0)
	grantIssue(t, f, f.OwnerBech, "type.a")

	holderHex := common.HexToAddress("0x5555555555555555555555555555555555555555")
	method := ABI.Methods[IssueLicensesMethod]

	bz, err := f.precompile.IssueLicenses(
		f.ctx,
		f.newContract(f.OwnerHex),
		f.stateDB,
		&method,
		[]interface{}{[]IssueLicenseEntryArg{
			{LicenseTypeId: "type.a", Holder: holderHex, StartDate: "2025-01-01", EndDate: "", Count: 3},
		}},
	)
	require.NoError(t, err)
	out, err := method.Outputs.Unpack(bz)
	require.NoError(t, err)
	ids := out[0].([]uint64)
	require.Equal(t, []uint64{1, 2, 3}, ids)

	lt, err := f.keeper.LicenseTypes.Get(f.ctx, "type.a")
	require.NoError(t, err)
	require.Equal(t, "3", lt.IssuedCount.String())
	require.Equal(t, "3", lt.ActiveCount.String())

	// LicenseIssued event with the right indexed topics
	last := f.stateDB.logs[len(f.stateDB.logs)-1]
	require.Equal(t, ABI.Events[EventTypeLicenseIssued].ID, last.Topics[0])
	// topic[1] = issuer (left-padded)
	require.Equal(t, common.LeftPadBytes(f.OwnerHex.Bytes(), 32), last.Topics[1].Bytes())
	require.Equal(t, common.LeftPadBytes(holderHex.Bytes(), 32), last.Topics[2].Bytes())
}

// TestTxIssueLicensesRunsValidateBasic verifies that the precompile path
// invokes the SDK message's ValidateBasic — which an empty license_type_id
// must surface as ErrEmptyLicenseTypeID rather than the keeper's downstream
// "no permission" path that would fire if ValidateBasic were skipped.
func TestTxIssueLicensesRunsValidateBasic(t *testing.T) {
	f := newTestFixture(t)
	seedLicenseType(t, f, "type.a", true, 0)
	grantIssue(t, f, f.OwnerBech, "type.a")

	method := ABI.Methods[IssueLicensesMethod]
	_, err := f.precompile.IssueLicenses(
		f.ctx,
		f.newContract(f.OwnerHex),
		f.stateDB,
		&method,
		[]interface{}{[]IssueLicenseEntryArg{
			{LicenseTypeId: "", Holder: common.HexToAddress("0x5555555555555555555555555555555555555555"), StartDate: "2025-01-01", EndDate: "", Count: 1},
		}},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "license type id cannot be empty")
}

func TestTxIssueLicensesWithoutPermission(t *testing.T) {
	f := newTestFixture(t)
	seedLicenseType(t, f, "type.a", true, 0)
	// owner has no admin grants for type.a

	method := ABI.Methods[IssueLicensesMethod]
	_, err := f.precompile.IssueLicenses(
		f.ctx,
		f.newContract(f.OwnerHex),
		f.stateDB,
		&method,
		[]interface{}{[]IssueLicenseEntryArg{
			{LicenseTypeId: "type.a", Holder: common.HexToAddress("0x5555555555555555555555555555555555555555"), StartDate: "2025-01-01", EndDate: "", Count: 1},
		}},
	)
	require.Error(t, err)
}

func TestTxRevokeLicenses(t *testing.T) {
	f := newTestFixture(t)
	seedLicenseType(t, f, "type.a", true, 0)
	grantIssue(t, f, f.OwnerBech, "type.a")

	holderHex := common.HexToAddress("0x6666666666666666666666666666666666666666")
	issueOne(t, f, f.OwnerHex, "type.a", holderHex, "2025-01-01", "")
	issueOne(t, f, f.OwnerHex, "type.a", holderHex, "2025-01-01", "")

	method := ABI.Methods[RevokeLicensesMethod]
	bz, err := f.precompile.RevokeLicenses(
		f.ctx,
		f.newContract(f.OwnerHex),
		f.stateDB,
		&method,
		[]interface{}{"type.a", holderHex, uint64(1)},
	)
	require.NoError(t, err)

	out, err := method.Outputs.Unpack(bz)
	require.NoError(t, err)
	ids := out[0].([]uint64)
	require.Len(t, ids, 1)

	lt, err := f.keeper.LicenseTypes.Get(f.ctx, "type.a")
	require.NoError(t, err)
	require.Equal(t, "2", lt.IssuedCount.String(), "issued count should not decrease on revoke")
	require.Equal(t, "1", lt.ActiveCount.String())
	require.Equal(t, "1", lt.RevokedCount.String())
}

func TestTxTransferLicense(t *testing.T) {
	f := newTestFixture(t)
	seedLicenseType(t, f, "type.a", true, 0)
	grantIssue(t, f, f.OwnerBech, "type.a")

	holderHex := common.HexToAddress("0x7777777777777777777777777777777777777777")
	recipientHex := common.HexToAddress("0x8888888888888888888888888888888888888888")
	ids := issueOne(t, f, f.OwnerHex, "type.a", holderHex, "2025-01-01", "")
	require.Len(t, ids, 1)

	method := ABI.Methods[TransferLicenseMethod]
	bz, err := f.precompile.TransferLicense(
		f.ctx,
		f.newContract(holderHex),
		f.stateDB,
		&method,
		[]interface{}{"type.a", ids[0], recipientHex},
	)
	require.NoError(t, err)
	out, err := method.Outputs.Unpack(bz)
	require.NoError(t, err)
	require.True(t, out[0].(bool))

	// Verify the on-chain license now points at the recipient.
	licM := ABI.Methods[LicenseMethod]
	bz, err = f.precompile.License(f.ctx, &licM, []interface{}{"type.a", ids[0]})
	require.NoError(t, err)
	licOut, err := licM.Outputs.Unpack(bz)
	require.NoError(t, err)
	got := licOut[0].(struct {
		Id          uint64         `json:"id"`
		TypeId      string         `json:"typeId"`
		Holder      common.Address `json:"holder"`
		StartDate   string         `json:"startDate"`
		EndDate     string         `json:"endDate"`
		Status      string         `json:"status"`
		RevokedDate string         `json:"revokedDate"`
	})
	require.Equal(t, recipientHex, got.Holder)
}

func TestTxTransferLicenseNonHolderRejected(t *testing.T) {
	f := newTestFixture(t)
	seedLicenseType(t, f, "type.a", true, 0)
	grantIssue(t, f, f.OwnerBech, "type.a")

	holderHex := common.HexToAddress("0x7777777777777777777777777777777777777777")
	attackerHex := common.HexToAddress("0x9999999999999999999999999999999999999999")
	recipientHex := common.HexToAddress("0x8888888888888888888888888888888888888888")
	ids := issueOne(t, f, f.OwnerHex, "type.a", holderHex, "2025-01-01", "")

	method := ABI.Methods[TransferLicenseMethod]
	_, err := f.precompile.TransferLicense(
		f.ctx,
		f.newContract(attackerHex),
		f.stateDB,
		&method,
		[]interface{}{"type.a", ids[0], recipientHex},
	)
	require.Error(t, err, "non-holder should not be able to transfer")
}

// TestTxIssueLicensesMultipleEntries issues to multiple holders across
// multiple license types in one call and expects one LicenseIssued event
// per entry.
func TestTxIssueLicensesMultipleEntries(t *testing.T) {
	f := newTestFixture(t)
	seedLicenseType(t, f, "type.a", true, 0)
	seedLicenseType(t, f, "type.b", true, 0)
	require.NoError(t, f.keeper.Permissions.Set(f.ctx, collections.Join3(f.OwnerBech, int32(licensestypes.PermissionIssue), "type.a")))
	require.NoError(t, f.keeper.Permissions.Set(f.ctx, collections.Join3(f.OwnerBech, int32(licensestypes.PermissionIssue), "type.b")))

	holderA := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	holderB := common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	entries := []IssueLicenseEntryArg{
		{LicenseTypeId: "type.a", Holder: holderA, StartDate: "2025-01-01", EndDate: "", Count: 1},
		{LicenseTypeId: "type.a", Holder: holderB, StartDate: "2025-02-01", EndDate: "2026-02-01", Count: 2},
		{LicenseTypeId: "type.b", Holder: holderB, StartDate: "2025-03-01", EndDate: "", Count: 1},
	}

	method := ABI.Methods[IssueLicensesMethod]
	bz, err := f.precompile.IssueLicenses(
		f.ctx,
		f.newContract(f.OwnerHex),
		f.stateDB,
		&method,
		[]interface{}{entries},
	)
	require.NoError(t, err)
	out, err := method.Outputs.Unpack(bz)
	require.NoError(t, err)
	ids := out[0].([]uint64)
	require.Len(t, ids, 4, "ids flattened in entry order")

	lt, err := f.keeper.LicenseTypes.Get(f.ctx, "type.a")
	require.NoError(t, err)
	require.Equal(t, "3", lt.IssuedCount.String())
	require.Equal(t, "3", lt.ActiveCount.String())
	lt, err = f.keeper.LicenseTypes.Get(f.ctx, "type.b")
	require.NoError(t, err)
	require.Equal(t, "1", lt.IssuedCount.String())

	// One LicenseIssued event per entry, in entry order.
	require.GreaterOrEqual(t, len(f.stateDB.logs), 3)
	eventLogs := f.stateDB.logs[len(f.stateDB.logs)-3:]
	for i, log := range eventLogs {
		require.Equal(t, ABI.Events[EventTypeLicenseIssued].ID, log.Topics[0])
		require.Equal(t, common.LeftPadBytes(f.OwnerHex.Bytes(), 32), log.Topics[1].Bytes())
		require.Equal(t, common.LeftPadBytes(entries[i].Holder.Bytes(), 32), log.Topics[2].Bytes())
	}
}

// bechFor is a small fixture helper for the address conversions tx tests need.
func (f *testFixture) bechFor(t *testing.T, hex common.Address) string {
	t.Helper()
	b, err := f.addrCdc.BytesToString(hex.Bytes())
	require.NoError(t, err)
	return b
}
