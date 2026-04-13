package keeper

import (
	"context"

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

func (q Querier) LicensesByHolder(ctx context.Context, req *types.QueryLicensesByHolderRequest) (*types.QueryLicensesByHolderResponse, error) {
	rng := collections.NewPrefixedTripleRange[string, string, uint64](req.Holder)
	var licenses []types.License

	err := q.Keeper.LicenseByHolder.Walk(ctx, rng, func(key collections.Triple[string, string, uint64], _ uint64) (bool, error) {
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

func (q Querier) LicensesByHolderAndType(ctx context.Context, req *types.QueryLicensesByHolderAndTypeRequest) (*types.QueryLicensesByHolderAndTypeResponse, error) {
	rng := collections.NewSuperPrefixedTripleRange[string, string, uint64](req.Holder, req.TypeId)
	var licenses []types.License

	err := q.Keeper.LicenseByHolder.Walk(ctx, rng, func(key collections.Triple[string, string, uint64], _ uint64) (bool, error) {
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
	ak, err := q.Keeper.AdminKeys.Get(ctx, req.Address)
	if err != nil {
		return nil, types.ErrAdminKeyNotFound.Wrapf("admin key for %s not found", req.Address)
	}
	return &types.QueryAdminKeyResponse{AdminKey: ak}, nil
}

func (q Querier) AdminKeys(ctx context.Context, req *types.QueryAdminKeysRequest) (*types.QueryAdminKeysResponse, error) {
	results, pageResp, err := query.CollectionPaginate(ctx, q.Keeper.AdminKeys, req.Pagination,
		func(_ string, ak types.AdminKey) (types.AdminKey, error) {
			return ak, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return &types.QueryAdminKeysResponse{AdminKeys: results, Pagination: pageResp}, nil
}

func (q Querier) AdminKeysByLicenseType(ctx context.Context, req *types.QueryAdminKeysByLicenseTypeRequest) (*types.QueryAdminKeysByLicenseTypeResponse, error) {
	var filtered []types.AdminKey

	_, pageResp, err := query.CollectionPaginate(ctx, q.Keeper.AdminKeys, req.Pagination,
		func(_ string, ak types.AdminKey) (types.AdminKey, error) {
			for _, grant := range ak.Grants {
				if req.Permission != "" && grant.Permission != req.Permission {
					continue
				}
				for _, lt := range grant.LicenseTypes {
					if lt == req.LicenseTypeId {
						filtered = append(filtered, ak)
						return ak, nil
					}
				}
			}
			return ak, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return &types.QueryAdminKeysByLicenseTypeResponse{AdminKeys: filtered, Pagination: pageResp}, nil
}

func (q Querier) Permissions(_ context.Context, _ *types.QueryPermissionsRequest) (*types.QueryPermissionsResponse, error) {
	return &types.QueryPermissionsResponse{Permissions: types.Permissions}, nil
}
