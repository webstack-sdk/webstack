package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/webstack-sdk/webstack/x/network/types"
)

type Keeper struct {
	cdc    codec.BinaryCodec
	logger log.Logger

	// state management
	Schema collections.Schema
	Params collections.Item[types.Params]

	// Nodes holds every node ever activated, keyed by node address. Records
	// are never removed: a deactivated node's record is what prevents the
	// address from ever being re-activated.
	Nodes collections.Map[string, types.Node]

	// Operators is the set of operator wallets registered by authorizing a
	// key or activating a node.
	Operators collections.KeySet[string]

	// ActivationKeys holds every activation key ever authorized, keyed
	// globally by activation address. Records are never removed: a disabled
	// key's record is what permanently binds (burns) the address.
	ActivationKeys collections.Map[string, types.ActivationKey]

	// NodeStatusCounters is the per-node rolling daily MsgUpdateNodeStatus
	// counter, incremented by ante admission and re-read by the handler.
	NodeStatusCounters collections.Map[string, types.ActivityCounter]

	// NodeTypes holds every registered node type, keyed by id. Records are
	// never removed: a node's type must stay resolvable for the life of the
	// chain, and the registry is what gates activation.
	NodeTypes collections.Map[string, types.NodeType]

	// OperatorNodes indexes (operator, nodeAddress) for all of an operator's
	// nodes, active and deactivated.
	OperatorNodes collections.KeySet[collections.Pair[string, string]]

	// OperatorActivationKeys indexes (operator, activationAddress) for all of
	// an operator's keys, active and disabled.
	OperatorActivationKeys collections.KeySet[collections.Pair[string, string]]

	// OperatorNodeCounts is the denormalized {total, active} node tally per
	// (operator, nodeType), maintained by ActivateNode/DeactivateNode. The
	// node-type dimension is load-bearing: activation limits are per node
	// type, so a tally pooled across types would let one type's nodes consume
	// another's allowance.
	OperatorNodeCounts collections.Map[collections.Pair[string, string], types.OperatorNodeCounts]

	// RecentNodeActivity indexes (operator, nodeType, dayEpoch, nodeAddress)
	// with exactly one entry per node at all times: seeded by ActivateNode and
	// moved between day buckets by TouchNodeActivity. Deactivation does not
	// touch it. The two-prefix (today/yesterday) count over it is the
	// activation algorithm's RecentActiveNodeCount, which is per node type for
	// the same reason OperatorNodeCounts is.
	RecentNodeActivity collections.KeySet[collections.Quad[string, string, uint64, string]]

	// GaslessCounters holds the remaining ante-admission daily counters,
	// keyed (kind, signer): authorize per operator, activate per activation
	// key, deactivate per signer.
	GaslessCounters collections.Map[collections.Pair[string, string], types.ActivityCounter]

	// NodeTypeByLicenseType maps a license type id to the single node type bound to
	// it. The binding is one-to-one in both directions — a node type names
	// exactly one license type, and a license type backs at most one node type
	// — so this is a map rather than an index, and the map shape is what makes
	// a second binding to the same license type impossible to represent.
	NodeTypeByLicenseType collections.Map[string, string]

	licenseKeeper    types.LicenseKeeper
	permissionKeeper types.PermissionKeeper
	accountKeeper    types.AccountKeeper
	bankKeeper       types.BankKeeper

	authority string
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	logger log.Logger,
	authority string,
	licenseKeeper types.LicenseKeeper,
	permissionKeeper types.PermissionKeeper,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
) Keeper {
	logger = logger.With(log.ModuleKey, "x/"+types.ModuleName)

	sb := collections.NewSchemaBuilder(storeService)

	if authority == "" {
		authority = authtypes.NewModuleAddress(govtypes.ModuleName).String()
	}

	k := Keeper{
		cdc:    cdc,
		logger: logger,

		Params:             collections.NewItem(sb, types.ParamsPrefix, "params", codec.CollValue[types.Params](cdc)),
		Nodes:              collections.NewMap(sb, types.NodesPrefix, "nodes", collections.StringKey, codec.CollValue[types.Node](cdc)),
		Operators:          collections.NewKeySet(sb, types.OperatorsPrefix, "operators", collections.StringKey),
		ActivationKeys:     collections.NewMap(sb, types.ActivationKeysPrefix, "activation_keys", collections.StringKey, codec.CollValue[types.ActivationKey](cdc)),
		NodeStatusCounters: collections.NewMap(sb, types.NodeStatusCountersPrefix, "node_status_counters", collections.StringKey, codec.CollValue[types.ActivityCounter](cdc)),
		NodeTypes:          collections.NewMap(sb, types.NodeTypesPrefix, "node_types", collections.StringKey, codec.CollValue[types.NodeType](cdc)),

		OperatorNodes:          collections.NewKeySet(sb, types.OperatorNodesPrefix, "operator_nodes", collections.PairKeyCodec(collections.StringKey, collections.StringKey)),
		OperatorActivationKeys: collections.NewKeySet(sb, types.OperatorActivationKeysPrefix, "operator_activation_keys", collections.PairKeyCodec(collections.StringKey, collections.StringKey)),
		OperatorNodeCounts:     collections.NewMap(sb, types.OperatorNodeCountsPrefix, "operator_node_counts", collections.PairKeyCodec(collections.StringKey, collections.StringKey), codec.CollValue[types.OperatorNodeCounts](cdc)),
		RecentNodeActivity:     collections.NewKeySet(sb, types.RecentNodeActivityPrefix, "recent_node_activity", collections.QuadKeyCodec(collections.StringKey, collections.StringKey, collections.Uint64Key, collections.StringKey)),
		GaslessCounters:        collections.NewMap(sb, types.GaslessCountersPrefix, "gasless_counters", collections.PairKeyCodec(collections.StringKey, collections.StringKey), codec.CollValue[types.ActivityCounter](cdc)),
		NodeTypeByLicenseType:  collections.NewMap(sb, types.NodeTypeByLicenseTypePrefix, "node_type_by_license_type", collections.StringKey, collections.StringValue),

		licenseKeeper:    licenseKeeper,
		permissionKeeper: permissionKeeper,
		accountKeeper:    accountKeeper,
		bankKeeper:       bankKeeper,

		authority: authority,
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}

	k.Schema = schema

	return k
}

