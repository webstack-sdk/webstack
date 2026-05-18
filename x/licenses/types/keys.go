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

// MaxIssueBatchSize bounds the number of entries in a single
// MsgBatchIssueLicense. Per-tx work is otherwise only bounded by the
// CometBFT tx-size limit; this gives a clean error before the keeper
// starts iterating a pathologically large batch.
const MaxIssueBatchSize = 100

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
