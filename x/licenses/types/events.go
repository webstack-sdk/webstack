package types

const (
	EventTypeCreateLicenseType = "create_license_type"
	EventTypeSetAdminKey       = "set_admin_key"
	EventTypeRemoveAdminKey    = "remove_admin_key"
	EventTypeIssueLicense      = "issue_license"
	EventTypeRevokeLicense     = "revoke_license"
	EventTypeUpdateLicense     = "update_license"
	EventTypeTransferLicense   = "transfer_license"
	EventTypeUpdateParams      = "update_params"
	EventTypeUpdateLicenseType = "update_license_type"
	EventTypeBatchIssueLicense = "batch_issue_license"

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
