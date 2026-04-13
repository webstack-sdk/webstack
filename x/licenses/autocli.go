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
					RpcMethod: "Permissions",
					Use:       "permissions",
					Short:     "List valid admin key permissions",
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
					Use:       "create-license-type [id] [transferrable]",
					Short:     "Create a new license type",
					Long:      "Create a new license type. Use --max-supply to limit the number of licenses (default 0 = unlimited).",
					Example:   "webstackd tx licenses create-license-type node.license true --max-supply 1000 --from owner",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "id"},
						{ProtoField: "transferrable"},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"max_supply": {Name: "max-supply", DefaultValue: "0", Usage: "maximum number of licenses that can be issued (0 = unlimited)"},
					},
				},
				{
					RpcMethod: "UpdateLicenseType",
					Use:       "update-license-type [id]",
					Short:     "Update a license type",
					Long:      "Update a license type's transferrability and/or max supply.",
					Example:   "webstackd tx licenses update-license-type node.license --transferrable true --max-supply 2000 --from owner",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "id"},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"transferrable": {Name: "transferrable", Usage: "whether licenses of this type can be transferred"},
						"max_supply":    {Name: "max-supply", DefaultValue: "0", Usage: "maximum number of licenses (0 = unlimited)"},
					},
				},
				{
					RpcMethod: "IssueLicense",
					Use:       "issue-license [license-type-id] [holder] [count] [start-date]",
					Short:     "Issue one or more licenses",
					Long:      "Issue licenses to a holder. Use --end-date flag to set an expiry.",
					Example:   "webstackd tx licenses issue-license node.license cosmos1abc... 1 2026-01-01 --end-date 2027-01-01 --from admin",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "license_type_id"},
						{ProtoField: "holder"},
						{ProtoField: "count"},
						{ProtoField: "start_date"},
					},
					FlagOptions: map[string]*autocliv1.FlagOptions{
						"end_date": {Name: "end-date", Usage: "expiry date in YYYY-MM-DD format (optional)"},
					},
				},
				{
					RpcMethod: "RevokeLicense",
					Use:       "revoke-license [license-type-id] [holder] [count]",
					Short:     "Revoke licenses for a holder, most recent first",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "license_type_id"},
						{ProtoField: "holder"},
						{ProtoField: "count"},
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
					RpcMethod: "RemoveAdminKey",
					Use:       "remove-admin-key [address]",
					Short:     "Remove all admin key grants for an address",
					Example:   "webstackd tx licenses remove-admin-key cosmos1abc... --from owner",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "address"},
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
					RpcMethod: "BatchIssueLicense",
					Skip:      true,
				},
			},
		},
	}
}
