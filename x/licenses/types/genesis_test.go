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
	require.NoError(t, gs.Validate())
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
			name:    "valid default",
			genesis: *types.DefaultGenesis(),
			expErr:  false,
		},
		{
			name: "valid with data",
			genesis: types.GenesisState{
				Params: types.Params{Owner: owner},
				LicenseTypes: []types.LicenseType{
					{Id: "node", MaxSupply: math.NewInt(100), IssuedCount: math.NewInt(1)},
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
					{Id: "dup", MaxSupply: math.ZeroInt(), IssuedCount: math.ZeroInt()},
					{Id: "dup", MaxSupply: math.ZeroInt(), IssuedCount: math.ZeroInt()},
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
					{Id: "t1", MaxSupply: math.ZeroInt(), IssuedCount: math.ZeroInt()},
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
					{Id: "t1", MaxSupply: math.ZeroInt(), IssuedCount: math.ZeroInt()},
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
					{Id: "t1", MaxSupply: math.ZeroInt(), IssuedCount: math.ZeroInt()},
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
