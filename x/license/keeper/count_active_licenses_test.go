package keeper_test

import (
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	keepertest "github.com/webstack-sdk/webstack/testutil/keeper"
	"github.com/webstack-sdk/webstack/testutil/sample"
	"github.com/webstack-sdk/webstack/x/license/keeper"
	"github.com/webstack-sdk/webstack/x/license/types"
)

// seedLicenses writes count active licenses of typeID for holder straight
// into license state and the holder index.
func seedLicenses(t testing.TB, f *keepertest.LicenseFixture, holder, typeID string, count uint64) {
	t.Helper()

	lt, found, err := f.Keeper.GetLicenseType(f.Ctx, typeID)
	require.NoError(t, err)
	if !found {
		lt = types.LicenseType{Id: typeID, MaxSupply: math.ZeroInt(), IssuedCount: math.ZeroInt(), ActiveCount: math.ZeroInt(), RevokedCount: math.ZeroInt()}
	}

	for i := uint64(0); i < count; i++ {
		id, err := f.Keeper.LicenseCounts.Get(f.Ctx, typeID)
		if err != nil {
			id = 0
		}
		id++
		require.NoError(t, f.Keeper.LicenseCounts.Set(f.Ctx, typeID, id))
		require.NoError(t, f.Keeper.Licenses.Set(f.Ctx, collections.Join(typeID, id), types.License{
			Id: id, Type: typeID, Holder: holder, StartDate: "2025-01-01", Status: types.StatusActive,
		}))
		require.NoError(t, f.Keeper.ActiveLicensesByHolder.Set(f.Ctx, collections.Join3(holder, typeID, id)))
		lt.IssuedCount = lt.IssuedCount.AddRaw(1)
		lt.ActiveCount = lt.ActiveCount.AddRaw(1)
	}
	require.NoError(t, f.Keeper.LicenseTypes.Set(f.Ctx, typeID, lt))
}

func TestCountActiveLicenses(t *testing.T) {
	f := keepertest.NewLicenseFixture(t)
	holder := sample.AccAddress()
	other := sample.AccAddress()

	seedLicenses(t, f, holder, "type.a", 3)
	seedLicenses(t, f, holder, "type.b", 2)
	seedLicenses(t, f, holder, "type.uncounted", 4)
	seedLicenses(t, f, other, "type.a", 5)

	counted := []string{"type.a", "type.b"}

	// Full count aggregates across the requested types only, per holder.
	count, err := f.Keeper.CountActiveLicenses(f.Ctx, holder, counted, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(5), count)

	// stopAt caps the walk.
	count, err = f.Keeper.CountActiveLicenses(f.Ctx, holder, counted, 1)
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)

	count, err = f.Keeper.CountActiveLicenses(f.Ctx, holder, counted, 4)
	require.NoError(t, err)
	require.Equal(t, uint64(4), count)

	// A stopAt above the holdings returns the true count.
	count, err = f.Keeper.CountActiveLicenses(f.Ctx, holder, counted, 100)
	require.NoError(t, err)
	require.Equal(t, uint64(5), count)

	// No counted types means zero without walking.
	count, err = f.Keeper.CountActiveLicenses(f.Ctx, holder, nil, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(0), count)

	// Revoked licenses drop out of the count (the index holds active only).
	f.Grant(t, other, types.PermissionRevoke, "type.a")
	// Revoke two of holder's type.a licenses via the msg path.
	msgServer := keeper.NewMsgServerImpl(f.Keeper)
	_, err = msgServer.RevokeLicenses(f.Ctx, &types.MsgRevokeLicenses{
		Revoker:       other,
		LicenseTypeId: "type.a",
		Holder:        holder,
		Count:         2,
	})
	require.NoError(t, err)

	count, err = f.Keeper.CountActiveLicenses(f.Ctx, holder, counted, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(3), count)
}
