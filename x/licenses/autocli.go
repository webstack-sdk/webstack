package licenses

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	modulev1 "github.com/webstack-sdk/webstack/api/licenses/v1"
)

func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Query the licenses module parameters",
				},
				{
					RpcMethod: "LicenseType",
					Use:       "license-type [id]",
					Short:     "Query a license type by id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "id"},
					},
				},
				{
					RpcMethod: "LicenseTypes",
					Use:       "license-types",
					Short:     "Query all license types",
				},
				{
					RpcMethod: "License",
					Use:       "license [type-id] [id]",
					Short:     "Query a license by type and id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "type_id"},
						{ProtoField: "id"},
					},
				},
				{
					RpcMethod: "LicensesByType",
					Use:       "licenses-by-type [type-id]",
					Short:     "Query all licenses for a given type",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "type_id"},
					},
				},
				{
					RpcMethod: "LicensesByHolder",
					Use:       "licenses-by-holder [holder]",
					Short:     "Query all licenses held by an address",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "holder"},
					},
				},
				{
					RpcMethod: "LicensesByHolderAndType",
					Use:       "licenses-by-holder-and-type [holder] [type-id]",
					Short:     "Query licenses by holder and type",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "holder"},
						{ProtoField: "type_id"},
					},
				},
				{
					RpcMethod: "AdminKey",
					Use:       "admin-key [address]",
					Short:     "Query admin key grants for an address",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "address"},
					},
				},
				{
					RpcMethod: "AdminKeys",
					Use:       "admin-keys",
					Short:     "Query all admin keys",
				},
				{
					RpcMethod: "AdminKeysByLicenseType",
					Use:       "admin-keys-by-license-type [license-type-id]",
					Short:     "Query admin keys for a license type",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "license_type_id"},
					},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "CreateLicenseType",
					Use:       "create-license-type [id] [transferrable] [max-supply]",
					Short:     "Create a new license type",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "id"},
						{ProtoField: "transferrable"},
						{ProtoField: "max_supply"},
					},
				},
				{
					RpcMethod: "UpdateLicenseType",
					Use:       "update-license-type [id] [transferrable] [max-supply]",
					Short:     "Update a license type",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "id"},
						{ProtoField: "transferrable"},
						{ProtoField: "max_supply"},
					},
				},
				{
					RpcMethod: "IssueLicense",
					Use:       "issue-license [license-type-id] [holder]",
					Short:     "Issue a license",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "license_type_id"},
						{ProtoField: "holder"},
					},
				},
				{
					RpcMethod: "RevokeLicense",
					Use:       "revoke-license [license-type-id] [id]",
					Short:     "Revoke a license",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "license_type_id"},
						{ProtoField: "id"},
					},
				},
				{
					RpcMethod: "UpdateLicense",
					Use:       "update-license [license-type-id] [id] [status]",
					Short:     "Update a license status",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "license_type_id"},
						{ProtoField: "id"},
						{ProtoField: "status"},
					},
				},
				{
					RpcMethod: "TransferLicense",
					Use:       "transfer-license [license-type-id] [id] [recipient]",
					Short:     "Transfer a license to a new holder",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "license_type_id"},
						{ProtoField: "id"},
						{ProtoField: "recipient"},
					},
				},
				{
					RpcMethod: "UpdateParams",
					Skip:      true,
				},
				{
					RpcMethod: "SetAdminKey",
					Skip:      true,
				},
				{
					RpcMethod: "RemoveAdminKey",
					Skip:      true,
				},
				{
					RpcMethod: "BatchIssueLicense",
					Skip:      true,
				},
			},
		},
	}
}
