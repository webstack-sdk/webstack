package licensesprecompile

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	licensestypes "github.com/webstack-sdk/webstack/x/licenses/types"
)

// seedLicenseType is a test helper that writes a license type directly via the
// keeper so query tests don't have to round-trip through tx methods first.
func seedLicenseType(t *testing.T, f *testFixture, id string, transferrable bool, maxSupply int64) {
	t.Helper()
	lt := licensestypes.LicenseType{
		Id:            id,
		Transferrable: transferrable,
		MaxSupply:     math.NewInt(maxSupply),
		IssuedCount:   math.ZeroInt(),
		ActiveCount:   math.ZeroInt(),
		RevokedCount:  math.ZeroInt(),
	}
	require.NoError(t, f.keeper.LicenseTypes.Set(f.ctx, id, lt))
}

func TestQueryParams(t *testing.T) {
	f := newTestFixture(t)
	method := ABI.Methods[ParamsMethod]

	bz, err := f.precompile.Params(f.ctx, &method, nil)
	require.NoError(t, err)

	out, err := method.Outputs.Unpack(bz)
	require.NoError(t, err)
	require.Len(t, out, 1)

	got := out[0].(struct {
		Owner common.Address `json:"owner"`
	})
	require.Equal(t, f.OwnerHex, got.Owner)
}

func TestQueryPermissions(t *testing.T) {
	f := newTestFixture(t)
	method := ABI.Methods[PermissionsMethod]

	bz, err := f.precompile.Permissions(f.ctx, &method, nil)
	require.NoError(t, err)

	out, err := method.Outputs.Unpack(bz)
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, licensestypes.ValidPermissionStrings(), out[0].([]string))
}

func TestQueryLicenseType(t *testing.T) {
	f := newTestFixture(t)
	seedLicenseType(t, f, "type.a", true, 100)

	method := ABI.Methods[LicenseTypeMethod]
	bz, err := f.precompile.LicenseType(f.ctx, &method, []interface{}{"type.a"})
	require.NoError(t, err)

	out, err := method.Outputs.Unpack(bz)
	require.NoError(t, err)

	got := out[0].(struct {
		Id            string   `json:"id"`
		Transferrable bool     `json:"transferrable"`
		MaxSupply     *big.Int `json:"maxSupply"`
		IssuedCount   *big.Int `json:"issuedCount"`
		ActiveCount   *big.Int `json:"activeCount"`
		RevokedCount  *big.Int `json:"revokedCount"`
	})
	require.Equal(t, "type.a", got.Id)
	require.True(t, got.Transferrable)
	require.Zero(t, got.MaxSupply.Cmp(big.NewInt(100)))
	require.Zero(t, got.IssuedCount.Sign())
}

func TestQueryLicenseTypeNotFound(t *testing.T) {
	f := newTestFixture(t)
	method := ABI.Methods[LicenseTypeMethod]

	_, err := f.precompile.LicenseType(f.ctx, &method, []interface{}{"nope"})
	require.Error(t, err)
}

func TestQueryLicenseTypes(t *testing.T) {
	f := newTestFixture(t)
	seedLicenseType(t, f, "type.a", true, 10)
	seedLicenseType(t, f, "type.b", false, 0)

	method := ABI.Methods[LicenseTypesMethod]
	bz, err := f.precompile.LicenseTypes(f.ctx, &method, nil)
	require.NoError(t, err)

	out, err := method.Outputs.Unpack(bz)
	require.NoError(t, err)
	got := out[0].([]struct {
		Id            string   `json:"id"`
		Transferrable bool     `json:"transferrable"`
		MaxSupply     *big.Int `json:"maxSupply"`
		IssuedCount   *big.Int `json:"issuedCount"`
		ActiveCount   *big.Int `json:"activeCount"`
		RevokedCount  *big.Int `json:"revokedCount"`
	})
	require.Len(t, got, 2)
	ids := []string{got[0].Id, got[1].Id}
	require.ElementsMatch(t, []string{"type.a", "type.b"}, ids)
}

