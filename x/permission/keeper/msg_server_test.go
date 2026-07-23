package keeper_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/webstack-sdk/webstack/testutil/keeper"
	"github.com/webstack-sdk/webstack/testutil/sample"
	"github.com/webstack-sdk/webstack/x/permission/keeper"
	"github.com/webstack-sdk/webstack/x/permission/types"
)

const (
	// testModule is a registered namespace that scopes its permissions to a
	// fixed set of resources.
	testModule = "testmod"
	// openModule is a registered namespace with unconstrained scopes.
	openModule = "openmod"

	scopeA = "scope-a"
	scopeB = "scope-b"
)

// setupMsgServer returns a keeper with the two test namespaces registered:
// testModule scoped to {scopeA, scopeB}, openModule unconstrained.
func setupMsgServer(t testing.TB) (keeper.Keeper, types.MsgServer, sdk.Context) {
	t.Helper()

	k, ctx := keepertest.PermissionKeeper(t)

	k.RegisterNamespace(testModule, types.NamespaceSpec{
		Permissions: []string{"issue", "revoke"},
		ScopeExists: func(_ context.Context, scope string) (bool, error) {
			return scope == scopeA || scope == scopeB, nil
		},
	})
	k.RegisterNamespace(openModule, types.NamespaceSpec{
		Permissions: []string{"operate"},
	})

	return k, keeper.NewMsgServerImpl(k), ctx
}

// setupWithNamespace additionally creates the testModule namespace with a
// fresh owner and returns that owner.
func setupWithNamespace(t testing.TB) (keeper.Keeper, types.MsgServer, sdk.Context, string) {
	t.Helper()

	k, ms, ctx := setupMsgServer(t)
	owner := sample.AccAddress()

	_, err := ms.CreateNamespace(ctx, &types.MsgCreateNamespace{
		Authority: k.GetAuthority(),
		Module:    testModule,
		Owner:     owner,
	})
	require.NoError(t, err)

	return k, ms, ctx, owner
}

// ---------------------------------------------------------------------------
// CreateNamespace
// ---------------------------------------------------------------------------

