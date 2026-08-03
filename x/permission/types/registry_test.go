package types_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/webstack-sdk/webstack/x/permission/types"
)

// alwaysExists is a stand-in scope validator; these tests only care whether a
// spec declares one, never what it answers.
func alwaysExists(_ context.Context, _ string) (bool, error) { return true, nil }

func TestNamespaceSpecValidate(t *testing.T) {
	tests := []struct {
		name   string
		spec   types.NamespaceSpec
		errMsg string
	}{
		{
			name: "unscoped namespace",
			spec: types.NamespaceSpec{Permissions: []string{"operate"}},
		},
		{
			name: "fully scoped namespace",
			spec: types.NamespaceSpec{
				Permissions: []string{"issue", "revoke"},
				ScopeExists: alwaysExists,
			},
		},
		{
			name: "mixed granularity",
			spec: types.NamespaceSpec{
				Permissions: []string{"issue", "revoke", "type.create"},
				ScopeExists: alwaysExists,
				Unscoped:    []string{"type.create"},
			},
		},
		{
			name:   "no permissions",
			spec:   types.NamespaceSpec{},
			errMsg: "at least one permission",
		},
		{
			name:   "invalid permission name",
			spec:   types.NamespaceSpec{Permissions: []string{"Issue"}},
			errMsg: `invalid permission "Issue"`,
		},
		{
			name:   "duplicate permission",
			spec:   types.NamespaceSpec{Permissions: []string{"issue", "issue"}},
			errMsg: `duplicate permission "issue"`,
		},
		{
			name: "unscoped without scope validator",
			spec: types.NamespaceSpec{
				Permissions: []string{"issue", "type.create"},
				Unscoped:    []string{"type.create"},
			},
			errMsg: "without a scope validator",
		},
		{
			name: "unscoped permission not in vocabulary",
			spec: types.NamespaceSpec{
				Permissions: []string{"issue", "revoke"},
				ScopeExists: alwaysExists,
				Unscoped:    []string{"type.create"},
			},
			errMsg: `unscoped permission "type.create" is not in the namespace vocabulary`,
		},
		{
			name: "duplicate unscoped permission",
			spec: types.NamespaceSpec{
				Permissions: []string{"issue", "type.create"},
				ScopeExists: alwaysExists,
				Unscoped:    []string{"type.create", "type.create"},
			},
			errMsg: `duplicate unscoped permission "type.create"`,
		},
		{
			name: "every permission unscoped",
			spec: types.NamespaceSpec{
				Permissions: []string{"issue", "type.create"},
				ScopeExists: alwaysExists,
				Unscoped:    []string{"issue", "type.create"},
			},
			errMsg: "drop ScopeExists instead",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if tt.errMsg == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestNamespaceSpecIsUnscoped(t *testing.T) {
	spec := types.NamespaceSpec{
		Permissions: []string{"issue", "revoke", "type.create"},
		ScopeExists: alwaysExists,
		Unscoped:    []string{"type.create"},
	}

	require.True(t, spec.IsUnscoped("type.create"))
	require.False(t, spec.IsUnscoped("issue"))
	require.False(t, spec.IsUnscoped("nonexistent"))

	// A namespace that does not scope at all has no unscoped list to consult:
	// its permissions are module-wide by virtue of the nil ScopeExists, and
	// IsUnscoped stays false so validateGrantPair falls through to the
	// unconstrained path rather than demanding an empty scope.
	open := types.NamespaceSpec{Permissions: []string{"operate"}}
	require.False(t, open.IsUnscoped("operate"))
}

func TestNamespaceSpecSortedPermissions(t *testing.T) {
	spec := types.NamespaceSpec{
		Permissions: []string{"revoke", "issue", "type.create"},
		ScopeExists: alwaysExists,
		Unscoped:    []string{"type.create"},
	}

	require.Equal(t, []string{"issue", "revoke", "type.create"}, spec.SortedPermissions())

	// Sorting must not disturb the spec itself.
	require.Equal(t, []string{"revoke", "issue", "type.create"}, spec.Permissions)
}
