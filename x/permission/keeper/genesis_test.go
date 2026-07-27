package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/webstack-sdk/webstack/testutil/sample"
	"github.com/webstack-sdk/webstack/x/permission/types"
)

func TestInitExportGenesisRoundTrip(t *testing.T) {
	k, _, ctx := setupMsgServer(t)

	ownerOne := sample.AccAddress()
	ownerTwo := sample.AccAddress()
	grantee := sample.AccAddress()

	genesis := &types.GenesisState{
		Namespaces: []types.Namespace{
			{Module: openModule, Owner: ownerTwo},
			{Module: testModule, Owner: ownerOne},
		},
		Grants: []types.Grant{
			{Module: testModule, Grantee: grantee, Permission: "issue", Scope: scopeA},
			{Module: testModule, Grantee: grantee, Permission: "revoke", Scope: scopeB},
			{Module: openModule, Grantee: grantee, Permission: "operate", Scope: ""},
		},
	}

	require.NoError(t, k.InitGenesis(ctx, genesis))

	require.True(t, k.HasPermission(ctx, testModule, grantee, "issue", scopeA))
	require.True(t, k.HasPermission(ctx, testModule, grantee, "revoke", scopeB))
	require.True(t, k.HasPermission(ctx, openModule, grantee, "operate", ""))

	isOwner, err := k.IsOwner(ctx, testModule, ownerOne)
	require.NoError(t, err)
	require.True(t, isOwner)

	exported := k.ExportGenesis(ctx)
	require.Len(t, exported.Namespaces, 2)
	require.Len(t, exported.Grants, 3)

	// Namespaces export in ascending module order.
	require.Equal(t, openModule, exported.Namespaces[0].Module)
	require.Equal(t, testModule, exported.Namespaces[1].Module)

	// A fresh keeper initialized from the export reproduces the same state.
	k2, _, ctx2 := setupMsgServer(t)
	require.NoError(t, k2.InitGenesis(ctx2, exported))
	require.Equal(t, exported, k2.ExportGenesis(ctx2))
}

func TestInitGenesisUnregisteredModule(t *testing.T) {
	k, _, ctx := setupMsgServer(t)

	genesis := &types.GenesisState{
		Namespaces: []types.Namespace{
			{Module: "ghostmod", Owner: sample.AccAddress()},
		},
	}

	err := k.InitGenesis(ctx, genesis)
	require.ErrorContains(t, err, "not registered")
}

func TestInitGenesisInvalidGrants(t *testing.T) {
	owner := sample.AccAddress()
	grantee := sample.AccAddress()

	tests := []struct {
		name      string
		grant     types.Grant
		expErrMsg string
	}{
		{
			name:      "unregistered permission",
			grant:     types.Grant{Module: testModule, Grantee: grantee, Permission: "mint", Scope: scopeA},
			expErrMsg: "is not registered",
		},
		{
			name:      "unknown scope",
			grant:     types.Grant{Module: testModule, Grantee: grantee, Permission: "issue", Scope: "scope-z"},
			expErrMsg: "does not exist",
		},
		{
			name:      "empty scope for scoped module",
			grant:     types.Grant{Module: testModule, Grantee: grantee, Permission: "issue", Scope: ""},
			expErrMsg: "scope must not be empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			k, _, ctx := setupMsgServer(t)
			genesis := &types.GenesisState{
				Namespaces: []types.Namespace{{Module: testModule, Owner: owner}},
				Grants:     []types.Grant{tc.grant},
			}
			err := k.InitGenesis(ctx, genesis)
			require.ErrorContains(t, err, tc.expErrMsg)
		})
	}
}

func TestGenesisStateValidate(t *testing.T) {
	owner := sample.AccAddress()
	grantee := sample.AccAddress()

	valid := func() *types.GenesisState {
		return &types.GenesisState{
			Namespaces: []types.Namespace{{Module: testModule, Owner: owner}},
			Grants: []types.Grant{
				{Module: testModule, Grantee: grantee, Permission: "issue", Scope: scopeA},
			},
		}
	}

	require.NoError(t, valid().Validate())
	require.NoError(t, types.DefaultGenesis().Validate())

	tests := []struct {
		name      string
		mutate    func(gs *types.GenesisState)
		expErrMsg string
	}{
		{
			name: "invalid module name",
			mutate: func(gs *types.GenesisState) {
				gs.Namespaces[0].Module = "Bad Module!"
			},
			expErrMsg: "invalid module",
		},
		{
			name: "duplicate namespace",
			mutate: func(gs *types.GenesisState) {
				gs.Namespaces = append(gs.Namespaces, gs.Namespaces[0])
			},
			expErrMsg: "duplicate namespace",
		},
		{
			name: "invalid namespace owner",
			mutate: func(gs *types.GenesisState) {
				gs.Namespaces[0].Owner = "invalid"
			},
			expErrMsg: "invalid owner address",
		},
		{
			name: "grant references unknown namespace",
			mutate: func(gs *types.GenesisState) {
				gs.Grants[0].Module = "ghostmod"
			},
			expErrMsg: "unknown namespace",
		},
		{
			name: "invalid grantee",
			mutate: func(gs *types.GenesisState) {
				gs.Grants[0].Grantee = "invalid"
			},
			expErrMsg: "invalid grantee address",
		},
		{
			name: "invalid permission name",
			mutate: func(gs *types.GenesisState) {
				gs.Grants[0].Permission = "Not Valid"
			},
			expErrMsg: "invalid permission",
		},
		{
			name: "duplicate grant",
			mutate: func(gs *types.GenesisState) {
				gs.Grants = append(gs.Grants, gs.Grants[0])
			},
			expErrMsg: "duplicate grant",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gs := valid()
			tc.mutate(gs)
			require.ErrorContains(t, gs.Validate(), tc.expErrMsg)
		})
	}
}
