package types

import (
	"cosmossdk.io/collections"
)

const (
	ModuleName   = "network"
	StoreKey     = ModuleName
	RouterKey    = ModuleName
	QuerierRoute = ModuleName
)

// Short aliases for the generated enum constants.
const (
	NodeActive      = NodeStatus_NODE_STATUS_ACTIVE
	NodeDeactivated = NodeStatus_NODE_STATUS_DEACTIVATED

	KeyActive   = ActivationKeyStatus_ACTIVATION_KEY_STATUS_ACTIVE
	KeyDisabled = ActivationKeyStatus_ACTIVATION_KEY_STATUS_DISABLED
)

// Permission names the network module registers with the x/permission module
// under the "network" namespace. Grants are module-wide (nil scope function);
// room is reserved for a future authorize-on-behalf permission.
const (
	PermissionWalletCreate = "wallet.create"

	// PermissionNodeTypeCreate authorizes registering node types, against any
	// existing license type. It is the whole authorization for the action, so
	// grant it only to addresses trusted with the node type registry: bindings
	// are one-to-one and permanent, so a node type registered against a
	// license type cannot later be replaced.
	PermissionNodeTypeCreate = "nodetype.create"
)

// ValidPermissions is the full permission vocabulary registered with the
// x/permission module.
var ValidPermissions = []string{PermissionWalletCreate, PermissionNodeTypeCreate}

// Bounds on MsgUpdateNodeStatus payloads. The gasless byte cap already bounds
// the allowlisted path; these give the paid path the same ceiling and a clean
// error instead of an oversized event.
const (
	MaxStatusFieldLen  = 512
	MaxStatusWorkloads = 100
)

var (
	// Data prefixes.
	ParamsPrefix             = collections.NewPrefix(1)
	NodesPrefix              = collections.NewPrefix(2)
	OperatorsPrefix          = collections.NewPrefix(3)
	ActivationKeysPrefix     = collections.NewPrefix(4)
	NodeStatusCountersPrefix = collections.NewPrefix(5)
	NodeTypesPrefix          = collections.NewPrefix(6)

	// Index / denormalized prefixes. All are derived from the data records
	// above and rebuilt on genesis import.
	OperatorNodesPrefix          = collections.NewPrefix(10)
	OperatorActivationKeysPrefix = collections.NewPrefix(11)
	OperatorNodeCountsPrefix     = collections.NewPrefix(12)
	RecentNodeActivityPrefix     = collections.NewPrefix(13)
	GaslessCountersPrefix        = collections.NewPrefix(14)
	NodeTypeByLicenseTypePrefix  = collections.NewPrefix(15)
)

// Gasless admission counter kinds, the first key of the GaslessCounters map.
// NodeStatusCounters is its own store because the plan's state schema names
// it; the remaining gasless msgs share one keyed map.
const (
	GaslessKindAuthorize  = "authorize"
	GaslessKindActivate   = "activate"
	GaslessKindDeactivate = "deactivate"
)

// GaslessScopeNone is the third GaslessCounters key component for the kinds
// that are not scoped. Only the activate kind scopes — it carries the node
// type, because its ceiling comes from the licenses backing that node type.
// The kinds measured against a flat param have nothing to scope by, and using
// one key form per kind keeps a counter lookup a point-read.
const GaslessScopeNone = ""

// Short returns the lowercase boundary form of a node status ("active",
// "deactivated"), or the raw enum name for unknown values.
func (s NodeStatus) Short() string {
	switch s {
	case NodeActive:
		return "active"
	case NodeDeactivated:
		return "deactivated"
	default:
		return s.String()
	}
}

// Short returns the lowercase boundary form of an activation key status
// ("active", "disabled"), or the raw enum name for unknown values.
func (s ActivationKeyStatus) Short() string {
	switch s {
	case KeyActive:
		return "active"
	case KeyDisabled:
		return "disabled"
	default:
		return s.String()
	}
}
