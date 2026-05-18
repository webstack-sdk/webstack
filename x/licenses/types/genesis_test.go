package types_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/webstack-sdk/webstack/testutil/sample"
	"github.com/webstack-sdk/webstack/x/licenses/types"
)

func TestDefaultGenesis(t *testing.T) {
	gs := types.DefaultGenesis()
	// Default genesis has no owner set, which is invalid — owner must be configured in genesis.json
	require.Error(t, gs.Validate())
	require.Empty(t, gs.LicenseTypes)
	require.Empty(t, gs.Licenses)
	require.Empty(t, gs.AdminKeys)
}

func TestGenesisValidation(t *testing.T) {
	owner := sample.AccAddress()
	holder := sample.AccAddress()

	tests := []struct {
		name      string
		genesis   types.GenesisState
		expErr    bool
		expErrMsg string
	}{
		{
			name:      "default genesis (no owner)",
			genesis:   *types.DefaultGenesis(),
			expErr:    true,
			expErrMsg: "owner must be set",
		},
		{
			name: "valid with data",
			genesis: types.GenesisState{
				Params: types.Params{Owner: owner},
				LicenseTypes: []types.LicenseType{
					{Id: "node", MaxSupply: math.NewInt(100), IssuedCount: math.NewInt(1), ActiveCount: math.NewInt(1), RevokedCount: math.ZeroInt()},
				},
				Licenses: []types.License{
					{Id: 1, Type: "node", Holder: holder, StartDate: "2026-01-01", Status: "active"},
				},
				AdminKeys: []types.AdminKey{
					{Address: owner, Grants: []types.AdminKeyGrant{{Permission: "issue", LicenseTypes: []string{"node"}}}},
				},
			},
			expErr: false,
		},
		{
			name: "invalid owner address",
			genesis: types.GenesisState{
				Params: types.Params{Owner: "bad"},
			},
			expErr:    true,
			expErrMsg: "invalid",
		},
		{
			name: "duplicate license type",
			genesis: types.GenesisState{
				Params: types.Params{Owner: owner},
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
				Params: types.Params{Owner: owner},
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.ZeroInt(), IssuedCount: math.ZeroInt(), ActiveCount: math.ZeroInt(), RevokedCount: math.ZeroInt()},
				},
				Licenses: []types.License{
					{Id: 1, Type: "t1", Holder: holder, Status: "active"},
					{Id: 1, Type: "t1", Holder: holder, Status: "active"},
				},
			},
			expErr:    true,
			expErrMsg: "duplicate license",
		},
		{
			name: "license references unknown type",
			genesis: types.GenesisState{
				Params: types.Params{Owner: owner},
				Licenses: []types.License{
					{Id: 1, Type: "missing", Holder: holder, Status: "active"},
				},
			},
			expErr:    true,
			expErrMsg: "unknown license type",
		},
		{
			name: "license invalid status",
			genesis: types.GenesisState{
				Params: types.Params{Owner: owner},
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.ZeroInt(), IssuedCount: math.ZeroInt(), ActiveCount: math.ZeroInt(), RevokedCount: math.ZeroInt()},
				},
				Licenses: []types.License{
					{Id: 1, Type: "t1", Holder: holder, Status: "suspended"},
				},
			},
			expErr:    true,
			expErrMsg: "invalid status",
		},
		{
			name: "license invalid holder",
			genesis: types.GenesisState{
				Params: types.Params{Owner: owner},
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.ZeroInt(), IssuedCount: math.ZeroInt(), ActiveCount: math.ZeroInt(), RevokedCount: math.ZeroInt()},
				},
				Licenses: []types.License{
					{Id: 1, Type: "t1", Holder: "bad", Status: "active"},
				},
			},
			expErr:    true,
			expErrMsg: "invalid holder address",
		},
		{
			name: "duplicate admin key",
			genesis: types.GenesisState{
				Params: types.Params{Owner: owner},
				AdminKeys: []types.AdminKey{
					{Address: holder},
					{Address: holder},
				},
			},
			expErr:    true,
			expErrMsg: "duplicate admin key",
		},
		{
			name: "admin key invalid address",
			genesis: types.GenesisState{
				Params: types.Params{Owner: owner},
				AdminKeys: []types.AdminKey{
					{Address: "bad"},
				},
			},
			expErr:    true,
			expErrMsg: "invalid address",
		},
		{
			name: "license type negative max_supply",
			genesis: types.GenesisState{
				Params: types.Params{Owner: owner},
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
				Params: types.Params{Owner: owner},
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
				Params: types.Params{Owner: owner},
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
				Params: types.Params{Owner: owner},
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.NewInt(100), IssuedCount: math.NewInt(2), ActiveCount: math.NewInt(1), RevokedCount: math.ZeroInt()},
				},
				Licenses: []types.License{
					{Id: 1, Type: "t1", Holder: holder, StartDate: "2026-01-01", Status: "active"},
				},
			},
			expErr:    true,
			expErrMsg: "issued_count 2 does not match",
		},
		{
			name: "active_count mismatch",
			genesis: types.GenesisState{
				Params: types.Params{Owner: owner},
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.NewInt(100), IssuedCount: math.NewInt(1), ActiveCount: math.NewInt(0), RevokedCount: math.ZeroInt()},
				},
				Licenses: []types.License{
					{Id: 1, Type: "t1", Holder: holder, StartDate: "2026-01-01", Status: "active"},
				},
			},
			expErr:    true,
			expErrMsg: "active_count 0 does not match 1",
		},
		{
			name: "revoked_count mismatch",
			genesis: types.GenesisState{
				Params: types.Params{Owner: owner},
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.NewInt(100), IssuedCount: math.NewInt(1), ActiveCount: math.NewInt(1), RevokedCount: math.NewInt(2)},
				},
				Licenses: []types.License{
					{Id: 1, Type: "t1", Holder: holder, StartDate: "2026-01-01", Status: "active"},
				},
			},
			expErr:    true,
			expErrMsg: "revoked_count 2 does not match",
		},
		{
			name: "license invalid date format",
			genesis: types.GenesisState{
				Params: types.Params{Owner: owner},
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.ZeroInt(), IssuedCount: math.NewInt(1), ActiveCount: math.NewInt(1), RevokedCount: math.ZeroInt()},
				},
				Licenses: []types.License{
					{Id: 1, Type: "t1", Holder: holder, StartDate: "01-01-2026", Status: "active"},
				},
			},
			expErr:    true,
			expErrMsg: "YYYY-MM-DD",
		},
		{
			name: "license end_date before start_date",
			genesis: types.GenesisState{
				Params: types.Params{Owner: owner},
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.ZeroInt(), IssuedCount: math.NewInt(1), ActiveCount: math.NewInt(1), RevokedCount: math.ZeroInt()},
				},
				Licenses: []types.License{
					{Id: 1, Type: "t1", Holder: holder, StartDate: "2026-06-01", EndDate: "2026-01-01", Status: "active"},
				},
			},
			expErr:    true,
			expErrMsg: "must not be before",
		},
		{
			name: "admin grant invalid permission",
			genesis: types.GenesisState{
				Params: types.Params{Owner: owner},
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.ZeroInt(), IssuedCount: math.ZeroInt(), ActiveCount: math.ZeroInt(), RevokedCount: math.ZeroInt()},
				},
				AdminKeys: []types.AdminKey{
					{Address: owner, Grants: []types.AdminKeyGrant{{Permission: "destroy", LicenseTypes: []string{"t1"}}}},
				},
			},
			expErr:    true,
			expErrMsg: "invalid permission",
		},
		{
			name: "admin grant empty license types",
			genesis: types.GenesisState{
				Params: types.Params{Owner: owner},
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.ZeroInt(), IssuedCount: math.ZeroInt(), ActiveCount: math.ZeroInt(), RevokedCount: math.ZeroInt()},
				},
				AdminKeys: []types.AdminKey{
					{Address: owner, Grants: []types.AdminKeyGrant{{Permission: "issue", LicenseTypes: []string{}}}},
				},
			},
			expErr:    true,
			expErrMsg: "at least one license type",
		},
		{
			name: "admin grant unknown license type",
			genesis: types.GenesisState{
				Params: types.Params{Owner: owner},
				LicenseTypes: []types.LicenseType{
					{Id: "t1", MaxSupply: math.ZeroInt(), IssuedCount: math.ZeroInt(), ActiveCount: math.ZeroInt(), RevokedCount: math.ZeroInt()},
				},
				AdminKeys: []types.AdminKey{
					{Address: owner, Grants: []types.AdminKeyGrant{{Permission: "issue", LicenseTypes: []string{"missing"}}}},
				},
			},
			expErr:    true,
			expErrMsg: "unknown license type",
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
