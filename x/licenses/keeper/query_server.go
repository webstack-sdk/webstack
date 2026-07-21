package keeper

import (
	"context"
	"sort"

	"cosmossdk.io/collections"

	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/webstack-sdk/webstack/x/licenses/types"
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

func (q Querier) LicensesByType(ctx context.Context, req *types.QueryLicensesByTypeRequest) (*types.QueryLicensesByTypeResponse, error) {
	rng := collections.NewPrefixedPairRange[string, uint64](req.TypeId)
	var licenses []types.License

	err := q.Keeper.Licenses.Walk(ctx, rng, func(_ collections.Pair[string, uint64], l types.License) (bool, error) {
		licenses = append(licenses, l)
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryLicensesByTypeResponse{Licenses: licenses}, nil
}

// LicensesByHolder returns the holder's active licenses. Revoked licenses are
// not indexed by holder; they remain reachable via License / LicensesByType.
func (q Querier) LicensesByHolder(ctx context.Context, req *types.QueryLicensesByHolderRequest) (*types.QueryLicensesByHolderResponse, error) {
	rng := collections.NewPrefixedTripleRange[string, string, uint64](req.Holder)
	var licenses []types.License

	err := q.Keeper.ActiveLicensesByHolder.Walk(ctx, rng, func(key collections.Triple[string, string, uint64]) (bool, error) {
		l, err := q.Keeper.Licenses.Get(ctx, collections.Join(key.K2(), key.K3()))
		if err != nil {
			return true, err
		}
		licenses = append(licenses, l)
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryLicensesByHolderResponse{Licenses: licenses}, nil
}

// LicensesByHolderAndType returns the holder's active licenses of one type.
// Revoked licenses are not indexed by holder; they remain reachable via
// License / LicensesByType.
func (q Querier) LicensesByHolderAndType(ctx context.Context, req *types.QueryLicensesByHolderAndTypeRequest) (*types.QueryLicensesByHolderAndTypeResponse, error) {
	rng := collections.NewSuperPrefixedTripleRange[string, string, uint64](req.Holder, req.TypeId)
	var licenses []types.License

	err := q.Keeper.ActiveLicensesByHolder.Walk(ctx, rng, func(key collections.Triple[string, string, uint64]) (bool, error) {
		l, err := q.Keeper.Licenses.Get(ctx, collections.Join(key.K2(), key.K3()))
		if err != nil {
			return true, err
		}
		licenses = append(licenses, l)
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryLicensesByHolderAndTypeResponse{Licenses: licenses}, nil
}

func (q Querier) AdminKey(ctx context.Context, req *types.QueryAdminKeyRequest) (*types.QueryAdminKeyResponse, error) {
	ak, found, err := q.Keeper.GetAdminKey(ctx, req.Address)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, types.ErrAdminKeyNotFound.Wrapf("admin key for %s not found", req.Address)
	}
	return &types.QueryAdminKeyResponse{AdminKey: ak}, nil
}

// paginateAdminKeys applies address-level pagination to a grouped admin key
// slice (already in ascending address order). PageRequest.Key is an address:
// results start at the first entry >= that address; NextKey is the address of
// the first entry beyond the returned page. Offset is honored when Key is
// unset.
func paginateAdminKeys(adminKeys []types.AdminKey, page *query.PageRequest) ([]types.AdminKey, *query.PageResponse) {
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

func (q Querier) AdminKeys(ctx context.Context, req *types.QueryAdminKeysRequest) (*types.QueryAdminKeysResponse, error) {
	all, err := q.Keeper.GetAdminKeys(ctx)
	if err != nil {
		return nil, err
	}
	page, pageResp := paginateAdminKeys(all, req.Pagination)
	return &types.QueryAdminKeysResponse{AdminKeys: page, Pagination: pageResp}, nil
}

func (q Querier) AdminKeysByLicenseType(ctx context.Context, req *types.QueryAdminKeysByLicenseTypeRequest) (*types.QueryAdminKeysByLicenseTypeResponse, error) {
	all, err := q.Keeper.GetAdminKeys(ctx)
	if err != nil {
		return nil, err
	}

	matches := func(ak types.AdminKey) bool {
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

	var filtered []types.AdminKey
	for _, ak := range all {
		if matches(ak) {
			filtered = append(filtered, ak)
		}
	}

	page, pageResp := paginateAdminKeys(filtered, req.Pagination)
	return &types.QueryAdminKeysByLicenseTypeResponse{AdminKeys: page, Pagination: pageResp}, nil
}

func (q Querier) Permissions(_ context.Context, _ *types.QueryPermissionsRequest) (*types.QueryPermissionsResponse, error) {
	return &types.QueryPermissionsResponse{Permissions: types.ValidPermissionStrings()}, nil
}
