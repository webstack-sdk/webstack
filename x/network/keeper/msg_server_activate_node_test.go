package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/webstack-sdk/webstack/testutil/keeper"
	"github.com/webstack-sdk/webstack/testutil/sample"
	"github.com/webstack-sdk/webstack/x/network/types"
)

// recentEntries returns every RecentNodeActivity key for an operator as
// (dayEpoch, nodeAddress) pairs, across all day buckets.
func recentEntries(t testing.TB, f *keepertest.NetworkFixture, operator string) map[string]uint64 {
	t.Helper()
	entries := make(map[string]uint64)
	rng := collections.NewPrefixedTripleRange[string, uint64, string](operator)
	require.NoError(t, f.Keeper.RecentNodeActivity.Walk(f.Ctx, rng, func(key collections.Triple[string, uint64, string]) (bool, error) {
		entries[key.K3()] = key.K2()
		return false, nil
	}))
	return entries
}

// ---------------------------------------------------------------------------
// Step 0 — authorization
// ---------------------------------------------------------------------------

func TestActivateNodeAuthorization(t *testing.T) {
	f, ms := setup(t)
	operator := sample.AccAddress()
	f.IssueLicenses(t, operator, 1)
	key := authorizeKey(t, f, ms, f.Ctx, operator)

	// Unknown activation key.
	_, err := ms.ActivateNode(f.Ctx, &types.MsgActivateNode{
		ActivationAddress: sample.AccAddress(),
		Operator:          operator,
		NodeAddress:       sample.AccAddress(),
		NodeType:          "test.node",
	})
	require.ErrorIs(t, err, types.ErrUnauthorized)

	// Key bound to a different operator than the msg names.
	other := sample.AccAddress()
	f.IssueLicenses(t, other, 1)
	_, err = ms.ActivateNode(f.Ctx, &types.MsgActivateNode{
		ActivationAddress: key,
		Operator:          other,
		NodeAddress:       sample.AccAddress(),
		NodeType:          "test.node",
	})
	require.ErrorIs(t, err, types.ErrUnauthorized)

	// Disabled keys cannot activate.
	_, err = ms.DeauthorizeActivationKey(f.Ctx, &types.MsgDeauthorizeActivationKey{
		Operator:          operator,
		ActivationAddress: key,
	})
	require.NoError(t, err)
	_, err = ms.ActivateNode(f.Ctx, &types.MsgActivateNode{
		ActivationAddress: key,
		Operator:          operator,
		NodeAddress:       sample.AccAddress(),
		NodeType:          "test.node",
	})
	require.ErrorIs(t, err, types.ErrUnauthorized)
}

// ---------------------------------------------------------------------------
// Step 1 — node addresses are single-use
// ---------------------------------------------------------------------------

func TestActivateNodeAddressSingleUse(t *testing.T) {
	f, ms := setup(t)
	operator := sample.AccAddress()
	f.IssueLicenses(t, operator, 2)
	key := authorizeKey(t, f, ms, f.Ctx, operator)
	node := activateNode(t, f, ms, f.Ctx, key, operator)

	// Re-activating the same address fails while active...
	_, err := ms.ActivateNode(f.Ctx, &types.MsgActivateNode{
		ActivationAddress: key,
		Operator:          operator,
		NodeAddress:       node,
		NodeType:          "test.node",
	})
	require.ErrorIs(t, err, types.ErrNodeExists)

	// ...and stays burned after deactivation — no reactivation ever.
	_, err = ms.DeactivateNode(f.Ctx, &types.MsgDeactivateNode{Signer: operator, NodeAddress: node})
	require.NoError(t, err)
	_, err = ms.ActivateNode(f.Ctx, &types.MsgActivateNode{
		ActivationAddress: key,
		Operator:          operator,
		NodeAddress:       node,
		NodeType:          "test.node",
	})
	require.ErrorIs(t, err, types.ErrNodeExists)

	// Even under a different operator.
	other := sample.AccAddress()
	f.IssueLicenses(t, other, 1)
	otherKey := authorizeKey(t, f, ms, f.Ctx, other)
	_, err = ms.ActivateNode(f.Ctx, &types.MsgActivateNode{
		ActivationAddress: otherKey,
		Operator:          other,
		NodeAddress:       node,
		NodeType:          "test.node",
	})
	require.ErrorIs(t, err, types.ErrNodeExists)
}

