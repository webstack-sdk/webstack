package types

import "cosmossdk.io/collections"

const (
	ModuleName   = "licenses"
	StoreKey     = ModuleName
	RouterKey    = ModuleName
	QuerierRoute = ModuleName
)

// ValidPermissions is the set of permissions that can be granted via admin keys.
var Permissions = []string{"issue", "revoke", "update"}

var (
	ParamsKey          = collections.NewPrefix(0)
	LicenseTypePrefix  = collections.NewPrefix(1)
	LicensePrefix      = collections.NewPrefix(2)
	LicenseCountPrefix = collections.NewPrefix(3)
	AdminKeyPrefix     = collections.NewPrefix(4)

	// Index prefixes
	LicenseByHolderPrefix        = collections.NewPrefix(10)
	LicenseByHolderAndTypePrefix = collections.NewPrefix(11)
)
