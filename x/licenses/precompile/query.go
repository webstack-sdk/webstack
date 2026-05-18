package licensesprecompile

import (
	"github.com/ethereum/go-ethereum/accounts/abi"

	sdk "github.com/cosmos/cosmos-sdk/types"

	licensestypes "github.com/webstack-sdk/webstack/x/licenses/types"
)

// Query method names. Must match the function names in LicensesI.sol / abi.json.
const (
	ParamsMethod                  = "params"
	PermissionsMethod             = "permissions"
	LicenseTypeMethod             = "licenseType"
	LicenseTypesMethod            = "licenseTypes"
	LicenseMethod                 = "license"
	LicensesByTypeMethod          = "licensesByType"
	LicensesByHolderMethod        = "licensesByHolder"
	LicensesByHolderAndTypeMethod = "licensesByHolderAndType"
	AdminKeyMethod                = "adminKey"
	AdminKeysMethod               = "adminKeys"
	AdminKeysByLicenseTypeMethod  = "adminKeysByLicenseType"
)

// Params returns module params as a LicensesParams tuple.
func (p Precompile) Params(ctx sdk.Context, method *abi.Method, args []interface{}) ([]byte, error) {
	if err := argCount(args, 0); err != nil {
		return nil, err
	}

	res, err := p.queryServer.Params(ctx, &licensestypes.QueryParamsRequest{})
	if err != nil {
		return nil, err
	}

	owner, err := bech32ToHex(res.Params.Owner)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(LicensesParamsOutput{Owner: owner})
}

// Permissions returns the list of valid grant permission strings.
func (p Precompile) Permissions(ctx sdk.Context, method *abi.Method, args []interface{}) ([]byte, error) {
	if err := argCount(args, 0); err != nil {
		return nil, err
	}

	res, err := p.queryServer.Permissions(ctx, &licensestypes.QueryPermissionsRequest{})
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(res.Permissions)
}

// LicenseType returns a single license type by id.
func (p Precompile) LicenseType(ctx sdk.Context, method *abi.Method, args []interface{}) ([]byte, error) {
	if err := argCount(args, 1); err != nil {
		return nil, err
	}
	id, err := argToString(args[0], "id")
	if err != nil {
		return nil, err
	}

	res, err := p.queryServer.LicenseType(ctx, &licensestypes.QueryLicenseTypeRequest{Id: id})
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(licenseTypeToOutput(res.LicenseType))
}

// LicenseTypes returns all license types.
func (p Precompile) LicenseTypes(ctx sdk.Context, method *abi.Method, args []interface{}) ([]byte, error) {
	if err := argCount(args, 0); err != nil {
		return nil, err
	}

	res, err := p.queryServer.LicenseTypes(ctx, &licensestypes.QueryLicenseTypesRequest{})
	if err != nil {
		return nil, err
	}

	out := make([]LicenseTypeOutput, 0, len(res.LicenseTypes))
	for _, lt := range res.LicenseTypes {
		out = append(out, licenseTypeToOutput(lt))
	}
	return method.Outputs.Pack(out)
}

// License returns a single license by type+id.
func (p Precompile) License(ctx sdk.Context, method *abi.Method, args []interface{}) ([]byte, error) {
	if err := argCount(args, 2); err != nil {
		return nil, err
	}
	typeID, err := argToString(args[0], "typeId")
	if err != nil {
		return nil, err
	}
	id, err := argToUint64(args[1], "id")
	if err != nil {
		return nil, err
	}

	res, err := p.queryServer.License(ctx, &licensestypes.QueryLicenseRequest{TypeId: typeID, Id: id})
	if err != nil {
		return nil, err
	}

	out, err := licenseToOutput(res.License)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(out)
}

// LicensesByType returns all licenses of a given type.
func (p Precompile) LicensesByType(ctx sdk.Context, method *abi.Method, args []interface{}) ([]byte, error) {
	if err := argCount(args, 1); err != nil {
		return nil, err
	}
	typeID, err := argToString(args[0], "typeId")
	if err != nil {
		return nil, err
	}

	res, err := p.queryServer.LicensesByType(ctx, &licensestypes.QueryLicensesByTypeRequest{TypeId: typeID})
	if err != nil {
		return nil, err
	}

	out, err := licensesToOutputs(res.Licenses)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(out)
}