// ---------------------------------------------------------------------------
// Steps 2-6 — licenses, limits, spam
// ---------------------------------------------------------------------------

func TestActivateNodeNoLicenses(t *testing.T) {
	f, ms := setup(t)
	operator := sample.AccAddress()
	// Authorizing needs no licenses; activating does.
	key := authorizeKey(t, f, ms, f.Ctx, operator)

	_, err := ms.ActivateNode(f.Ctx, &types.MsgActivateNode{
		ActivationAddress: key,
		Operator:          operator,
		NodeAddress:       sample.AccAddress(),
		NodeType:          "test.node",
	})
	require.ErrorIs(t, err, types.ErrNoActiveLicenses)
}

func TestActivateNodeRevokedLicensesStopCounting(t *testing.T) {
	f, ms := setup(t)
	operator := sample.AccAddress()
	f.IssueLicenses(t, operator, 1)
	key := authorizeKey(t, f, ms, f.Ctx, operator)
	activateNode(t, f, ms, f.Ctx, key, operator)

	f.RevokeLicenses(t, operator, 1)

	_, err := ms.ActivateNode(f.Ctx, &types.MsgActivateNode{
		ActivationAddress: key,
		Operator:          operator,
		NodeAddress:       sample.AccAddress(),
		NodeType:          "test.node",
	})
	require.ErrorIs(t, err, types.ErrNoActiveLicenses)
}

// TestActivateNodeLimitBoundary: 1 license x multiplier 3 admits exactly
// three concurrently active nodes on the first day.
func TestActivateNodeLimitBoundary(t *testing.T) {
	f, ms := setup(t)
	operator := sample.AccAddress()
	f.IssueLicenses(t, operator, 1)
	key := authorizeKey(t, f, ms, f.Ctx, operator)

	for i := 0; i < 3; i++ {
		activateNode(t, f, ms, f.Ctx, key, operator)
	}

	_, err := ms.ActivateNode(f.Ctx, &types.MsgActivateNode{
		ActivationAddress: key,
		Operator:          operator,
		NodeAddress:       sample.AccAddress(),
		NodeType:          "test.node",
	})
	require.ErrorIs(t, err, types.ErrActivationLimit)
	require.ErrorContains(t, err, "(3)")

	// Deactivating one frees an active slot.
	nodes := recentEntries(t, f, operator)
	var anyNode string
	for node := range nodes {
		anyNode = node
		break
	}
	_, err = ms.DeactivateNode(f.Ctx, &types.MsgDeactivateNode{Signer: operator, NodeAddress: anyNode})
	require.NoError(t, err)
	activateNode(t, f, ms, f.Ctx, key, operator)
}

