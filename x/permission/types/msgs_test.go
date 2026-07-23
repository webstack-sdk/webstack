package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/webstack-sdk/webstack/testutil/sample"
	"github.com/webstack-sdk/webstack/x/permission/types"
)

func TestValidateName(t *testing.T) {
	require.NoError(t, types.ValidateName("module", "license"))
	require.NoError(t, types.ValidateName("permission", "issue-v2"))
	require.NoError(t, types.ValidateName("permission", "node.license_admin"))

	require.ErrorContains(t, types.ValidateName("module", ""), "must not be empty")
	require.ErrorContains(t, types.ValidateName("module", "License"), "invalid module")
	require.ErrorContains(t, types.ValidateName("permission", "is sue"), "invalid permission")
	require.ErrorContains(t, types.ValidateName("permission", "issue,revoke"), "invalid permission")
	require.ErrorContains(t, types.ValidateName("permission", "issue:a"), "invalid permission")
}

func TestNamespaceSpecValidate(t *testing.T) {
	require.NoError(t, types.NamespaceSpec{Permissions: []string{"issue", "revoke"}}.Validate())

	require.ErrorContains(t, types.NamespaceSpec{}.Validate(), "at least one permission")
	require.ErrorContains(t, types.NamespaceSpec{Permissions: []string{"issue", "issue"}}.Validate(), "duplicate permission")
	require.ErrorContains(t, types.NamespaceSpec{Permissions: []string{"Bad"}}.Validate(), "invalid permission")
}

func TestMsgValidateBasic(t *testing.T) {
	addr := sample.AccAddress()

	tests := []struct {
		name      string
		msg       interface{ ValidateBasic() error }
		expErrMsg string
	}{
		{
			name: "create namespace valid",
			msg:  &types.MsgCreateNamespace{Authority: addr, Module: "license", Owner: addr},
		},
		{
			name:      "create namespace bad authority",
			msg:       &types.MsgCreateNamespace{Authority: "x", Module: "license", Owner: addr},
			expErrMsg: "invalid authority",
		},
		{
			name:      "create namespace bad module",
			msg:       &types.MsgCreateNamespace{Authority: addr, Module: "Bad Mod", Owner: addr},
			expErrMsg: "invalid module",
		},
		{
			name: "transfer ownership valid",
			msg:  &types.MsgTransferOwnership{Owner: addr, Module: "license", NewOwner: sample.AccAddress()},
		},
		{
			name:      "transfer ownership bad new owner",
			msg:       &types.MsgTransferOwnership{Owner: addr, Module: "license", NewOwner: "x"},
			expErrMsg: "invalid new owner",
		},
		{
			name: "grant valid",
			msg: &types.MsgGrantPermissions{
				Owner: addr, Module: "license", Grantee: addr,
				Grants: []types.PermissionScopes{{Permission: "issue", Scopes: []string{"a"}}},
			},
		},
		{
			name: "grant empty grants",
			msg: &types.MsgGrantPermissions{
				Owner: addr, Module: "license", Grantee: addr,
			},
			expErrMsg: "grants must not be empty",
		},
		{
			name: "grant invalid permission name",
			msg: &types.MsgGrantPermissions{
				Owner: addr, Module: "license", Grantee: addr,
				Grants: []types.PermissionScopes{{Permission: "IS SUE"}},
			},
			expErrMsg: "invalid permission",
		},
		{
			name: "revoke valid",
			msg: &types.MsgRevokePermissions{
				Owner: addr, Module: "license", Grantee: addr,
				Permissions: []types.PermissionScope{{Permission: "issue", Scope: "a"}},
			},
		},
		{
			name: "revoke empty permissions",
			msg: &types.MsgRevokePermissions{
				Owner: addr, Module: "license", Grantee: addr,
			},
			expErrMsg: "permissions must not be empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.expErrMsg != "" {
				require.ErrorContains(t, err, tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
