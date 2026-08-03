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

// setupWithNamespace additionally sets a fresh owner on the testModule
// namespace (via the governance upsert) and returns that owner.
func setupWithNamespace(t testing.TB) (keeper.Keeper, types.MsgServer, sdk.Context, string) {
	t.Helper()

	k, ms, ctx := setupMsgServer(t)
	owner := sample.AccAddress()

	_, err := ms.UpdateNamespaceOwner(ctx, &types.MsgUpdateNamespaceOwner{
		Authority: k.GetAuthority(),
		Module:    testModule,
		Owner:     owner,
	})
	require.NoError(t, err)

	return k, ms, ctx, owner
}

// ---------------------------------------------------------------------------
// UpdateNamespaceOwner
// ---------------------------------------------------------------------------

func TestUpdateNamespaceOwner(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	owner := sample.AccAddress()

	tests := []struct {
		name      string
		input     *types.MsgUpdateNamespaceOwner
		expErrMsg string
	}{
		{
			name: "invalid authority",
			input: &types.MsgUpdateNamespaceOwner{
				Authority: sample.AccAddress(),
				Module:    testModule,
				Owner:     owner,
			},
			expErrMsg: "invalid authority",
		},
		{
			name: "unregistered module",
			input: &types.MsgUpdateNamespaceOwner{
				Authority: k.GetAuthority(),
				Module:    "ghostmod",
				Owner:     owner,
			},
			expErrMsg: "not registered",
		},
		{
			name: "invalid owner address",
			input: &types.MsgUpdateNamespaceOwner{
				Authority: k.GetAuthority(),
				Module:    testModule,
				Owner:     "invalid",
			},
			expErrMsg: "invalid owner address",
		},
		{
			name: "upsert: sets the first owner",
			input: &types.MsgUpdateNamespaceOwner{
				Authority: k.GetAuthority(),
				Module:    testModule,
				Owner:     owner,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.UpdateNamespaceOwner(ctx, tc.input)
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

	// Upsert: rotating an already-set owner works the same way.
	newOwner := sample.AccAddress()
	_, err := ms.UpdateNamespaceOwner(ctx, &types.MsgUpdateNamespaceOwner{
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
	require.ErrorContains(t, err, "no owner is set")

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
			name: "no owner set",
			input: &types.MsgGrantPermissions{
				Owner:   owner,
				Module:  openModule,
				Grantee: grantee,
				Grants:  []types.PermissionScopes{{Permission: "operate"}},
			},
			expErrMsg: "no owner is set",
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

	_, err := ms.UpdateNamespaceOwner(ctx, &types.MsgUpdateNamespaceOwner{
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
// Mixed granularity
// ---------------------------------------------------------------------------

const (
	// mixedModule scopes "issue" to {scopeA, scopeB} while declaring
	// createPerm module-wide.
	mixedModule = "mixedmod"
	// createPerm is the motivating case for per-permission granularity: the
	// right to create the very resources the other permissions are scoped to.
	// It has no scope to name, because the resource does not exist yet.
	createPerm = "type.create"
)

// setupMixed returns a keeper whose only namespace is mixedModule, already
// owned. It builds its own keeper rather than extending setupMsgServer so the
// registered-module counts asserted in the query tests stay put.
func setupMixed(t testing.TB) (keeper.Keeper, types.MsgServer, sdk.Context, string) {
	t.Helper()

	k, ctx := keepertest.PermissionKeeper(t)
	k.RegisterNamespace(mixedModule, types.NamespaceSpec{
		Permissions: []string{"issue", createPerm},
		ScopeExists: func(_ context.Context, scope string) (bool, error) {
			return scope == scopeA || scope == scopeB, nil
		},
		Unscoped: []string{createPerm},
	})

	ms := keeper.NewMsgServerImpl(k)
	owner := sample.AccAddress()
	_, err := ms.UpdateNamespaceOwner(ctx, &types.MsgUpdateNamespaceOwner{
		Authority: k.GetAuthority(),
		Module:    mixedModule,
		Owner:     owner,
	})
	require.NoError(t, err)

	return k, ms, ctx, owner
}

// TestGrantMixedGranularity: within one namespace, a scoped and a module-wide
// permission are both grantable and land under different key forms.
func TestGrantMixedGranularity(t *testing.T) {
	k, ms, ctx, owner := setupMixed(t)
	grantee := sample.AccAddress()

	_, err := ms.GrantPermissions(ctx, &types.MsgGrantPermissions{
		Owner:   owner,
		Module:  mixedModule,
		Grantee: grantee,
		Grants: []types.PermissionScopes{
			{Permission: "issue", Scopes: []string{scopeA}},
			{Permission: createPerm}, // no scopes: the module-wide form
		},
	})
	require.NoError(t, err)

	require.True(t, k.HasPermission(ctx, mixedModule, grantee, "issue", scopeA))
	require.True(t, k.HasPermission(ctx, mixedModule, grantee, createPerm, ""))

	// The module-wide grant does not leak into the scoped keyspace.
	require.False(t, k.HasPermission(ctx, mixedModule, grantee, createPerm, scopeA))
	require.False(t, k.HasPermission(ctx, mixedModule, grantee, "issue", ""))

	require.Len(t, k.ExportGenesis(ctx).Grants, 2)
}

// TestGrantUnscopedRejectsScope: an unscoped permission must carry the empty
// scope. A scope that exists is still rejected — the point is one canonical
// key form per (grantee, permission), not merely a valid identifier.
func TestGrantUnscopedRejectsScope(t *testing.T) {
	k, ms, ctx, owner := setupMixed(t)
	grantee := sample.AccAddress()

	_, err := ms.GrantPermissions(ctx, &types.MsgGrantPermissions{
		Owner:   owner,
		Module:  mixedModule,
		Grantee: grantee,
		Grants:  []types.PermissionScopes{{Permission: createPerm, Scopes: []string{scopeA}}},
	})
	require.ErrorIs(t, err, types.ErrInvalidScope)
	require.Contains(t, err.Error(), "is module-wide")

	require.Empty(t, k.ExportGenesis(ctx).Grants)
}

// TestGrantScopedStillRequiresScope: declaring one permission unscoped does
// not relax the rest of the namespace.
func TestGrantScopedStillRequiresScope(t *testing.T) {
	k, ms, ctx, owner := setupMixed(t)
	grantee := sample.AccAddress()

	_, err := ms.GrantPermissions(ctx, &types.MsgGrantPermissions{
		Owner:   owner,
		Module:  mixedModule,
		Grantee: grantee,
		Grants:  []types.PermissionScopes{{Permission: "issue"}},
	})
	require.ErrorIs(t, err, types.ErrInvalidScope)
	require.Contains(t, err.Error(), "scope must not be empty")

	// An unknown scope is still rejected for the scoped permission.
	_, err = ms.GrantPermissions(ctx, &types.MsgGrantPermissions{
		Owner:   owner,
		Module:  mixedModule,
		Grantee: grantee,
		Grants:  []types.PermissionScopes{{Permission: "issue", Scopes: []string{"ghost"}}},
	})
	require.ErrorIs(t, err, types.ErrInvalidScope)

	require.Empty(t, k.ExportGenesis(ctx).Grants)
}

// TestGrantMixedGranularityAtomic: a message whose valid and invalid grants
// are interleaved writes nothing, matching the all-or-nothing behaviour the
// single-granularity path already has.
func TestGrantMixedGranularityAtomic(t *testing.T) {
	k, ms, ctx, owner := setupMixed(t)
	grantee := sample.AccAddress()

	_, err := ms.GrantPermissions(ctx, &types.MsgGrantPermissions{
		Owner:   owner,
		Module:  mixedModule,
		Grantee: grantee,
		Grants: []types.PermissionScopes{
			{Permission: "issue", Scopes: []string{scopeA}}, // valid
			{Permission: createPerm, Scopes: []string{scopeB}},
		},
	})
	require.ErrorIs(t, err, types.ErrInvalidScope)

	require.False(t, k.HasPermission(ctx, mixedModule, grantee, "issue", scopeA))
	require.Empty(t, k.ExportGenesis(ctx).Grants)
}

// TestRevokeUnscopedPermission: revoking the module-wide form uses the empty
// scope, the same key the grant wrote.
func TestRevokeUnscopedPermission(t *testing.T) {
	k, ms, ctx, owner := setupMixed(t)
	grantee := sample.AccAddress()

	_, err := ms.GrantPermissions(ctx, &types.MsgGrantPermissions{
		Owner:   owner,
		Module:  mixedModule,
		Grantee: grantee,
		Grants:  []types.PermissionScopes{{Permission: createPerm}},
	})
	require.NoError(t, err)
	require.True(t, k.HasPermission(ctx, mixedModule, grantee, createPerm, ""))

	_, err = ms.RevokePermissions(ctx, &types.MsgRevokePermissions{
		Owner:       owner,
		Module:      mixedModule,
		Grantee:     grantee,
		Permissions: []types.PermissionScope{{Permission: createPerm}},
	})
	require.NoError(t, err)
	require.False(t, k.HasPermission(ctx, mixedModule, grantee, createPerm, ""))
}

// TestInitGenesisMixedGranularity: genesis import runs the same validation, so
// an unscoped grant round-trips and a scoped one for the same permission does
// not survive import.
func TestInitGenesisMixedGranularity(t *testing.T) {
	k, _, ctx, owner := setupMixed(t)
	grantee := sample.AccAddress()

	namespaces := []types.Namespace{{Module: mixedModule, Owner: owner}}

	require.NoError(t, k.InitGenesis(ctx, &types.GenesisState{
		Namespaces: namespaces,
		Grants: []types.Grant{
			{Module: mixedModule, Grantee: grantee, Permission: "issue", Scope: scopeA},
			{Module: mixedModule, Grantee: grantee, Permission: createPerm, Scope: ""},
		},
	}))
	require.True(t, k.HasPermission(ctx, mixedModule, grantee, createPerm, ""))

	k2, _, ctx2, owner2 := setupMixed(t)
	err := k2.InitGenesis(ctx2, &types.GenesisState{
		Namespaces: []types.Namespace{{Module: mixedModule, Owner: owner2}},
		Grants: []types.Grant{
			{Module: mixedModule, Grantee: grantee, Permission: createPerm, Scope: scopeA},
		},
	})
	require.ErrorIs(t, err, types.ErrInvalidScope)
}

// TestRegisterNamespaceRejectsBadMixedSpec: a malformed unscoped list is a
// wiring bug and fails at startup rather than at grant time.
func TestRegisterNamespaceRejectsBadMixedSpec(t *testing.T) {
	k, _ := keepertest.PermissionKeeper(t)

	require.Panics(t, func() {
		k.RegisterNamespace("badmod", types.NamespaceSpec{
			Permissions: []string{"issue"},
			ScopeExists: func(_ context.Context, _ string) (bool, error) { return true, nil },
			Unscoped:    []string{"nonexistent"},
		})
	})
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