// TestActivateNodeSpamLimit drives an activate/deactivate churn cycle to the
// spam ceiling: with 1 license, the ninth node activated within the recent
// window trips spam_limit_multiplier x count = 9 for the tenth, regardless
// of free active slots. Two days later the recent window has drained and
// activation works again.
func TestActivateNodeSpamLimit(t *testing.T) {
	f, ms := setup(t)
	operator := sample.AccAddress()
	f.IssueLicenses(t, operator, 1)
	key := authorizeKey(t, f, ms, f.Ctx, operator)

	// Rounds of activate-3 / deactivate-3: each round adds 3 recent
	// entries that deactivation deliberately retains.
	var lastNodes []string
	for round := 0; round < 3; round++ {
		lastNodes = lastNodes[:0]
		for i := 0; i < 3; i++ {
			lastNodes = append(lastNodes, activateNode(t, f, ms, f.Ctx, key, operator))
		}
		if round < 2 {
			for _, node := range lastNodes {
				_, err := ms.DeactivateNode(f.Ctx, &types.MsgDeactivateNode{Signer: operator, NodeAddress: node})
				require.NoError(t, err)
			}
		}
	}

	// Nine nodes were activated today; even with active slots free the
	// spam ceiling rejects the tenth.
	_, err := ms.DeactivateNode(f.Ctx, &types.MsgDeactivateNode{Signer: operator, NodeAddress: lastNodes[0]})
	require.NoError(t, err)
	_, err = ms.ActivateNode(f.Ctx, &types.MsgActivateNode{
		ActivationAddress: key,
		Operator:          operator,
		NodeAddress:       sample.AccAddress(),
		NodeType:          "test.node",
	})
	require.ErrorIs(t, err, types.ErrRecentActivationLimit)
	require.ErrorContains(t, err, "(9)")

	// Two days on, the recent window is empty and activation is admitted
	// through the recent-count check despite the all-time total.
	later := f.WithBlockTime(keepertest.NetworkFixtureBlockTime.Add(48 * time.Hour))
	_, err = ms.ActivateNode(later, &types.MsgActivateNode{
		ActivationAddress: key,
		Operator:          operator,
		NodeAddress:       sample.AccAddress(),
		NodeType:          "test.node",
	})
	require.NoError(t, err)
}

// TestActivateNodeAggregatesLicenseTypes: limits aggregate across every
// counted license type.
func TestActivateNodeAggregatesLicenseTypes(t *testing.T) {
	f, ms := setup(t)
	operator := sample.AccAddress()

	const secondType = "node.license.2"
	setParams(t, f, func(p *types.Params) {
		p.LicenseTypes = []string{f.LicenseType, secondType}
	})
	f.IssueLicenses(t, operator, 1)
	f.IssueLicensesOfType(t, operator, secondType, 1)

	key := authorizeKey(t, f, ms, f.Ctx, operator)
	for i := 0; i < 6; i++ {
		activateNode(t, f, ms, f.Ctx, key, operator)
	}
	_, err := ms.ActivateNode(f.Ctx, &types.MsgActivateNode{
		ActivationAddress: key,
		Operator:          operator,
		NodeAddress:       sample.AccAddress(),
		NodeType:          "test.node",
	})
	require.ErrorIs(t, err, types.ErrActivationLimit)
	require.ErrorContains(t, err, "(6)")
}

