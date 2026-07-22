package keeper

import (
	"context"
	"sort"

	"cosmossdk.io/collections"

	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/webstack-sdk/webstack/x/license/types"
)

var _ types.QueryServer = Querier{}

type Querier struct {
	Keeper
}

func NewQuerier(keeper Keeper) Querier {
	return Querier{Keeper: keeper}
}

func (q Querier) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	p, err := q.Keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &types.QueryParamsResponse{Params: p}, nil
}

func (q Querier) LicenseType(ctx context.Context, req *types.QueryLicenseTypeRequest) (*types.QueryLicenseTypeResponse, error) {
	lt, err := q.Keeper.LicenseTypes.Get(ctx, req.Id)
	if err != nil {
		return nil, types.ErrLicenseTypeNotFound.Wrapf("license type %s not found", req.Id)
	}
	return &types.QueryLicenseTypeResponse{LicenseType: lt}, nil
}

func (q Querier) LicenseTypes(ctx context.Context, req *types.QueryLicenseTypesRequest) (*types.QueryLicenseTypesResponse, error) {
	results, pageResp, err := query.CollectionPaginate(ctx, q.Keeper.LicenseTypes, req.Pagination,
		func(_ string, lt types.LicenseType) (types.LicenseType, error) {
			return lt, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return &types.QueryLicenseTypesResponse{LicenseTypes: results, Pagination: pageResp}, nil
}

func (q Querier) License(ctx context.Context, req *types.QueryLicenseRequest) (*types.QueryLicenseResponse, error) {
	l, err := q.Keeper.Licenses.Get(ctx, collections.Join(req.TypeId, req.Id))
	if err != nil {
		return nil, types.ErrLicenseNotFound.Wrapf("license %d of type %s not found", req.Id, req.TypeId)
	}
	return &types.QueryLicenseResponse{License: l}, nil
}

// Licenses returns every license across all license types, active and
// revoked, paginated over the (type_id, id) key space.
func (q Querier) Licenses(ctx context.Context, req *types.QueryLicensesRequest) (*types.QueryLicensesResponse, error) {
	licenses, pageResp, err := query.CollectionPaginate(ctx, q.Keeper.Licenses, req.Pagination,
		func(_ collections.Pair[string, uint64], l types.License) (types.License, error) {
			return l, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return &types.QueryLicensesResponse{Licenses: licenses, Pagination: pageResp}, nil
}

// withTriplePrefix constrains pagination over a triple-keyed collection to
// keys whose first component equals k1.
func withTriplePrefix[K1, K2, K3 any](k1 K1) func(o *query.CollectionsPaginateOptions[collections.Triple[K1, K2, K3]]) {
	return func(o *query.CollectionsPaginateOptions[collections.Triple[K1, K2, K3]]) {
		prefix := collections.TriplePrefix[K1, K2, K3](k1)
		o.Prefix = &prefix
	}
}

// withTripleSuperPrefix constrains pagination over a triple-keyed collection
// to keys whose first two components equal (k1, k2).
func withTripleSuperPrefix[K1, K2, K3 any](k1 K1, k2 K2) func(o *query.CollectionsPaginateOptions[collections.Triple[K1, K2, K3]]) {
	return func(o *query.CollectionsPaginateOptions[collections.Triple[K1, K2, K3]]) {
		prefix := collections.TripleSuperPrefix[K1, K2, K3](k1, k2)
		o.Prefix = &prefix
	}
}

func (q Querier) LicensesByType(ctx context.Context, req *types.QueryLicensesByTypeRequest) (*types.QueryLicensesByTypeResponse, error) {
	licenses, pageResp, err := query.CollectionPaginate(ctx, q.Keeper.Licenses, req.Pagination,
		func(_ collections.Pair[string, uint64], l types.License) (types.License, error) {
			return l, nil
		},
		query.WithCollectionPaginationPairPrefix[string, uint64](req.TypeId),
	)
	if err != nil {
		return nil, err
	}
	return &types.QueryLicensesByTypeResponse{Licenses: licenses, Pagination: pageResp}, nil
}

// LicensesByHolder returns the holder's active licenses. Revoked licenses are
// not indexed by holder; they remain reachable via License / LicensesByType.
func (q Querier) LicensesByHolder(ctx context.Context, req *types.QueryLicensesByHolderRequest) (*types.QueryLicensesByHolderResponse, error) {
	licenses, pageResp, err := query.CollectionPaginate(ctx, q.Keeper.ActiveLicensesByHolder, req.Pagination,
		func(key collections.Triple[string, string, uint64], _ collections.NoValue) (types.License, error) {
			return q.Keeper.Licenses.Get(ctx, collections.Join(key.K2(), key.K3()))
		},
		withTriplePrefix[string, string, uint64](req.Holder),
	)
	if err != nil {
		return nil, err
	}
	return &types.QueryLicensesByHolderResponse{Licenses: licenses, Pagination: pageResp}, nil
}

// LicensesByHolderAndType returns the holder's active licenses of one type.
// Revoked licenses are not indexed by holder; they remain reachable via
// License / LicensesByType.
func (q Querier) LicensesByHolderAndType(ctx context.Context, req *types.QueryLicensesByHolderAndTypeRequest) (*types.QueryLicensesByHolderAndTypeResponse, error) {
	licenses, pageResp, err := query.CollectionPaginate(ctx, q.Keeper.ActiveLicensesByHolder, req.Pagination,
		func(key collections.Triple[string, string, uint64], _ collections.NoValue) (types.License, error) {
			return q.Keeper.Licenses.Get(ctx, collections.Join(key.K2(), key.K3()))
		},
		withTripleSuperPrefix[string, string, uint64](req.Holder, req.TypeId),
	)
	if err != nil {
		return nil, err
	}
	return &types.QueryLicensesByHolderAndTypeResponse{Licenses: licenses, Pagination: pageResp}, nil
}

func (q Querier) PermissionsByAddress(ctx context.Context, req *types.QueryPermissionsByAddressRequest) (*types.QueryPermissionsByAddressResponse, error) {
	ak, found, err := q.Keeper.GetPermissionsByAddress(ctx, req.Address)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, types.ErrPermissionsNotFound.Wrapf("permissions for %s not found", req.Address)
	}
	return &types.QueryPermissionsByAddressResponse{Permissions: ak}, nil
}

// paginatePermissions applies address-level pagination to a grouped permissions entry
// slice (already in ascending address order). PageRequest.Key is an address:
// results start at the first entry >= that address; NextKey is the address of
// the first entry beyond the returned page. Offset is honored when Key is
// unset.
func paginatePermissions(adminKeys []types.AddressPermissions, page *query.PageRequest) ([]types.AddressPermissions, *query.PageResponse) {
	limit := uint64(query.DefaultLimit)
	var offset uint64
	var startAddr string
	if page != nil {
		if page.Limit > 0 {
			limit = page.Limit
		}
		if len(page.Key) > 0 {
			startAddr = string(page.Key)
		} else {
			offset = page.Offset
		}
	}

	start := 0
	if startAddr != "" {
		start = sort.Search(len(adminKeys), func(i int) bool { return adminKeys[i].Address >= startAddr })
	} else if offset > 0 {
		if offset > uint64(len(adminKeys)) {
			offset = uint64(len(adminKeys))
		}
		start = int(offset)
	}

	end := start + int(limit)
	if end > len(adminKeys) || end < start {
		end = len(adminKeys)
	}

	pageResp := &query.PageResponse{}
	if end < len(adminKeys) {
		pageResp.NextKey = []byte(adminKeys[end].Address)
	}
	return adminKeys[start:end], pageResp
}

func (q Querier) Permissions(ctx context.Context, req *types.QueryPermissionsRequest) (*types.QueryPermissionsResponse, error) {
	all, err := q.Keeper.GetAllPermissions(ctx)
	if err != nil {
		return nil, err
	}
	page, pageResp := paginatePermissions(all, req.Pagination)
	return &types.QueryPermissionsResponse{Permissions: page, Pagination: pageResp}, nil
}

func (q Querier) PermissionsByLicenseType(ctx context.Context, req *types.QueryPermissionsByLicenseTypeRequest) (*types.QueryPermissionsByLicenseTypeResponse, error) {
	all, err := q.Keeper.GetAllPermissions(ctx)
	if err != nil {
		return nil, err
	}

	matches := func(ak types.AddressPermissions) bool {
		for _, grant := range ak.Grants {
			// The request carries the lowercase boundary form ("issue").
			if req.Permission != "" && grant.Permission.Short() != req.Permission {
				continue
			}
			for _, lt := range grant.LicenseTypes {
				if lt == req.LicenseTypeId {
					return true
				}
			}
		}
		return false
	}

	var filtered []types.AddressPermissions
	for _, ak := range all {
		if matches(ak) {
			filtered = append(filtered, ak)
		}
	}

	page, pageResp := paginatePermissions(filtered, req.Pagination)
	return &types.QueryPermissionsByLicenseTypeResponse{Permissions: page, Pagination: pageResp}, nil
}
