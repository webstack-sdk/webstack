package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/webstack-sdk/webstack/testutil/sample"
	"github.com/webstack-sdk/webstack/x/licenses/keeper"
	"github.com/webstack-sdk/webstack/x/licenses/types"
)

func setupQuerier(k keeper.Keeper) keeper.Querier {
	return keeper.NewQuerier(k)
}

func TestQueryParams(t *testing.T) {
	k, ms, ctx, owner := setupWithOwner(t)
	q := setupQuerier(k)

	resp, err := q.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, owner, resp.Params.Owner)

	// Update params and re-query
	newOwner := sample.AccAddress()
	_, err = ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: k.GetAuthority(),
		Params:    types.Params{Owner: newOwner},
	})
	require.NoError(t, err)

	resp, err = q.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, newOwner, resp.Params.Owner)
}

func TestQueryLicenseType(t *testing.T) {
	k, ms, ctx, owner := setupWithOwner(t)
	q := setupQuerier(k)

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "node", Transferrable: true, MaxSupply: math.NewInt(50),
	})
	require.NoError(t, err)

	resp, err := q.LicenseType(ctx, &types.QueryLicenseTypeRequest{Id: "node"})
	require.NoError(t, err)
	require.Equal(t, "node", resp.LicenseType.Id)
	require.True(t, resp.LicenseType.Transferrable)
	require.Equal(t, math.NewInt(50), resp.LicenseType.MaxSupply)

	// Not found
	_, err = q.LicenseType(ctx, &types.QueryLicenseTypeRequest{Id: "missing"})
	require.Error(t, err)
}

func TestQueryLicenseTypes(t *testing.T) {
	k, ms, ctx, owner := setupWithOwner(t)
	q := setupQuerier(k)

	for _, id := range []string{"a", "b", "c"} {
		_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
			Owner: owner, Id: id, MaxSupply: math.ZeroInt(),
		})
		require.NoError(t, err)
	}

	resp, err := q.LicenseTypes(ctx, &types.QueryLicenseTypesRequest{})
	require.NoError(t, err)
	require.Len(t, resp.LicenseTypes, 3)
}

func TestQueryLicense(t *testing.T) {
	k, ms, ctx, owner := setupWithOwner(t)
	q := setupQuerier(k)
	issuer := sample.AccAddress()
	holder := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "ql", MaxSupply: math.ZeroInt(),
	})
	require.NoError(t, err)
	_, err = ms.SetAdminKey(ctx, &types.MsgSetAdminKey{
		Owner: owner, Address: issuer,
		Grants: []types.AdminKeyGrant{{Permission: "issue", LicenseTypes: []string{"ql"}}},
	})
	require.NoError(t, err)

	issueResp, err := ms.IssueLicense(ctx, &types.MsgIssueLicense{
		Issuer: issuer, LicenseTypeId: "ql", Holder: holder, StartDate: "2026-01-01",
	})
	require.NoError(t, err)

	resp, err := q.License(ctx, &types.QueryLicenseRequest{TypeId: "ql", Id: issueResp.Ids[0]})
	require.NoError(t, err)
	require.Equal(t, holder, resp.License.Holder)
	require.Equal(t, "active", resp.License.Status)

	// Not found
	_, err = q.License(ctx, &types.QueryLicenseRequest{TypeId: "ql", Id: 999})
	require.Error(t, err)
}

func TestQueryLicensesByHolder(t *testing.T) {
	k, ms, ctx, owner := setupWithOwner(t)
	q := setupQuerier(k)
	issuer := sample.AccAddress()
	holder := sample.AccAddress()

	_, err := ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "h1", MaxSupply: math.ZeroInt(),
	})
	require.NoError(t, err)
	_, err = ms.CreateLicenseType(ctx, &types.MsgCreateLicenseType{
		Owner: owner, Id: "h2", MaxSupply: math.ZeroInt(),
	})
	require.NoError(t, err)
	_, err = ms.SetAdminKey(ctx, &types.MsgSetAdminKey{
		Owner: owner, Address: issuer,
		Grants: []types.AdminKeyGrant{{Permission: "issue", LicenseTypes: []string{"h1", "h2"}}},
	})
	require.NoError(t, err)

	// Issue 2 of h1 and 1 of h2 to holder
	_, err = ms.IssueLicense(ctx, &types.MsgIssueLicense{
		Issuer: issuer, LicenseTypeId: "h1", Holder: holder, StartDate: "2026-01-01", Count: 2,
	})
	require.NoError(t, err)
	_, err = ms.IssueLicense(ctx, &types.MsgIssueLicense{
		Issuer: issuer, LicenseTypeId: "h2", Holder: holder, StartDate: "2026-01-01",
	})
	require.NoError(t, err)
	// Issue 1 to someone else
	_, err = ms.IssueLicense(ctx, &types.MsgIssueLicense{
		Issuer: issuer, LicenseTypeId: "h1", Holder: sample.AccAddress(), StartDate: "2026-01-01",
	})
	require.NoError(t, err)

	resp, err := q.LicensesByHolder(ctx, &types.QueryLicensesByHolderRequest{Holder: holder})
	require.NoError(t, err)
	require.Len(t, resp.Licenses, 3)

	// Filter by holder and type
	resp2, err := q.LicensesByHolderAndType(ctx, &types.QueryLicensesByHolderAndTypeRequest{
		Holder: holder, TypeId: "h1",
	})
	require.NoError(t, err)
	require.Len(t, resp2.Licenses, 2)
}

func TestQueryAdminKeys(t *testing.T) {
	k, ms, ctx, owner := setupWithOwner(t)
	q := setupQuerier(k)
	admin1 := sample.AccAddress()
	admin2 := sample.AccAddress()

	_, err := ms.SetAdminKey(ctx, &types.MsgSetAdminKey{
		Owner: owner, Address: admin1,
		Grants: []types.AdminKeyGrant{{Permission: "issue", LicenseTypes: []string{"t1"}}},
	})
	require.NoError(t, err)
	_, err = ms.SetAdminKey(ctx, &types.MsgSetAdminKey{
		Owner: owner, Address: admin2,
		Grants: []types.AdminKeyGrant{
			{Permission: "issue", LicenseTypes: []string{"t1"}},
			{Permission: "revoke", LicenseTypes: []string{"t1"}},
		},
	})
	require.NoError(t, err)

	// Query single
	resp, err := q.AdminKey(ctx, &types.QueryAdminKeyRequest{Address: admin1})
	require.NoError(t, err)
	require.Equal(t, admin1, resp.AdminKey.Address)
	require.Len(t, resp.AdminKey.Grants, 1)

	// Not found
	_, err = q.AdminKey(ctx, &types.QueryAdminKeyRequest{Address: sample.AccAddress()})
	require.Error(t, err)

	// Query all
	allResp, err := q.AdminKeys(ctx, &types.QueryAdminKeysRequest{})
	require.NoError(t, err)
	require.Len(t, allResp.AdminKeys, 2)
}
