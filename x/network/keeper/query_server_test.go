package keeper_test

import (
	"sort"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"

	keepertest "github.com/webstack-sdk/webstack/testutil/keeper"
	"github.com/webstack-sdk/webstack/testutil/sample"
	"github.com/webstack-sdk/webstack/x/network/keeper"
	"github.com/webstack-sdk/webstack/x/network/types"
)

// nodeFleet activates count nodes for a fresh operator and returns the
// operator plus the node addresses in activation order.
func nodeFleet(t testing.TB, f *keepertest.NetworkFixture, ms types.MsgServer, count int) (string, []string) {
	t.Helper()

	operator := sample.AccAddress()
	f.IssueLicenses(t, operator, uint64(count))
	key := authorizeKey(t, f, ms, f.Ctx, operator)

	addrs := make([]string, 0, count)
	for i := 0; i < count; i++ {
		addrs = append(addrs, activateNode(t, f, ms, f.Ctx, key, operator))
	}
	return operator, addrs
}

// deactivate takes a node out of service through the real msg path.
func deactivate(t testing.TB, f *keepertest.NetworkFixture, ms types.MsgServer, operator, node string) {
	t.Helper()
	_, err := ms.DeactivateNode(f.Ctx, &types.MsgDeactivateNode{
		Signer:      operator,
		NodeAddress: node,
	})
	require.NoError(t, err)
}

// addressesOf extracts node addresses so assertions read in terms of identity
// rather than whole records.
func addressesOf(nodes []types.Node) []string {
	out := make([]string, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, n.Address)
	}
	return out
}

// TestQueryNodesStatusFilter: an unset status keeps the original behaviour of
// listing every node ever activated, and each concrete status selects only
// its own.
func TestQueryNodesStatusFilter(t *testing.T) {
	f, ms := setup(t)
	q := keeper.NewQuerier(f.Keeper)

	operator, addrs := nodeFleet(t, f, ms, 3)
	deactivate(t, f, ms, operator, addrs[1])

	// Unset status: deactivated records are retained, so all three come back.
	all, err := q.Nodes(f.Ctx, &types.QueryNodesRequest{})
	require.NoError(t, err)
	require.ElementsMatch(t, addrs, addressesOf(all.Nodes))

	active, err := q.Nodes(f.Ctx, &types.QueryNodesRequest{Status: types.NodeActive})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{addrs[0], addrs[2]}, addressesOf(active.Nodes))

	gone, err := q.Nodes(f.Ctx, &types.QueryNodesRequest{Status: types.NodeDeactivated})
	require.NoError(t, err)
	require.Equal(t, []string{addrs[1]}, addressesOf(gone.Nodes))
	require.Equal(t, types.NodeDeactivated, gone.Nodes[0].Status)
}