func (k Keeper) Logger() log.Logger {
	return k.logger
}

func (k Keeper) GetAuthority() string {
	return k.authority
}

// GetParams returns the module parameters. A missing params item is
// surfaced as an error: params are always seeded at genesis or upgrade.
func (k Keeper) GetParams(ctx context.Context) (types.Params, error) {
	return k.Params.Get(ctx)
}

// GetNode returns a node by address and whether it was found.
func (k Keeper) GetNode(ctx context.Context, address string) (types.Node, bool, error) {
	node, err := k.Nodes.Get(ctx, address)
	if err != nil {
		return types.Node{}, false, nil
	}
	return node, true, nil
}

// GetActivationKey returns an activation key by address and whether it was
// found.
func (k Keeper) GetActivationKey(ctx context.Context, address string) (types.ActivationKey, bool, error) {
	key, err := k.ActivationKeys.Get(ctx, address)
	if err != nil {
		return types.ActivationKey{}, false, nil
	}
	return key, true, nil
}

// IsActiveNode returns the node record for nodeAddr and whether it exists
// with active status. It is the read half of the interface consumed by
// attestation-style modules.
func (k Keeper) IsActiveNode(ctx context.Context, nodeAddr string) (types.Node, bool, error) {
	node, err := k.Nodes.Get(ctx, nodeAddr)
	if err != nil {
		return types.Node{}, false, nil
	}
	return node, node.Status == types.NodeActive, nil
}

// EnsureOperatorLicensed rejects operators that hold no active license of any
// license type backing a registered node type. It is the gate for actions that
// name no node type — authorizing an activation key happens before any node
// exists, so there is nothing narrower to check.
//
// The walk stops at the first license type the operator holds, so a licensed
// operator costs one lookup; only an unlicensed one pays the full sweep, and
// the registry is chain-level configuration rather than user-controlled.
func (k Keeper) EnsureOperatorLicensed(ctx context.Context, operator string) error {
	var licensed bool
	err := k.NodeTypeByLicenseType.Walk(ctx, nil, func(licenseTypeID, _ string) (bool, error) {
		count, err := k.licenseKeeper.CountActiveLicenses(ctx, operator, []string{licenseTypeID}, 1)
		if err != nil {
			return true, err
		}
		licensed = count > 0
		return licensed, nil
	})
	if err != nil {
		return err
	}
	if !licensed {
		return types.ErrNoActiveLicenses.Wrapf("operator %s", operator)
	}
	return nil
}

// EnsureOperatorLicensedForNodeType rejects operators that hold no active
// license of the type backing nodeType. It is the gate for actions that do
// name a node type, and is strictly narrower than EnsureOperatorLicensed:
// licenses of another node type's license type do not satisfy it.
func (k Keeper) EnsureOperatorLicensedForNodeType(ctx context.Context, operator, nodeType string) error {
	nt, err := k.NodeTypes.Get(ctx, nodeType)
	if err != nil {
		return types.ErrInvalidNodeType.Wrapf("node type %q is not registered", nodeType)
	}
	count, err := k.licenseKeeper.CountActiveLicenses(ctx, operator, []string{nt.LicenseTypeId}, 1)
	if err != nil {
		return err
	}
	if count == 0 {
		return types.ErrNoActiveLicenses.Wrapf("operator %s holds no active %s licenses", operator, nt.LicenseTypeId)
	}
	return nil
}

// CreateAccountIfNotExists creates a BaseAccount for addr if none exists.
// Pubkeys are set lazily by the stock SetPubKeyDecorator on the account's
// first signed tx.
func (k Keeper) CreateAccountIfNotExists(ctx context.Context, addr sdk.AccAddress) {
	if k.accountKeeper.HasAccount(ctx, addr) {
		return
	}
	acc := k.accountKeeper.NewAccountWithAddress(ctx, addr)
	k.accountKeeper.SetAccount(ctx, acc)
}

// hasPermission checks whether an address holds a permission via the
// x/permission module's "network" namespace. Network grants are module-wide,
// so the scope is always empty.
func (k Keeper) hasPermission(ctx context.Context, address, permission string) (bool, error) {
	return k.permissionKeeper.Has(ctx, types.ModuleName, address, permission, "")
}

// registerOperator adds operator to the operator set.
func (k Keeper) registerOperator(ctx context.Context, operator string) error {
	return k.Operators.Set(ctx, operator)
}