func TestCreateNamespace(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	owner := sample.AccAddress()

	tests := []struct {
		name      string
		input     *types.MsgCreateNamespace
		expErrMsg string
	}{
		{
			name: "invalid authority",
			input: &types.MsgCreateNamespace{
				Authority: sample.AccAddress(),
				Module:    testModule,
				Owner:     owner,
			},
			expErrMsg: "invalid authority",
		},
		{
			name: "unregistered module",
			input: &types.MsgCreateNamespace{
				Authority: k.GetAuthority(),
				Module:    "ghostmod",
				Owner:     owner,
			},
			expErrMsg: "not registered",
		},
		{
			name: "invalid owner address",
			input: &types.MsgCreateNamespace{
				Authority: k.GetAuthority(),
				Module:    testModule,
				Owner:     "invalid",
			},
			expErrMsg: "invalid owner address",
		},
		{
			name: "valid",
			input: &types.MsgCreateNamespace{
				Authority: k.GetAuthority(),
				Module:    testModule,
				Owner:     owner,
			},
		},
		{
			name: "duplicate namespace",
			input: &types.MsgCreateNamespace{
				Authority: k.GetAuthority(),
				Module:    testModule,
				Owner:     owner,
			},
			expErrMsg: "already exists",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.CreateNamespace(ctx, tc.input)
			if tc.expErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				return
			}
			require.NoError(t, err)

			ns, found, err := k.GetNamespace(ctx, testModule)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, owner, ns.Owner)
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateNamespaceOwner
// ---------------------------------------------------------------------------

func TestUpdateNamespaceOwner(t *testing.T) {
	k, ms, ctx, _ := setupWithNamespace(t)
	newOwner := sample.AccAddress()

	_, err := ms.UpdateNamespaceOwner(ctx, &types.MsgUpdateNamespaceOwner{
		Authority: sample.AccAddress(),
		Module:    testModule,
		Owner:     newOwner,
	})
	require.ErrorContains(t, err, "invalid authority")

	_, err = ms.UpdateNamespaceOwner(ctx, &types.MsgUpdateNamespaceOwner{
		Authority: k.GetAuthority(),
		Module:    openModule,
		Owner:     newOwner,
	})
	require.ErrorContains(t, err, "not found")

	_, err = ms.UpdateNamespaceOwner(ctx, &types.MsgUpdateNamespaceOwner{
		Authority: k.GetAuthority(),
		Module:    testModule,
		Owner:     newOwner,
	})
	require.NoError(t, err)

	isOwner, err := k.IsOwner(ctx, testModule, newOwner)
	require.NoError(t, err)
	require.True(t, isOwner)
}

// ---------------------------------------------------------------------------
// TransferOwnership
// ---------------------------------------------------------------------------

func TestTransferOwnership(t *testing.T) {
	k, ms, ctx, owner := setupWithNamespace(t)
	newOwner := sample.AccAddress()

	_, err := ms.TransferOwnership(ctx, &types.MsgTransferOwnership{
		Owner:    owner,
		Module:   openModule,
		NewOwner: newOwner,
	})
	require.ErrorContains(t, err, "not found")

	_, err = ms.TransferOwnership(ctx, &types.MsgTransferOwnership{
		Owner:    sample.AccAddress(),
		Module:   testModule,
		NewOwner: newOwner,
	})
	require.ErrorContains(t, err, "is not the owner")

	_, err = ms.TransferOwnership(ctx, &types.MsgTransferOwnership{
		Owner:    owner,
		Module:   testModule,
		NewOwner: newOwner,
	})
	require.NoError(t, err)

	isOwner, err := k.IsOwner(ctx, testModule, newOwner)
	require.NoError(t, err)
	require.True(t, isOwner)

	// The old owner can no longer transfer.
	_, err = ms.TransferOwnership(ctx, &types.MsgTransferOwnership{
		Owner:    owner,
		Module:   testModule,
		NewOwner: owner,
	})
	require.ErrorContains(t, err, "is not the owner")
}

// ---------------------------------------------------------------------------
// GrantPermissions
// ---------------------------------------------------------------------------

func TestGrantPermissions(t *testing.T) {
	k, ms, ctx, owner := setupWithNamespace(t)
	grantee := sample.AccAddress()

	tests := []struct {
		name      string
		input     *types.MsgGrantPermissions
		expErrMsg string
	}{
		{
			name: "namespace not found",
			input: &types.MsgGrantPermissions{
				Owner:   owner,
				Module:  openModule,
				Grantee: grantee,
				Grants:  []types.PermissionScopes{{Permission: "operate"}},
			},
			expErrMsg: "not found",
		},
		{
			name: "not the owner",
			input: &types.MsgGrantPermissions{
				Owner:   sample.AccAddress(),
				Module:  testModule,
				Grantee: grantee,
				Grants:  []types.PermissionScopes{{Permission: "issue", Scopes: []string{scopeA}}},
			},
			expErrMsg: "is not the owner",
		},
		{
			name: "invalid grantee",
			input: &types.MsgGrantPermissions{
				Owner:   owner,
				Module:  testModule,
				Grantee: "invalid",
				Grants:  []types.PermissionScopes{{Permission: "issue", Scopes: []string{scopeA}}},
			},
			expErrMsg: "invalid grantee address",
		},
		{
			name: "unregistered permission",
			input: &types.MsgGrantPermissions{
				Owner:   owner,
				Module:  testModule,
				Grantee: grantee,
				Grants:  []types.PermissionScopes{{Permission: "mint", Scopes: []string{scopeA}}},
			},
			expErrMsg: "is not registered",
		},
		{
			name: "unknown scope",
			input: &types.MsgGrantPermissions{
				Owner:   owner,
				Module:  testModule,
				Grantee: grantee,
				Grants:  []types.PermissionScopes{{Permission: "issue", Scopes: []string{"scope-z"}}},
			},
			expErrMsg: "does not exist",
		},
		{
			name: "module-wide grant rejected for scoped module",
			input: &types.MsgGrantPermissions{
				Owner:   owner,
				Module:  testModule,
				Grantee: grantee,
				Grants:  []types.PermissionScopes{{Permission: "issue"}},
			},
			expErrMsg: "scope must not be empty",
		},
		{
			name: "valid",
			input: &types.MsgGrantPermissions{
				Owner:   owner,
				Module:  testModule,
				Grantee: grantee,
				Grants: []types.PermissionScopes{
					{Permission: "issue", Scopes: []string{scopeA, scopeB}},
					{Permission: "revoke", Scopes: []string{scopeA}},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.GrantPermissions(ctx, tc.input)
			if tc.expErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				return
			}
			require.NoError(t, err)

			require.True(t, k.HasPermission(ctx, testModule, grantee, "issue", scopeA))
			require.True(t, k.HasPermission(ctx, testModule, grantee, "issue", scopeB))
			require.True(t, k.HasPermission(ctx, testModule, grantee, "revoke", scopeA))
			require.False(t, k.HasPermission(ctx, testModule, grantee, "revoke", scopeB))
		})
	}
}

func TestGrantPermissionsUnion(t *testing.T) {
	k, ms, ctx, owner := setupWithNamespace(t)
	grantee := sample.AccAddress()

	grant := func(permission string, scopes ...string) {
		t.Helper()
		_, err := ms.GrantPermissions(ctx, &types.MsgGrantPermissions{
			Owner:   owner,
			Module:  testModule,
			Grantee: grantee,
			Grants:  []types.PermissionScopes{{Permission: permission, Scopes: scopes}},
		})
		require.NoError(t, err)
	}

	grant("issue", scopeA)
	grant("issue", scopeB)
	// Re-granting an existing pair is an idempotent overwrite.
	grant("issue", scopeA)

	require.True(t, k.HasPermission(ctx, testModule, grantee, "issue", scopeA))
	require.True(t, k.HasPermission(ctx, testModule, grantee, "issue", scopeB))

	all := k.ExportGenesis(ctx)
	require.Len(t, all.Grants, 2)
}

func TestGrantPermissionsModuleWide(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	owner := sample.AccAddress()
	grantee := sample.AccAddress()

	_, err := ms.CreateNamespace(ctx, &types.MsgCreateNamespace{
		Authority: k.GetAuthority(),
		Module:    openModule,
		Owner:     owner,
	})
	require.NoError(t, err)

	// openModule doesn't scope its permissions: an empty scope list grants
	// the module-wide (empty scope) form.
	_, err = ms.GrantPermissions(ctx, &types.MsgGrantPermissions{
		Owner:   owner,
		Module:  openModule,
		Grantee: grantee,
		Grants:  []types.PermissionScopes{{Permission: "operate"}},
	})
	require.NoError(t, err)

	require.True(t, k.HasPermission(ctx, openModule, grantee, "operate", ""))
	require.False(t, k.HasPermission(ctx, openModule, grantee, "operate", "anything"))

	// Arbitrary opaque scopes are also allowed.
	_, err = ms.GrantPermissions(ctx, &types.MsgGrantPermissions{
		Owner:   owner,
		Module:  openModule,
		Grantee: grantee,
		Grants:  []types.PermissionScopes{{Permission: "operate", Scopes: []string{"region-1"}}},
	})
	require.NoError(t, err)
	require.True(t, k.HasPermission(ctx, openModule, grantee, "operate", "region-1"))
}

// ---------------------------------------------------------------------------
// RevokePermissions
// ---------------------------------------------------------------------------

func TestRevokePermissions(t *testing.T) {
	k, ms, ctx, owner := setupWithNamespace(t)
	grantee := sample.AccAddress()

	_, err := ms.GrantPermissions(ctx, &types.MsgGrantPermissions{
		Owner:   owner,
		Module:  testModule,
		Grantee: grantee,
		Grants: []types.PermissionScopes{
			{Permission: "issue", Scopes: []string{scopeA, scopeB}},
			{Permission: "revoke", Scopes: []string{scopeA}},
		},
	})
	require.NoError(t, err)

	_, err = ms.RevokePermissions(ctx, &types.MsgRevokePermissions{
		Owner:   sample.AccAddress(),
		Module:  testModule,
		Grantee: grantee,
		Permissions: []types.PermissionScope{
			{Permission: "issue", Scope: scopeA},
		},
	})
	require.ErrorContains(t, err, "is not the owner")

	_, err = ms.RevokePermissions(ctx, &types.MsgRevokePermissions{
		Owner:   owner,
		Module:  testModule,
		Grantee: grantee,
		Permissions: []types.PermissionScope{
			{Permission: "issue", Scope: scopeA},
			// Not currently granted: silently ignored.
			{Permission: "revoke", Scope: scopeB},
		},
	})
	require.NoError(t, err)

	require.False(t, k.HasPermission(ctx, testModule, grantee, "issue", scopeA))
	require.True(t, k.HasPermission(ctx, testModule, grantee, "issue", scopeB))
	require.True(t, k.HasPermission(ctx, testModule, grantee, "revoke", scopeA))

	// Re-sending the same revoke is idempotent.
	_, err = ms.RevokePermissions(ctx, &types.MsgRevokePermissions{
		Owner:   owner,
		Module:  testModule,
		Grantee: grantee,
		Permissions: []types.PermissionScope{
			{Permission: "issue", Scope: scopeA},
		},
	})
	require.NoError(t, err)
	require.False(t, k.HasPermission(ctx, testModule, grantee, "issue", scopeA))
}
