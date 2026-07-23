package types_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/webstack-sdk/webstack/testutil/sample"
	"github.com/webstack-sdk/webstack/x/license/types"
)

func TestDefaultGenesis(t *testing.T) {
	gs := types.DefaultGenesis()
	require.NoError(t, gs.Validate())
	require.Empty(t, gs.LicenseTypes)
	require.Empty(t, gs.Licenses)
	require.Empty(t, gs.LicenseCounts)
}

func TestGenesisValidation(t *testing.T) {
	holder := sample.AccAddress()

	tests := []struct {
		name      string
		genesis   types.GenesisState
		expErr    bool
		expErrMsg string
	}{
		{
			name:    "default genesis",
			genesis: *types.DefaultGenesis(),
			expErr:  false,
		},
		{
			name: "valid with data",
			genesis: types.GenesisState{
				LicenseTypes: []types.LicenseType{
					{Id: "node", MaxSupply: math.NewInt(100), IssuedCount: math.NewInt(1), ActiveCount: math.NewInt(1), RevokedCount: math.ZeroInt()},
				},
				Licenses: []types.License{
					{Id: 1, Type: "node", Holder: holder, StartDate: "2026-01-01", Status: types.StatusActive},
				},
				LicenseCounts: []types.LicenseCount{
					{LicenseTypeId: "node", Count: 1},
				},
			},
			expErr: false,
		},
		{
			name: "duplicate license type",
			genesis: types.GenesisState{
				LicenseTypes: []types.LicenseType{
					{Id: "dup", MaxSupply: math.ZeroInt(), IssuedCount: math.ZeroInt(), ActiveCount: math.ZeroInt(), RevokedCount: math.ZeroInt()},
					{Id: "dup", MaxSupply: math.ZeroInt(), IssuedCount: math.ZeroInt(), ActiveCount: math.ZeroInt(), RevokedCount: math.ZeroInt()},
				},
			},
			expErr:    true,
			expErrMsg: "duplicate license type",
		},
		{
			name: "duplicate license",
			genesis: types.GenesisState{
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.ZeroInt(), IssuedCount: math.ZeroInt(), ActiveCount: math.ZeroInt(), RevokedCount: math.ZeroInt()},
				},
				Licenses: []types.License{
					{Id: 1, Type: "t1", Holder: holder, Status: types.StatusActive},
					{Id: 1, Type: "t1", Holder: holder, Status: types.StatusActive},
				},
			},
			expErr:    true,
			expErrMsg: "duplicate license",
		},
		{
			name: "license references unknown type",
			genesis: types.GenesisState{
				Licenses: []types.License{
					{Id: 1, Type: "missing", Holder: holder, Status: types.StatusActive},
				},
			},
			expErr:    true,
			expErrMsg: "unknown license type",
		},
		{
			name: "license invalid status",
			genesis: types.GenesisState{
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.ZeroInt(), IssuedCount: math.ZeroInt(), ActiveCount: math.ZeroInt(), RevokedCount: math.ZeroInt()},
				},
				Licenses: []types.License{
					{Id: 1, Type: "t1", Holder: holder, Status: types.LicenseStatus(99)},
				},
			},
			expErr:    true,
			expErrMsg: "invalid status",
		},
		{
			name: "license invalid holder",
			genesis: types.GenesisState{
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.ZeroInt(), IssuedCount: math.ZeroInt(), ActiveCount: math.ZeroInt(), RevokedCount: math.ZeroInt()},
				},
				Licenses: []types.License{
					{Id: 1, Type: "t1", Holder: "bad", Status: types.StatusActive},
				},
			},
			expErr:    true,
			expErrMsg: "invalid holder address",
		},
		{
			name: "license type negative max_supply",
			genesis: types.GenesisState{
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.NewInt(-1), IssuedCount: math.ZeroInt(), ActiveCount: math.ZeroInt(), RevokedCount: math.ZeroInt()},
				},
			},
			expErr:    true,
			expErrMsg: "max_supply must not be negative",
		},
		{
			name: "license type nil counter",
			genesis: types.GenesisState{
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.ZeroInt(), ActiveCount: math.ZeroInt(), RevokedCount: math.ZeroInt()},
				},
			},
			expErr:    true,
			expErrMsg: "issued_count must be set",
		},
		{
			name: "license type negative counter",
			genesis: types.GenesisState{
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.ZeroInt(), IssuedCount: math.ZeroInt(), ActiveCount: math.NewInt(-1), RevokedCount: math.ZeroInt()},
				},
			},
			expErr:    true,
			expErrMsg: "active_count must not be negative",
		},
		{
			name: "issued_count mismatch",
			genesis: types.GenesisState{
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.NewInt(100), IssuedCount: math.NewInt(2), ActiveCount: math.NewInt(1), RevokedCount: math.ZeroInt()},
				},
				Licenses: []types.License{
					{Id: 1, Type: "t1", Holder: holder, StartDate: "2026-01-01", Status: types.StatusActive},
				},
			},
			expErr:    true,
			expErrMsg: "issued_count 2 does not match",
		},
		{
			name: "active_count mismatch",
			genesis: types.GenesisState{
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.NewInt(100), IssuedCount: math.NewInt(1), ActiveCount: math.NewInt(0), RevokedCount: math.ZeroInt()},
				},
				Licenses: []types.License{
					{Id: 1, Type: "t1", Holder: holder, StartDate: "2026-01-01", Status: types.StatusActive},
				},
			},
			expErr:    true,
			expErrMsg: "active_count 0 does not match 1",
		},
		{
			name: "revoked_count mismatch",
			genesis: types.GenesisState{
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.NewInt(100), IssuedCount: math.NewInt(1), ActiveCount: math.NewInt(1), RevokedCount: math.NewInt(2)},
				},
				Licenses: []types.License{
					{Id: 1, Type: "t1", Holder: holder, StartDate: "2026-01-01", Status: types.StatusActive},
				},
			},
			expErr:    true,
			expErrMsg: "revoked_count 2 does not match",
		},
		{
			name: "license invalid date format",
			genesis: types.GenesisState{
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.ZeroInt(), IssuedCount: math.NewInt(1), ActiveCount: math.NewInt(1), RevokedCount: math.ZeroInt()},
				},
				Licenses: []types.License{
					{Id: 1, Type: "t1", Holder: holder, StartDate: "01-01-2026", Status: types.StatusActive},
				},
			},
			expErr:    true,
			expErrMsg: "YYYY-MM-DD",
		},
		{
			name: "license end_date before start_date",
			genesis: types.GenesisState{
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.ZeroInt(), IssuedCount: math.NewInt(1), ActiveCount: math.NewInt(1), RevokedCount: math.ZeroInt()},
				},
				Licenses: []types.License{
					{Id: 1, Type: "t1", Holder: holder, StartDate: "2026-06-01", EndDate: "2026-01-01", Status: types.StatusActive},
				},
			},
			expErr:    true,
			expErrMsg: "must not be before",
		},
		{
			name: "revoked license without revoked_date",
			genesis: types.GenesisState{
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.ZeroInt(), IssuedCount: math.NewInt(1), ActiveCount: math.ZeroInt(), RevokedCount: math.NewInt(1)},
				},
				Licenses: []types.License{
					{Id: 1, Type: "t1", Holder: holder, StartDate: "2026-01-01", Status: types.StatusRevoked},
				},
				LicenseCounts: []types.LicenseCount{
					{LicenseTypeId: "t1", Count: 1},
				},
			},
			expErr:    true,
			expErrMsg: "no revoked_date",
		},
		{
			name: "active license with revoked_date",
			genesis: types.GenesisState{
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.ZeroInt(), IssuedCount: math.NewInt(1), ActiveCount: math.NewInt(1), RevokedCount: math.ZeroInt()},
				},
				Licenses: []types.License{
					{Id: 1, Type: "t1", Holder: holder, StartDate: "2026-01-01", Status: types.StatusActive, RevokedDate: "2026-02-01"},
				},
				LicenseCounts: []types.LicenseCount{
					{LicenseTypeId: "t1", Count: 1},
				},
			},
			expErr:    true,
			expErrMsg: "is active but has revoked_date",
		},
		{
			name: "license count below max id",
			genesis: types.GenesisState{
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.ZeroInt(), IssuedCount: math.NewInt(1), ActiveCount: math.NewInt(1), RevokedCount: math.ZeroInt()},
				},
				Licenses: []types.License{
					{Id: 5, Type: "t1", Holder: holder, StartDate: "2026-01-01", Status: types.StatusActive},
				},
				LicenseCounts: []types.LicenseCount{
					{LicenseTypeId: "t1", Count: 4},
				},
			},
			expErr:    true,
			expErrMsg: "below the highest license id",
		},
		{
			name: "licenses without a license count entry",
			genesis: types.GenesisState{
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.ZeroInt(), IssuedCount: math.NewInt(1), ActiveCount: math.NewInt(1), RevokedCount: math.ZeroInt()},
				},
				Licenses: []types.License{
					{Id: 1, Type: "t1", Holder: holder, StartDate: "2026-01-01", Status: types.StatusActive},
				},
			},
			expErr:    true,
			expErrMsg: "no license count entry",
		},
		{
			name: "license count references unknown type",
			genesis: types.GenesisState{
				LicenseCounts: []types.LicenseCount{
					{LicenseTypeId: "missing", Count: 1},
				},
			},
			expErr:    true,
			expErrMsg: "license count references unknown license type",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.genesis.Validate()
			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
