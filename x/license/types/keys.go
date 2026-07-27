package types

import (
	"cosmossdk.io/collections"
)

const (
	ModuleName   = "license"
	StoreKey     = ModuleName
	RouterKey    = ModuleName
	QuerierRoute = ModuleName
)

// Short aliases for the generated enum constants.
const (
	StatusActive  = LicenseStatus_LICENSE_STATUS_ACTIVE
	StatusRevoked = LicenseStatus_LICENSE_STATUS_REVOKED
)

// Permission names the license module registers with the x/permission module
// under the "license" namespace. Grants are scoped per license type id.
const (
	PermissionIssue  = "issue"
	PermissionRevoke = "revoke"
)

// ValidPermissions is the full permission vocabulary registered with the
// x/permission module.
var ValidPermissions = []string{PermissionIssue, PermissionRevoke}

// Short returns the lowercase boundary form of a license status ("active",
// "revoked"), or the raw enum name for unknown values.
func (s LicenseStatus) Short() string {
	switch s {
	case StatusActive:
		return "active"
	case StatusRevoked:
		return "revoked"
	default:
		return s.String()
	}
}

// MaxIssueBatchSize bounds the number of entries in a single
// MsgIssueLicenses. Per-tx work is otherwise only bounded by the
// CometBFT tx-size limit; this gives a clean error before the keeper
// starts iterating a pathologically large batch.
const MaxIssueBatchSize = 100

var (
	LicenseTypePrefix  = collections.NewPrefix(1)
	LicensePrefix      = collections.NewPrefix(2)
	LicenseCountPrefix = collections.NewPrefix(3)

	// Index prefixes
	ActiveLicensesByHolderPrefix = collections.NewPrefix(10)
)