// TestQueryLicenseAndByHolder exercises the License, LicensesByType, LicensesByHolder
// and LicensesByHolderAndType queries with a license issued via the tx path so we
// also cover the holder bech32→hex round trip in outputs.
func TestQueryLicenseAndByHolder(t *testing.T) {
	f := newTestFixture(t)
	seedLicenseType(t, f, "type.a", true, 0)

	// grant the owner the issue permission so they can issue
	grantIssue(t, f, f.OwnerBech, "type.a")

	holderHex := common.HexToAddress("0x2222222222222222222222222222222222222222")
	issueOne(t, f, f.OwnerHex, "type.a", holderHex, "2025-01-01", "")

	// License(typeId, id) ---------------------------------------------
	licM := ABI.Methods[LicenseMethod]
	bz, err := f.precompile.License(f.ctx, &licM, []interface{}{"type.a", uint64(1)})
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
	require.Equal(t, uint64(1), got.Id)
	require.Equal(t, "type.a", got.TypeId)
	require.Equal(t, holderHex, got.Holder)
	require.Equal(t, "active", got.Status)

	// LicensesByHolder(holder) ---------------------------------------
	holderM := ABI.Methods[LicensesByHolderMethod]
	bz, err = f.precompile.LicensesByHolder(f.ctx, &holderM, []interface{}{holderHex})
	require.NoError(t, err)
	holderOut, err := holderM.Outputs.Unpack(bz)
	require.NoError(t, err)
	holderList := holderOut[0].([]struct {
		Id          uint64         `json:"id"`
		TypeId      string         `json:"typeId"`
		Holder      common.Address `json:"holder"`
		StartDate   string         `json:"startDate"`
		EndDate     string         `json:"endDate"`
		Status      string         `json:"status"`
		RevokedDate string         `json:"revokedDate"`
	})
	require.Len(t, holderList, 1)
	require.Equal(t, uint64(1), holderList[0].Id)

	// LicensesByHolderAndType(holder, typeId) -----------------------
	htM := ABI.Methods[LicensesByHolderAndTypeMethod]
	bz, err = f.precompile.LicensesByHolderAndType(f.ctx, &htM, []interface{}{holderHex, "type.a"})
	require.NoError(t, err)
	htOut, err := htM.Outputs.Unpack(bz)
	require.NoError(t, err)
	htList := htOut[0].([]struct {
		Id          uint64         `json:"id"`
		TypeId      string         `json:"typeId"`
		Holder      common.Address `json:"holder"`
		StartDate   string         `json:"startDate"`
		EndDate     string         `json:"endDate"`
		Status      string         `json:"status"`
		RevokedDate string         `json:"revokedDate"`
	})
	require.Len(t, htList, 1)

	// LicensesByType(typeId) -----------------------------------------
	typeM := ABI.Methods[LicensesByTypeMethod]
	bz, err = f.precompile.LicensesByType(f.ctx, &typeM, []interface{}{"type.a"})
	require.NoError(t, err)
	typeOut, err := typeM.Outputs.Unpack(bz)
	require.NoError(t, err)
	typeList := typeOut[0].([]struct {
		Id          uint64         `json:"id"`
		TypeId      string         `json:"typeId"`
		Holder      common.Address `json:"holder"`
		StartDate   string         `json:"startDate"`
		EndDate     string         `json:"endDate"`
		Status      string         `json:"status"`
		RevokedDate string         `json:"revokedDate"`
	})
	require.Len(t, typeList, 1)
}

// TestQueryAdminKeys covers all three admin-key queries.
func TestQueryAdminKeys(t *testing.T) {
	f := newTestFixture(t)
	seedLicenseType(t, f, "type.a", true, 0)
	seedLicenseType(t, f, "type.b", true, 0)

	adminHex := common.HexToAddress("0x3333333333333333333333333333333333333333")
	adminBech, err := f.addrCdc.BytesToString(adminHex.Bytes())
	require.NoError(t, err)

	require.NoError(t, f.keeper.AdminGrants.Set(f.ctx, collections.Join3(adminBech, int32(licensestypes.PermissionIssue), "type.a")))
	require.NoError(t, f.keeper.AdminGrants.Set(f.ctx, collections.Join3(adminBech, int32(licensestypes.PermissionRevoke), "type.b")))

	// AdminKey ---------------------------------------------------------
	akM := ABI.Methods[AdminKeyMethod]
	bz, err := f.precompile.AdminKey(f.ctx, &akM, []interface{}{adminHex})
	require.NoError(t, err)
	akOut, err := akM.Outputs.Unpack(bz)
	require.NoError(t, err)
	ak := akOut[0].(struct {
		AdminAddress common.Address `json:"adminAddress"`
		Grants       []struct {
			Permission   string   `json:"permission"`
			LicenseTypes []string `json:"licenseTypes"`
		} `json:"grants"`
	})
	require.Equal(t, adminHex, ak.AdminAddress)
	require.Len(t, ak.Grants, 2)

	// AdminKeys --------------------------------------------------------
	allM := ABI.Methods[AdminKeysMethod]
	bz, err = f.precompile.AdminKeys(f.ctx, &allM, nil)
	require.NoError(t, err)
	allOut, err := allM.Outputs.Unpack(bz)
	require.NoError(t, err)
	all := allOut[0].([]struct {
		AdminAddress common.Address `json:"adminAddress"`
		Grants       []struct {
			Permission   string   `json:"permission"`
			LicenseTypes []string `json:"licenseTypes"`
		} `json:"grants"`
	})
	require.Len(t, all, 1)
	require.Equal(t, adminHex, all[0].AdminAddress)

	// AdminKeysByLicenseType -----------------------------------------
	byM := ABI.Methods[AdminKeysByLicenseTypeMethod]
	bz, err = f.precompile.AdminKeysByLicenseType(f.ctx, &byM, []interface{}{"type.a", "issue"})
	require.NoError(t, err)
	byOut, err := byM.Outputs.Unpack(bz)
	require.NoError(t, err)
	by := byOut[0].([]struct {
		AdminAddress common.Address `json:"adminAddress"`
		Grants       []struct {
			Permission   string   `json:"permission"`
			LicenseTypes []string `json:"licenseTypes"`
		} `json:"grants"`
	})
	require.Len(t, by, 1)
	require.Equal(t, adminHex, by[0].AdminAddress)

	// negative match: permission that the admin does not hold for type.a
	bz, err = f.precompile.AdminKeysByLicenseType(f.ctx, &byM, []interface{}{"type.a", "revoke"})
	require.NoError(t, err)
	byOut, err = byM.Outputs.Unpack(bz)
	require.NoError(t, err)
	emptyList := byOut[0].([]struct {
		AdminAddress common.Address `json:"adminAddress"`
		Grants       []struct {
			Permission   string   `json:"permission"`
			LicenseTypes []string `json:"licenseTypes"`
		} `json:"grants"`
	})
	require.Empty(t, emptyList)
}
