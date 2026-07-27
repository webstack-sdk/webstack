package types

const (
	EventTypeCreateLicenseType = "create_license_type"
	EventTypeUpdateLicenseType = "update_license_type"
	EventTypeIssueLicenses     = "issue_licenses"
	EventTypeRevokeLicenses    = "revoke_licenses"
	EventTypeTransferLicense   = "transfer_license"

	AttributeKeyLicenseTypeID = "license_type_id"
	AttributeKeyLicenseID     = "license_id"
	AttributeKeyHolder        = "holder"
	AttributeKeyRecipient     = "recipient"
	AttributeKeyStatus        = "status"
)
