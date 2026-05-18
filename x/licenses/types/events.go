package types

const (
	EventTypeCreateLicenseType         = "create_license_type"
	EventTypeUpdateLicenseType         = "update_license_type"
	EventTypeGrantAdminPermissions     = "grant_admin_permissions"
	EventTypeRevokeAdminKeyPermissions = "revoke_admin_key_permissions"
	EventTypeIssueLicense              = "issue_license"
	EventTypeRevokeLicense             = "revoke_license"
	EventTypeTransferLicense           = "transfer_license"
	EventTypeBatchIssueLicense         = "batch_issue_license"
	EventTypeUpdateParams              = "update_params"

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