func TestActivateNodeTypeRestriction(t *testing.T) {
	f, ms := setup(t)
	operator := sample.AccAddress()
	f.IssueLicenses(t, operator, 1)
	key := authorizeKey(t, f, ms, f.Ctx, operator)

	setParams(t, f, func(p *types.Params) { p.AllowedNodeTypes = []string{"trust"} })

	_, err := ms.ActivateNode(f.Ctx, &types.MsgActivateNode{
		ActivationAddress: key,
		Operator:          operator,
		NodeAddress:       sample.AccAddress(),
		NodeType:          "nano",
	})
	require.ErrorIs(t, err, types.ErrInvalidNodeType)

	_, err = ms.ActivateNode(f.Ctx, &types.MsgActivateNode{
		ActivationAddress: key,
		Operator:          operator,
		NodeAddress:       sample.AccAddress(),
		NodeType:          "trust",
	})
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Index-maintenance invariants
// ---------------------------------------------------------------------------

// TestRecentActivityInvariants pins the index contract: exactly one
// RecentNodeActivity entry per node at all times; activation seeds it,
// TouchNodeActivity moves it between day buckets, and deactivation retains
// it (deactivated nodes keep counting in the recent window).
func TestRecentActivityInvariants(t *testing.T) {
	f, ms := setup(t)
	operator := sample.AccAddress()
	f.IssueLicenses(t, operator, 1)
	key := authorizeKey(t, f, ms, f.Ctx, operator)
	node := activateNode(t, f, ms, f.Ctx, key, operator)

	day0 := types.DayEpoch(keepertest.NetworkFixtureBlockTime)

	// Activation seeds exactly one entry in today's bucket.
	entries := recentEntries(t, f, operator)
	require.Equal(t, map[string]uint64{node: day0}, entries)

	// A same-day touch keeps the single entry in place.
	require.NoError(t, f.Keeper.TouchNodeActivity(f.Ctx, node))
	require.Equal(t, map[string]uint64{node: day0}, recentEntries(t, f, operator))

	// A touch on a later day moves the entry — never duplicates it.
	later := f.WithBlockTime(keepertest.NetworkFixtureBlockTime.Add(48 * time.Hour))
	require.NoError(t, f.Keeper.TouchNodeActivity(later, node))
	require.Equal(t, map[string]uint64{node: day0 + 2}, recentEntries(t, f, operator))

	rec, err := f.Keeper.Nodes.Get(f.Ctx, node)
	require.NoError(t, err)
	require.Equal(t, later.BlockTime().Unix(), rec.LastActiveTime.Unix())

	// Deactivation touches neither the entry nor last_active_time.
	evenLater := f.WithBlockTime(keepertest.NetworkFixtureBlockTime.Add(72 * time.Hour))
	_, err = ms.DeactivateNode(evenLater, &types.MsgDeactivateNode{Signer: operator, NodeAddress: node})
	require.NoError(t, err)
	require.Equal(t, map[string]uint64{node: day0 + 2}, recentEntries(t, f, operator))
	rec, err = f.Keeper.Nodes.Get(f.Ctx, node)
	require.NoError(t, err)
	require.Equal(t, later.BlockTime().Unix(), rec.LastActiveTime.Unix())
}

// TestRecentCountWindow pins CountRecentActiveNodes to the today+yesterday
// two-prefix walk.
func TestRecentCountWindow(t *testing.T) {
	f, ms := setup(t)
	operator := sample.AccAddress()
	f.IssueLicenses(t, operator, 1)
	key := authorizeKey(t, f, ms, f.Ctx, operator)
	activateNode(t, f, ms, f.Ctx, key, operator)

	base := keepertest.NetworkFixtureBlockTime

	count, err := f.Keeper.CountRecentActiveNodes(f.Ctx, operator, base)
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)

	// Still visible the next day (yesterday prefix)...
	count, err = f.Keeper.CountRecentActiveNodes(f.Ctx, operator, base.Add(24*time.Hour))
	require.NoError(t, err)
	require.Equal(t, uint64(1), count)

	// ...and gone the day after.
	count, err = f.Keeper.CountRecentActiveNodes(f.Ctx, operator, base.Add(48*time.Hour))
	require.NoError(t, err)
	require.Equal(t, uint64(0), count)
}

// ---------------------------------------------------------------------------
// Bookkeeping on activation
// ---------------------------------------------------------------------------

func TestActivateNodeBookkeeping(t *testing.T) {
	f, ms := setup(t)
	operator := sample.AccAddress()
	f.IssueLicenses(t, operator, 1)
	key := authorizeKey(t, f, ms, f.Ctx, operator)
	node := activateNode(t, f, ms, f.Ctx, key, operator)

	rec, err := f.Keeper.Nodes.Get(f.Ctx, node)
	require.NoError(t, err)
	require.Equal(t, operator, rec.Operator)
	require.Equal(t, key, rec.ActivatedBy)
	require.Equal(t, types.NodeActive, rec.Status)
	require.Equal(t, "test.node", rec.Type)

	has, err := f.Keeper.OperatorNodes.Has(f.Ctx, collections.Join(operator, node))
	require.NoError(t, err)
	require.True(t, has)

	counts, err := f.Keeper.GetOperatorNodeCounts(f.Ctx, operator)
	require.NoError(t, err)
	require.Equal(t, uint64(1), counts.Total)
	require.Equal(t, uint64(1), counts.Active)

	nodeAddr, err := sdk.AccAddressFromBech32(node)
	require.NoError(t, err)
	require.True(t, f.AccountKeeper.HasAccount(f.Ctx, nodeAddr))
}
