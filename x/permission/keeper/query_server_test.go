package keeper_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"

	"github.com/webstack-sdk/webstack/testutil/sample"
	"github.com/webstack-sdk/webstack/x/permission/keeper"
	"github.com/webstack-sdk/webstack/x/permission/types"
)

// TestModuleQueriesWithoutOwner: registered modules are queryable before any
// owner is set — the namespace comes back with an empty owner, and the
// vocabulary is still served from the registry.
func TestModuleQueriesWithoutOwner(t *testing.T) {
	k, _, ctx := setupMsgServer(t)
	q := keeper.NewQuerier(k)

	resp, err := q.Modules(ctx, &types.QueryModulesRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Namespaces, 2)
	for _, ns := range resp.Namespaces {
		require.Empty(t, ns.Owner)
	}

	mresp, err := q.Module(ctx, &types.QueryModuleRequest{Module: testModule})
	require.NoError(t, err)
	require.Equal(t, testModule, mresp.Namespace.Module)
	require.Empty(t, mresp.Namespace.Owner)
	require.Equal(t, []string{"issue", "revoke"}, mresp.Permissions)
}

// TestQueries exercises the query server over a fixed state:
//
//	granteeOne: (issue, scopeA), (issue, scopeB), (revoke, scopeA)  in testModule
//	granteeTwo: (revoke, scopeB)                                    in testModule
//	granteeOne: (operate, "")                                       in openModule
func TestQueries(t *testing.T) {
	k, ms, ctx, owner := setupWithNamespace(t)
	q := keeper.NewQuerier(k)

	openOwner := sample.AccAddress()
	_, err := ms.UpdateNamespaceOwner(ctx, &types.MsgUpdateNamespaceOwner{
		Authority: k.GetAuthority(),
		Module:    openModule,
		Owner:     openOwner,
	})
	require.NoError(t, err)

	granteeOne := sample.AccAddress()
	granteeTwo := sample.AccAddress()

	_, err = ms.GrantPermissions(ctx, &types.MsgGrantPermissions{
		Owner:   owner,
		Module:  testModule,
		Grantee: granteeOne,
		Grants: []types.PermissionScopes{
			{Permission: "issue", Scopes: []string{scopeA, scopeB}},
			{Permission: "revoke", Scopes: []string{scopeA}},
		},
	})
	require.NoError(t, err)

	_, err = ms.GrantPermissions(ctx, &types.MsgGrantPermissions{
		Owner:   owner,
		Module:  testModule,
		Grantee: granteeTwo,
		Grants:  []types.PermissionScopes{{Permission: "revoke", Scopes: []string{scopeB}}},
	})
	require.NoError(t, err)

	_, err = ms.GrantPermissions(ctx, &types.MsgGrantPermissions{
		Owner:   openOwner,
		Module:  openModule,
		Grantee: granteeOne,
		Grants:  []types.PermissionScopes{{Permission: "operate"}},
	})
	require.NoError(t, err)

	t.Run("modules", func(t *testing.T) {
		resp, err := q.Modules(ctx, &types.QueryModulesRequest{})
		require.NoError(t, err)
		require.Len(t, resp.Namespaces, 2)

		// Registered modules in ascending name order: openmod before testmod.
		require.Equal(t, openModule, resp.Namespaces[0].Module)
		require.Equal(t, openOwner, resp.Namespaces[0].Owner)
		require.Equal(t, testModule, resp.Namespaces[1].Module)
		require.Equal(t, owner, resp.Namespaces[1].Owner)
	})

	t.Run("module", func(t *testing.T) {
		resp, err := q.Module(ctx, &types.QueryModuleRequest{Module: testModule})
		require.NoError(t, err)
		require.Equal(t, owner, resp.Namespace.Owner)
		// Registered vocabulary, ascending.
		require.Equal(t, []string{"issue", "revoke"}, resp.Permissions)
	})

	t.Run("module not registered", func(t *testing.T) {
		_, err := q.Module(ctx, &types.QueryModuleRequest{Module: "ghostmod"})
		require.ErrorContains(t, err, "not registered")
	})

	t.Run("grants", func(t *testing.T) {
		resp, err := q.Grants(ctx, &types.QueryGrantsRequest{Module: testModule})
		require.NoError(t, err)
		require.Len(t, resp.Grants, 4)
		for _, g := range resp.Grants {
			require.Equal(t, testModule, g.Module)
		}
	})

	t.Run("grants pagination", func(t *testing.T) {
		var collected []types.Grant
		var nextKey []byte
		for {
			resp, err := q.Grants(ctx, &types.QueryGrantsRequest{
				Module:     testModule,
				Pagination: &query.PageRequest{Limit: 3, Key: nextKey},
			})
			require.NoError(t, err)
			collected = append(collected, resp.Grants...)
			nextKey = resp.Pagination.GetNextKey()
			if len(nextKey) == 0 {
				break
			}
		}
		require.Len(t, collected, 4)
	})

	t.Run("grants by grantee", func(t *testing.T) {
		resp, err := q.GrantsByGrantee(ctx, &types.QueryGrantsByGranteeRequest{
			Module:  testModule,
			Grantee: granteeOne,
		})
		require.NoError(t, err)
		require.Len(t, resp.Grants, 3)
		for _, g := range resp.Grants {
			require.Equal(t, granteeOne, g.Grantee)
		}
	})

	t.Run("grants by grantee empty", func(t *testing.T) {
		resp, err := q.GrantsByGrantee(ctx, &types.QueryGrantsByGranteeRequest{
			Module:  testModule,
			Grantee: sample.AccAddress(),
		})
		require.NoError(t, err)
		require.Empty(t, resp.Grants)
	})

	t.Run("grants by scope", func(t *testing.T) {
		resp, err := q.GrantsByScope(ctx, &types.QueryGrantsByScopeRequest{
			Module: testModule,
			Scope:  scopeA,
		})
		require.NoError(t, err)
		require.Len(t, resp.Grants, 2)
		for _, g := range resp.Grants {
			require.Equal(t, scopeA, g.Scope)
		}
	})

	t.Run("grants by scope with permission filter", func(t *testing.T) {
		resp, err := q.GrantsByScope(ctx, &types.QueryGrantsByScopeRequest{
			Module:     testModule,
			Scope:      scopeB,
			Permission: "revoke",
		})
		require.NoError(t, err)
		require.Len(t, resp.Grants, 1)
		require.Equal(t, granteeTwo, resp.Grants[0].Grantee)
	})

	t.Run("has permission", func(t *testing.T) {
		resp, err := q.HasPermission(ctx, &types.QueryHasPermissionRequest{
			Module:     testModule,
			Grantee:    granteeOne,
			Permission: "issue",
			Scope:      scopeA,
		})
		require.NoError(t, err)
		require.True(t, resp.HasPermission)

		resp, err = q.HasPermission(ctx, &types.QueryHasPermissionRequest{
			Module:     testModule,
			Grantee:    granteeTwo,
			Permission: "issue",
			Scope:      scopeA,
		})
		require.NoError(t, err)
		require.False(t, resp.HasPermission)

		// Module-wide grant: empty scope.
		resp, err = q.HasPermission(ctx, &types.QueryHasPermissionRequest{
			Module:     openModule,
			Grantee:    granteeOne,
			Permission: "operate",
		})
		require.NoError(t, err)
		require.True(t, resp.HasPermission)
	})
}
