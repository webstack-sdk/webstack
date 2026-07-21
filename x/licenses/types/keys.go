package types

import (
	"fmt"
	"strings"

	"cosmossdk.io/collections"
)

const (
	ModuleName   = "licenses"
	StoreKey     = ModuleName
	RouterKey    = ModuleName
	QuerierRoute = ModuleName
)

// Short aliases for the generated enum constants.
const (
	StatusActive  = LicenseStatus_LICENSE_STATUS_ACTIVE
	StatusRevoked = LicenseStatus_LICENSE_STATUS_REVOKED

	PermissionIssue  = Permission_PERMISSION_ISSUE
	PermissionRevoke = Permission_PERMISSION_REVOKE
)

// permissionShort maps each valid permission to the lowercase form used at the
// CLI/precompile/event boundary.
var permissionShort = map[Permission]string{
	PermissionIssue:  "issue",
	PermissionRevoke: "revoke",
}

// Short returns the lowercase boundary form of a permission ("issue",
// "revoke"), or the raw enum name for unknown values.
func (p Permission) Short() string {
	if s, ok := permissionShort[p]; ok {
		return s
	}
	return p.String()
}

// IsValid reports whether p is one of the known permissions-by-address permissions.
func (p Permission) IsValid() bool {
	_, ok := permissionShort[p]
	return ok
}

// ParsePermission converts a lowercase boundary string ("issue", "revoke")
// into its Permission enum value.
func ParsePermission(s string) (Permission, error) {
	for p, short := range permissionShort {
		if short == s {
			return p, nil
		}
	}
	return Permission_PERMISSION_UNSPECIFIED, fmt.Errorf("invalid permission %q: must be one of %s", s, strings.Join(ValidPermissionStrings(), ", "))
}

// ValidPermissionStrings returns the lowercase boundary forms of all valid
// permissions in enum order, for error text and the Permissions query.
func ValidPermissionStrings() []string {
	return []string{PermissionIssue.Short(), PermissionRevoke.Short()}
}

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

// MaxPermissions bounds the per-message slice length for admin-grant
// operations: the top-level Grants/Permissions lists on
// MsgGrantPermissions / MsgRevokePermissions, and the inner
// LicenseTypes slice within each grant.
const MaxPermissions = 100

var (
	ParamsKey          = collections.NewPrefix(0)
	LicenseTypePrefix  = collections.NewPrefix(1)
	LicensePrefix      = collections.NewPrefix(2)
	LicenseCountPrefix = collections.NewPrefix(3)
	PermissionPrefix   = collections.NewPrefix(4)

	// Index prefixes
	ActiveLicensesByHolderPrefix = collections.NewPrefix(10)
)
