package types

import "cosmossdk.io/errors"

var (
	ErrInvalidSigner          = errors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrLicenseTypeNotFound    = errors.Register(ModuleName, 1101, "license type not found")
	ErrLicenseTypeExists      = errors.Register(ModuleName, 1102, "license type already exists")
	ErrMaxSupplyReached       = errors.Register(ModuleName, 1103, "license type max supply reached")
	ErrLicenseNotFound        = errors.Register(ModuleName, 1104, "license not found")
	ErrNotLicenseHolder       = errors.Register(ModuleName, 1105, "signer is not the license holder")
	ErrLicenseNotTransferable = errors.Register(ModuleName, 1106, "license type is not transferrable")
	ErrUnauthorized           = errors.Register(ModuleName, 1107, "signer does not have the required admin key permission")
	ErrAdminKeyNotFound       = errors.Register(ModuleName, 1108, "admin key not found")
	ErrInvalidLicenseStatus   = errors.Register(ModuleName, 1109, "invalid license status")
	ErrInvalidPermission      = errors.Register(ModuleName, 1110, "invalid permission")
	ErrEmptyLicenseTypeID     = errors.Register(ModuleName, 1111, "license type id cannot be empty")
	ErrEmptyHolder            = errors.Register(ModuleName, 1112, "holder address cannot be empty")
	ErrEmptyBatchEntries      = errors.Register(ModuleName, 1113, "batch entries cannot be empty")
	ErrLicenseRevoked         = errors.Register(ModuleName, 1114, "license is already revoked")
)
