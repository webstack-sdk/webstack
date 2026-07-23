package keeper

import (
	"context"

	"cosmossdk.io/collections"

	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/webstack-sdk/webstack/x/permission/types"
)

var _ types.QueryServer = Querier{}

type Querier struct {
	Keeper
}

func NewQuerier(keeper Keeper) Querier {
	return Querier{Keeper: keeper}
}

func (q Querier) Namespaces(ctx context.Context, req *types.QueryNamespacesRequest) (*types.QueryNamespacesResponse, error) {
	namespaces, pageResp, err := query.CollectionPaginate(ctx, q.Keeper.Namespaces, req.Pagination,
		func(_ string, ns types.Namespace) (types.Namespace, error) {
			return ns, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return &types.QueryNamespacesResponse{Namespaces: namespaces, Pagination: pageResp}, nil
}

func (q Querier) Namespace(ctx context.Context, req *types.QueryNamespaceRequest) (*types.QueryNamespaceResponse, error) {
	ns, found, err := q.Keeper.GetNamespace(ctx, req.Module)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, types.ErrNamespaceNotFound.Wrapf("namespace for module %q not found", req.Module)
	}

	resp := &types.QueryNamespaceResponse{Namespace: ns}
	if spec, registered := q.Keeper.Spec(req.Module); registered {
		resp.Permissions = spec.SortedPermissions()
	}
	return resp, nil
}

// grantFromKey rebuilds the flat Grant view from a Grants keyset entry.
func grantFromKey(key collections.Quad[string, string, string, string]) types.Grant {
	return types.Grant{
		Module:     key.K1(),
		Grantee:    key.K2(),
		Permission: key.K3(),
		Scope:      key.K4(),
	}
}

// withQuadPrefix constrains pagination over a quad-keyed collection to keys
// whose first component equals k1.
func withQuadPrefix[K1, K2, K3, K4 any](k1 K1) func(o *query.CollectionsPaginateOptions[collections.Quad[K1, K2, K3, K4]]) {
	return func(o *query.CollectionsPaginateOptions[collections.Quad[K1, K2, K3, K4]]) {
		prefix := collections.QuadPrefix[K1, K2, K3, K4](k1)
		o.Prefix = &prefix
	}
}

// withQuadSuperPrefix constrains pagination over a quad-keyed collection to
// keys whose first two components equal (k1, k2).
func withQuadSuperPrefix[K1, K2, K3, K4 any](k1 K1, k2 K2) func(o *query.CollectionsPaginateOptions[collections.Quad[K1, K2, K3, K4]]) {
	return func(o *query.CollectionsPaginateOptions[collections.Quad[K1, K2, K3, K4]]) {
		prefix := collections.QuadSuperPrefix[K1, K2, K3, K4](k1, k2)
		o.Prefix = &prefix
	}
}

func (q Querier) Grants(ctx context.Context, req *types.QueryGrantsRequest) (*types.QueryGrantsResponse, error) {
	grants, pageResp, err := query.CollectionPaginate(ctx, q.Keeper.Grants, req.Pagination,
		func(key collections.Quad[string, string, string, string], _ collections.NoValue) (types.Grant, error) {
			return grantFromKey(key), nil
		},
		withQuadPrefix[string, string, string, string](req.Module),
	)
	if err != nil {
		return nil, err
	}
	return &types.QueryGrantsResponse{Grants: grants, Pagination: pageResp}, nil
}

func (q Querier) GrantsByGrantee(ctx context.Context, req *types.QueryGrantsByGranteeRequest) (*types.QueryGrantsByGranteeResponse, error) {
	grants, pageResp, err := query.CollectionPaginate(ctx, q.Keeper.Grants, req.Pagination,
		func(key collections.Quad[string, string, string, string], _ collections.NoValue) (types.Grant, error) {
			return grantFromKey(key), nil
		},
		withQuadSuperPrefix[string, string, string, string](req.Module, req.Grantee),
	)
	if err != nil {
		return nil, err
	}
	return &types.QueryGrantsByGranteeResponse{Grants: grants, Pagination: pageResp}, nil
}

// GrantsByScope walks the namespace's grants and keeps those matching the
// scope (and permission, when given). The scope is the last key component, so
// this is a filtered walk rather than a prefix read — same trade-off the
// license module made for its by-license-type query.
func (q Querier) GrantsByScope(ctx context.Context, req *types.QueryGrantsByScopeRequest) (*types.QueryGrantsByScopeResponse, error) {
	grants, pageResp, err := query.CollectionFilteredPaginate(ctx, q.Keeper.Grants, req.Pagination,
		func(key collections.Quad[string, string, string, string], _ collections.NoValue) (bool, error) {
			if key.K4() != req.Scope {
				return false, nil
			}
			if req.Permission != "" && key.K3() != req.Permission {
				return false, nil
			}
			return true, nil
		},
		func(key collections.Quad[string, string, string, string], _ collections.NoValue) (types.Grant, error) {
			return grantFromKey(key), nil
		},
		withQuadPrefix[string, string, string, string](req.Module),
	)
	if err != nil {
		return nil, err
	}
	return &types.QueryGrantsByScopeResponse{Grants: grants, Pagination: pageResp}, nil
}

func (q Querier) HasPermission(ctx context.Context, req *types.QueryHasPermissionRequest) (*types.QueryHasPermissionResponse, error) {
	has, err := q.Keeper.Has(ctx, req.Module, req.Grantee, req.Permission, req.Scope)
	if err != nil {
		return nil, err
	}
	return &types.QueryHasPermissionResponse{HasPermission: has}, nil
}