// TestQueryNodesByOperatorStatusFilter: the filter narrows within the operator
// prefix and never widens past it.
func TestQueryNodesByOperatorStatusFilter(t *testing.T) {
	f, ms := setup(t)
	q := keeper.NewQuerier(f.Keeper)

	operator, addrs := nodeFleet(t, f, ms, 3)
	deactivate(t, f, ms, operator, addrs[1])

	// A second operator whose nodes must never appear in the first's results.
	otherOp, otherAddrs := nodeFleet(t, f, ms, 2)

	all, err := q.NodesByOperator(f.Ctx, &types.QueryNodesByOperatorRequest{Operator: operator})
	require.NoError(t, err)
	require.ElementsMatch(t, addrs, addressesOf(all.Nodes))

	active, err := q.NodesByOperator(f.Ctx, &types.QueryNodesByOperatorRequest{
		Operator: operator,
		Status:   types.NodeActive,
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{addrs[0], addrs[2]}, addressesOf(active.Nodes))

	// The other operator's nodes are all active, so an active-only query for
	// the first operator must still not leak them.
	require.NotContains(t, addressesOf(active.Nodes), otherAddrs[0])

	gone, err := q.NodesByOperator(f.Ctx, &types.QueryNodesByOperatorRequest{
		Operator: operator,
		Status:   types.NodeDeactivated,
	})
	require.NoError(t, err)
	require.Equal(t, []string{addrs[1]}, addressesOf(gone.Nodes))

	// The second operator is unaffected by the first's deactivation.
	otherActive, err := q.NodesByOperator(f.Ctx, &types.QueryNodesByOperatorRequest{
		Operator: otherOp,
		Status:   types.NodeActive,
	})
	require.NoError(t, err)
	require.ElementsMatch(t, otherAddrs, addressesOf(otherActive.Nodes))
}

// TestQueryNodesFilteredPagination is the real risk in filtered pagination:
// the page key advances over the underlying keyspace, not over matches, so
// paging must still yield every match exactly once and never a filtered-out
// node — including when a page boundary lands next to one.
func TestQueryNodesFilteredPagination(t *testing.T) {
	f, ms := setup(t)
	q := keeper.NewQuerier(f.Keeper)

	operator, addrs := nodeFleet(t, f, ms, 6)

	// Nodes are keyed by address, so store order is lexicographic. Deactivate
	// alternating nodes in that order, interleaving filtered-out records with
	// matches rather than clustering them at one end.
	byAddr := append([]string(nil), addrs...)
	sort.Strings(byAddr)

	var wantActive []string
	for i, addr := range byAddr {
		if i%2 == 1 {
			deactivate(t, f, ms, operator, addr)
			continue
		}
		wantActive = append(wantActive, addr)
	}
	require.Len(t, wantActive, 3)

	var got []string
	var nextKey []byte
	pages := 0
	for {
		page, err := q.Nodes(f.Ctx, &types.QueryNodesRequest{
			Status:     types.NodeActive,
			Pagination: &query.PageRequest{Key: nextKey, Limit: 2},
		})
		require.NoError(t, err)
		got = append(got, addressesOf(page.Nodes)...)
		pages++
		require.LessOrEqual(t, pages, 10, "pagination failed to terminate")

		nextKey = page.Pagination.NextKey
		if len(nextKey) == 0 {
			break
		}
	}

	require.Equal(t, wantActive, got, "every active node exactly once, in store order")
	// 3 matches at 2 per page: the assertion above is only meaningful if the
	// walk actually crossed a page boundary.
	require.Equal(t, 2, pages, "expected the filtered result set to span two pages")
}

// TestQueryNodesUnknownStatusMatchesNothing: a status outside the enum is not
// an error, it simply selects no nodes.
func TestQueryNodesUnknownStatusMatchesNothing(t *testing.T) {
	f, ms := setup(t)
	q := keeper.NewQuerier(f.Keeper)

	nodeFleet(t, f, ms, 2)

	resp, err := q.Nodes(f.Ctx, &types.QueryNodesRequest{Status: types.NodeStatus(99)})
	require.NoError(t, err)
	require.Empty(t, resp.Nodes)
}

// ---------------------------------------------------------------------------
// NodeType / NodeTypes
// ---------------------------------------------------------------------------

// idsOf extracts node type ids so assertions read in terms of identity.
func idsOf(nodeTypes []types.NodeType) []string {
	out := make([]string, 0, len(nodeTypes))
	for _, nt := range nodeTypes {
		out = append(out, nt.Id)
	}
	return out
}

// TestQueryNodeType: a registered type comes back whole; an unregistered one is
// a typed not-found rather than an empty record.
func TestQueryNodeType(t *testing.T) {
	f, _ := setup(t)
	q := keeper.NewQuerier(f.Keeper)

	resp, err := q.NodeType(f.Ctx, &types.QueryNodeTypeRequest{Id: f.NodeType})
	require.NoError(t, err)
	require.Equal(t, types.NodeType{
		Id:            f.NodeType,
		Creator:       f.Owner,
		LicenseTypeId: f.LicenseType,
	}, resp.NodeType)

	_, err = q.NodeType(f.Ctx, &types.QueryNodeTypeRequest{Id: "never.registered"})
	require.ErrorIs(t, err, types.ErrNodeTypeNotFound)
}

// TestQueryNodeTypesLicenseFilter is the query the license/node-type binding
// exists to serve: given a license type id, return its node types and nothing
// else, without the caller filtering client-side.
func TestQueryNodeTypesLicenseFilter(t *testing.T) {
	f, _ := setup(t)
	q := keeper.NewQuerier(f.Keeper)

	// Two node types on the fixture's license (one is seeded by the fixture),
	// and one on an unrelated license that must never leak into its results.
	f.RegisterNodeType(t, "extra.node", f.LicenseType)
	f.RegisterNodeType(t, "other.node", "other.license")

	all, err := q.NodeTypes(f.Ctx, &types.QueryNodeTypesRequest{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{f.NodeType, "extra.node", "other.node"}, idsOf(all.NodeTypes))

	mine, err := q.NodeTypes(f.Ctx, &types.QueryNodeTypesRequest{LicenseTypeId: f.LicenseType})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{f.NodeType, "extra.node"}, idsOf(mine.NodeTypes))
	require.NotContains(t, idsOf(mine.NodeTypes), "other.node")

	// The filtered walk hydrates full records, not just ids.
	for _, nt := range mine.NodeTypes {
		require.Equal(t, f.LicenseType, nt.LicenseTypeId)
		require.Equal(t, f.Owner, nt.Creator)
	}

	other, err := q.NodeTypes(f.Ctx, &types.QueryNodeTypesRequest{LicenseTypeId: "other.license"})
	require.NoError(t, err)
	require.Equal(t, []string{"other.node"}, idsOf(other.NodeTypes))

	// A license type nobody bound to is empty, not an error.
	none, err := q.NodeTypes(f.Ctx, &types.QueryNodeTypesRequest{LicenseTypeId: "unused.license"})
	require.NoError(t, err)
	require.Empty(t, none.NodeTypes)
}

// TestQueryNodeTypesFilteredPagination: the filtered walk runs over the
// by-license index, so paging must stay inside the prefix and yield every
// match exactly once even when a page boundary falls mid-license.
func TestQueryNodeTypesFilteredPagination(t *testing.T) {
	f, _ := setup(t)
	q := keeper.NewQuerier(f.Keeper)

	// Interleave the two licenses' ids in the *global* node type keyspace, so
	// a walk that ignored the prefix would visibly pick up the decoys.
	want := []string{"a.one", "b.two", "c.three"}
	for i, id := range want {
		f.RegisterNodeType(t, id, "target.license")
		f.RegisterNodeType(t, string(rune('a'+i))+".decoy", "decoy.license")
	}

	var got []string
	var nextKey []byte
	pages := 0
	for {
		page, err := q.NodeTypes(f.Ctx, &types.QueryNodeTypesRequest{
			LicenseTypeId: "target.license",
			Pagination:    &query.PageRequest{Key: nextKey, Limit: 2},
		})
		require.NoError(t, err)
		got = append(got, idsOf(page.NodeTypes)...)
		pages++
		require.LessOrEqual(t, pages, 10, "pagination failed to terminate")

		nextKey = page.Pagination.NextKey
		if len(nextKey) == 0 {
			break
		}
	}

	require.Equal(t, want, got, "every bound node type exactly once, in id order")
	// 3 matches at 2 per page: the assertion above only means something if the
	// walk actually crossed a page boundary.
	require.Equal(t, 2, pages, "expected the filtered result set to span two pages")
}