// LicensesByHolder returns all licenses held by the given holder.
func (p Precompile) LicensesByHolder(ctx sdk.Context, method *abi.Method, args []interface{}) ([]byte, error) {
	if err := argCount(args, 1); err != nil {
		return nil, err
	}
	holderHex, err := argToAddress(args[0], "holder")
	if err != nil {
		return nil, err
	}
	holder, err := hexToBech32(p.addrCdc, holderHex)
	if err != nil {
		return nil, err
	}

	res, err := p.queryServer.LicensesByHolder(ctx, &licensestypes.QueryLicensesByHolderRequest{Holder: holder})
	if err != nil {
		return nil, err
	}

	out, err := licensesToOutputs(res.Licenses)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(out)
}

// LicensesByHolderAndType returns all licenses of a given type held by the given holder.
func (p Precompile) LicensesByHolderAndType(ctx sdk.Context, method *abi.Method, args []interface{}) ([]byte, error) {
	if err := argCount(args, 2); err != nil {
		return nil, err
	}
	holderHex, err := argToAddress(args[0], "holder")
	if err != nil {
		return nil, err
	}
	typeID, err := argToString(args[1], "typeId")
	if err != nil {
		return nil, err
	}
	holder, err := hexToBech32(p.addrCdc, holderHex)
	if err != nil {
		return nil, err
	}

	res, err := p.queryServer.LicensesByHolderAndType(ctx, &licensestypes.QueryLicensesByHolderAndTypeRequest{
		Holder: holder,
		TypeId: typeID,
	})
	if err != nil {
		return nil, err
	}

	out, err := licensesToOutputs(res.Licenses)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(out)
}

// AdminKey returns the admin key entry for an address.
func (p Precompile) AdminKey(ctx sdk.Context, method *abi.Method, args []interface{}) ([]byte, error) {
	if err := argCount(args, 1); err != nil {
		return nil, err
	}
	adminHex, err := argToAddress(args[0], "admin")
	if err != nil {
		return nil, err
	}
	admin, err := hexToBech32(p.addrCdc, adminHex)
	if err != nil {
		return nil, err
	}

	res, err := p.queryServer.AdminKey(ctx, &licensestypes.QueryAdminKeyRequest{Address: admin})
	if err != nil {
		return nil, err
	}

	out, err := adminKeyToOutput(res.AdminKey)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(out)
}

// AdminKeys returns all admin key entries.
func (p Precompile) AdminKeys(ctx sdk.Context, method *abi.Method, args []interface{}) ([]byte, error) {
	if err := argCount(args, 0); err != nil {
		return nil, err
	}

	res, err := p.queryServer.AdminKeys(ctx, &licensestypes.QueryAdminKeysRequest{})
	if err != nil {
		return nil, err
	}

	out, err := adminKeysToOutputs(res.AdminKeys)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(out)
}

// AdminKeysByLicenseType returns admin key entries that have `permission` over `licenseTypeId`.
func (p Precompile) AdminKeysByLicenseType(ctx sdk.Context, method *abi.Method, args []interface{}) ([]byte, error) {
	if err := argCount(args, 2); err != nil {
		return nil, err
	}
	licenseTypeID, err := argToString(args[0], "licenseTypeId")
	if err != nil {
		return nil, err
	}
	permission, err := argToString(args[1], "permission")
	if err != nil {
		return nil, err
	}

	res, err := p.queryServer.AdminKeysByLicenseType(ctx, &licensestypes.QueryAdminKeysByLicenseTypeRequest{
		LicenseTypeId: licenseTypeID,
		Permission:    permission,
	})
	if err != nil {
		return nil, err
	}

	out, err := adminKeysToOutputs(res.AdminKeys)
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(out)
}
