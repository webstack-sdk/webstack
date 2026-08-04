package keeper

import (
	"context"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/webstack-sdk/webstack/x/network/types"
)

var _ types.QueryServer = Querier{}

type Querier struct {
	Keeper
}

func NewQuerier(keeper Keeper) Querier {
	return Querier{Keeper: keeper}
}

func (q Querier) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params, err := q.Keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &types.QueryParamsResponse{Params: params}, nil
}

func (q Querier) Node(ctx context.Context, req *types.QueryNodeRequest) (*types.QueryNodeResponse, error) {
	node, err := q.Keeper.Nodes.Get(ctx, req.Address)
	if err != nil {
		return nil, types.ErrNodeNotFound.Wrapf("node %s not found", req.Address)
	}
	return &types.QueryNodeResponse{Node: node}, nil
}

// matchesStatus reports whether a node passes a request's status filter. The
// unspecified zero value means "no filter": it is never a valid stored status,
// so it cannot collide with a real one, and a request that omits the field
// therefore keeps the original unfiltered behaviour.
func matchesStatus(node types.Node, want types.NodeStatus) bool {
	return want == types.NodeStatus_NODE_STATUS_UNSPECIFIED || node.Status == want
}

// Nodes returns every node record, optionally filtered by status. Records are
// retained after deactivation, so an unfiltered listing includes every node
// ever activated.
func (q Querier) Nodes(ctx context.Context, req *types.QueryNodesRequest) (*types.QueryNodesResponse, error) {
	nodes, pageResp, err := query.CollectionFilteredPaginate(ctx, q.Keeper.Nodes, req.Pagination,
		func(_ string, node types.Node) (bool, error) {
			return matchesStatus(node, req.Status), nil
		},
		func(_ string, node types.Node) (types.Node, error) {
			return node, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return &types.QueryNodesResponse{Nodes: nodes, Pagination: pageResp}, nil
}

// NodesByOperator returns the operator's node records, optionally filtered by
// status. The status filter narrows within the operator prefix; it never
// widens the scan beyond that operator.
func (q Querier) NodesByOperator(ctx context.Context, req *types.QueryNodesByOperatorRequest) (*types.QueryNodesByOperatorResponse, error) {
	nodes, pageResp, err := query.CollectionFilteredPaginate(ctx, q.Keeper.OperatorNodes, req.Pagination,
		func(key collections.Pair[string, string], _ collections.NoValue) (bool, error) {
			node, err := q.Keeper.Nodes.Get(ctx, key.K2())
			if err != nil {
				return false, err
			}
			return matchesStatus(node, req.Status), nil
		},
		func(key collections.Pair[string, string], _ collections.NoValue) (types.Node, error) {
			return q.Keeper.Nodes.Get(ctx, key.K2())
		},
		query.WithCollectionPaginationPairPrefix[string, string](req.Operator),
	)
	if err != nil {
		return nil, err
	}
	return &types.QueryNodesByOperatorResponse{Nodes: nodes, Pagination: pageResp}, nil
}

// NodeType returns a registered node type by id.
func (q Querier) NodeType(ctx context.Context, req *types.QueryNodeTypeRequest) (*types.QueryNodeTypeResponse, error) {
	nt, err := q.Keeper.NodeTypes.Get(ctx, req.Id)
	if err != nil {
		return nil, types.ErrNodeTypeNotFound.Wrapf("node type %s not found", req.Id)
	}
	return &types.QueryNodeTypeResponse{NodeType: nt}, nil
}

// NodeTypes returns registered node types. When license_type_id is set the
// walk runs over the by-license index and touches only that license type's
// node types; when it is empty every node type is returned.
func (q Querier) NodeTypes(ctx context.Context, req *types.QueryNodeTypesRequest) (*types.QueryNodeTypesResponse, error) {
	if req.LicenseTypeId != "" {
		nodeTypes, pageResp, err := query.CollectionPaginate(ctx, q.Keeper.NodeTypesByLicense, req.Pagination,
			func(key collections.Pair[string, string], _ collections.NoValue) (types.NodeType, error) {
				return q.Keeper.NodeTypes.Get(ctx, key.K2())
			},
			query.WithCollectionPaginationPairPrefix[string, string](req.LicenseTypeId),
		)
		if err != nil {
			return nil, err
		}
		return &types.QueryNodeTypesResponse{NodeTypes: nodeTypes, Pagination: pageResp}, nil
	}

	nodeTypes, pageResp, err := query.CollectionPaginate(ctx, q.Keeper.NodeTypes, req.Pagination,
		func(_ string, nt types.NodeType) (types.NodeType, error) {
			return nt, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return &types.QueryNodeTypesResponse{NodeTypes: nodeTypes, Pagination: pageResp}, nil
}

func (q Querier) ActivationKey(ctx context.Context, req *types.QueryActivationKeyRequest) (*types.QueryActivationKeyResponse, error) {
	key, err := q.Keeper.ActivationKeys.Get(ctx, req.Address)
	if err != nil {
		return nil, types.ErrKeyNotFound.Wrapf("activation key %s not found", req.Address)
	}
	return &types.QueryActivationKeyResponse{ActivationKey: key}, nil
}

func (q Querier) ActivationKeys(ctx context.Context, req *types.QueryActivationKeysRequest) (*types.QueryActivationKeysResponse, error) {
	keys, pageResp, err := query.CollectionPaginate(ctx, q.Keeper.OperatorActivationKeys, req.Pagination,
		func(key collections.Pair[string, string], _ collections.NoValue) (types.ActivationKey, error) {
			return q.Keeper.ActivationKeys.Get(ctx, key.K2())
		},
		query.WithCollectionPaginationPairPrefix[string, string](req.Operator),
	)
	if err != nil {
		return nil, err
	}
	return &types.QueryActivationKeysResponse{ActivationKeys: keys, Pagination: pageResp}, nil
}

// NodeCounts returns the operator's node tallies alongside the limits its
// current license count implies, so a hosting provider can pre-check whether
// an activation would be admitted.
func (q Querier) NodeCounts(ctx context.Context, req *types.QueryNodeCountsRequest) (*types.QueryNodeCountsResponse, error) {
	params, err := q.Keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	counts, err := q.Keeper.GetOperatorNodeCounts(ctx, req.Operator)
	if err != nil {
		return nil, err
	}
	recent, err := q.Keeper.CountRecentActiveNodes(ctx, req.Operator, sdk.UnwrapSDKContext(ctx).BlockTime())
	if err != nil {
		return nil, err
	}
	licenses, err := q.Keeper.licenseKeeper.CountActiveLicenses(ctx, req.Operator, params.LicenseTypes, 0)
	if err != nil {
		return nil, err
	}

	return &types.QueryNodeCountsResponse{
		Total:           counts.Total,
		Active:          counts.Active,
		RecentActive:    recent,
		LicenseCount:    licenses,
		ActivationLimit: params.ActivationLimitMultiplier * licenses,
		SpamLimit:       params.SpamLimitMultiplier * licenses,
	}, nil
}
