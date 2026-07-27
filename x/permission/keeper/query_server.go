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

// Modules returns every registered module's namespace, in ascending module
// order. The owner is joined from state and empty for modules whose owner has
// not been set yet.
func (q Querier) Modules(ctx context.Context, _ *types.QueryModulesRequest) (*types.QueryModulesResponse, error) {
	registered := q.Keeper.RegisteredModules()
	namespaces := make([]types.Namespace, 0, len(registered))
	for _, module := range registered {
		ns, found, err := q.Keeper.GetNamespace(ctx, module)
		if err != nil {
			return nil, err
		}
		if !found {
			ns = types.Namespace{Module: module}
		}
		namespaces = append(namespaces, ns)
	}
	return &types.QueryModulesResponse{Namespaces: namespaces}, nil
}

// Module returns a registered module's namespace plus the permission
// vocabulary it registered in this binary. The owner is empty if it has not
// been set yet.
func (q Querier) Module(ctx context.Context, req *types.QueryModuleRequest) (*types.QueryModuleResponse, error) {
	spec, registered := q.Keeper.Spec(req.Module)
	if !registered {
		return nil, types.ErrModuleNotRegistered.Wrapf("module %q is not registered in this binary", req.Module)
	}

	ns, found, err := q.Keeper.GetNamespace(ctx, req.Module)
	if err != nil {
		return nil, err
	}
	if !found {
		ns = types.Namespace{Module: req.Module}
	}

	return &types.QueryModuleResponse{
		Namespace:   ns,
		Permissions: spec.SortedPermissions(),
	}, nil
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
