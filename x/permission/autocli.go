package permission

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	modulev1 "github.com/webstack-sdk/webstack/api/permission/v1"
)

func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Namespaces",
					Use:       "namespaces",
					Short:     "Query all namespaces",
				},
				{
					RpcMethod: "Namespace",
					Use:       "namespace [module]",
					Short:     "Query a namespace and its registered permissions by module name",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "module"},
					},
				},
				{
					RpcMethod: "Grants",
					Use:       "grants [module]",
					Short:     "Query all grants within a namespace",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "module"},
					},
				},
				{
					RpcMethod: "GrantsByGrantee",
					Use:       "grants-by-grantee [module] [grantee]",
					Short:     "Query the grants held by an address within a namespace",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "module"},
						{ProtoField: "grantee"},
					},
				},
				{
					RpcMethod: "GrantsByScope",
					Use:       "grants-by-scope [module] [scope]",
					Short:     "Query the grants within a namespace that apply to a scope",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "module"},
						{ProtoField: "scope"},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"permission": {Name: "permission", Usage: "narrow the result to a single permission"},
					},
				},
				{
					RpcMethod: "HasPermission",
					Use:       "has-permission [module] [grantee] [permission] [scope]",
					Short:     "Check whether a grantee holds a (permission, scope) grant",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "module"},
						{ProtoField: "grantee"},
						{ProtoField: "permission"},
						{ProtoField: "scope"},
					},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "TransferOwnership",
					Use:       "transfer-ownership [module] [new-owner]",
					Short:     "Transfer a namespace to a new owner",
					Long:      "Transfer a namespace to a new owner. The current namespace owner (--from) must sign.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "module"},
						{ProtoField: "new_owner"},
					},
				},
				{
					// Governance-gated; submitted via gov proposal, not the CLI.
					RpcMethod: "CreateNamespace",
					Skip:      true,
				},
				{
					// Governance-gated; submitted via gov proposal, not the CLI.
					RpcMethod: "UpdateNamespaceOwner",
					Skip:      true,
				},
				{
					// Handled by the custom CmdGrantPermissions command; the
					// repeated grants field doesn't map to positional args.
					RpcMethod: "GrantPermissions",
					Skip:      true,
				},
				{
					// Handled by the custom CmdRevokePermissions command; the
					// repeated permissions field doesn't map to positional args.
					RpcMethod: "RevokePermissions",
					Skip:      true,
				},
			},
		},
	}
}
