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

func (q Querier) Nodes(ctx context.Context, req *types.QueryNodesRequest) (*types.QueryNodesResponse, error) {
	nodes, pageResp, err := query.CollectionPaginate(ctx, q.Keeper.Nodes, req.Pagination,
		func(_ string, node types.Node) (types.Node, error) {
			return node, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return &types.QueryNodesResponse{Nodes: nodes, Pagination: pageResp}, nil
}

func (q Querier) NodesByOperator(ctx context.Context, req *types.QueryNodesByOperatorRequest) (*types.QueryNodesByOperatorResponse, error) {
	nodes, pageResp, err := query.CollectionPaginate(ctx, q.Keeper.OperatorNodes, req.Pagination,
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
