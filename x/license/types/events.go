package types

const (
	EventTypeCreateLicenseType = "create_license_type"
	EventTypeUpdateLicenseType = "update_license_type"
	EventTypeGrantPermissions  = "grant_permissions"
	EventTypeRevokePermissions = "revoke_permissions"
	EventTypeIssueLicenses     = "issue_licenses"
	EventTypeRevokeLicenses    = "revoke_licenses"
	EventTypeTransferLicense   = "transfer_license"
	EventTypeUpdateParams      = "update_params"

	AttributeKeyLicenseTypeID = "license_type_id"
	AttributeKeyLicenseID     = "license_id"
	AttributeKeyHolder        = "holder"
	AttributeKeyRecipient     = "recipient"
	AttributeKeyStatus        = "status"
	AttributeKeyAddress       = "address"
	AttributeKeyOwner         = "owner"
	AttributeKeyPermissions   = "permissions"
	AttributeKeyGrantTypes    = "grant_license_types"
)
