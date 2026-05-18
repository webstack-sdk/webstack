package licensesprecompile

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"

	cmn "github.com/cosmos/evm/precompiles/common"
	sdk "github.com/cosmos/cosmos-sdk/types"

	licensestypes "github.com/webstack-sdk/webstack/x/licenses/types"
)

// AdminKeyGrantArg mirrors the Solidity AdminKeyGrant tuple.
//
// NOTE: this is a type *alias* so it matches the anonymous struct that
// go-ethereum's ABI decoder generates for the corresponding `tuple[]` input.
type AdminKeyGrantArg = struct {
	Permission   string   `json:"permission"`
	LicenseTypes []string `json:"licenseTypes"`
}

// BatchIssueEntryArg mirrors the Solidity BatchIssueEntry tuple.
//
// NOTE: alias type, see AdminKeyGrantArg.
type BatchIssueEntryArg = struct {
	Holder    common.Address `json:"holder"`
	StartDate string         `json:"startDate"`
	EndDate   string         `json:"endDate"`
}

// AdminKeyPermissionArg mirrors the Solidity AdminKeyPermission tuple
// (licenseTypeId, permission).
//
// NOTE: alias type, see AdminKeyGrantArg.
type AdminKeyPermissionArg = struct {
	LicenseTypeId string `json:"licenseTypeId"`
	Permission    string `json:"permission"`
}

// LicensesParamsOutput mirrors the Solidity LicensesParams tuple (output side).
type LicensesParamsOutput struct {
	Owner common.Address `abi:"owner"`
}

// LicenseTypeOutput mirrors the Solidity LicenseType tuple (output side).
type LicenseTypeOutput struct {
	Id            string   `abi:"id"`
	Transferrable bool     `abi:"transferrable"`
	MaxSupply     *big.Int `abi:"maxSupply"`
	IssuedCount   *big.Int `abi:"issuedCount"`
	ActiveCount   *big.Int `abi:"activeCount"`
	RevokedCount  *big.Int `abi:"revokedCount"`
}

// LicenseOutput mirrors the Solidity License tuple (output side).
type LicenseOutput struct {
	Id        uint64         `abi:"id"`
	TypeId    string         `abi:"typeId"`
	Holder    common.Address `abi:"holder"`
	StartDate string         `abi:"startDate"`
	EndDate   string         `abi:"endDate"`
	Status    string         `abi:"status"`
}

// AdminKeyGrantOutput mirrors the Solidity AdminKeyGrant tuple (output side).
type AdminKeyGrantOutput struct {
	Permission   string   `abi:"permission"`
	LicenseTypes []string `abi:"licenseTypes"`
}

// AdminKeyOutput mirrors the Solidity AdminKey tuple (output side).
type AdminKeyOutput struct {
	AdminAddress common.Address        `abi:"adminAddress"`
	Grants       []AdminKeyGrantOutput `abi:"grants"`
}

// hexToBech32 converts an EVM address to its bech32 form using the chain's account codec.
func hexToBech32(addrCdc address.Codec, hex common.Address) (string, error) {
	bech, err := addrCdc.BytesToString(hex.Bytes())
	if err != nil {
		return "", fmt.Errorf("invalid address %s: %w", hex.Hex(), err)
	}
	return bech, nil
}

// bech32ToHex converts a bech32 account address to its 20-byte EVM form. An
// empty input returns the zero address with no error (the legitimate
// "unset" case, e.g. Params.Owner before SetParams). Any non-empty input
// that fails to decode returns an error so callers can surface state
// corruption instead of silently emitting a zero-address holder.
func bech32ToHex(bech string) (common.Address, error) {
	if bech == "" {
		return common.Address{}, nil
	}
	accAddr, err := sdk.AccAddressFromBech32(bech)
	if err != nil {
		return common.Address{}, fmt.Errorf("invalid bech32 address %q: %w", bech, err)
	}
	return common.BytesToAddress(accAddr.Bytes()), nil
}

// bigIntFromCosmosInt converts a (possibly nil) cosmos math.Int to a *big.Int,
// returning zero when the value is uninitialised.
func bigIntFromCosmosInt(v math.Int) *big.Int {
	if v.IsNil() {
		return new(big.Int)
	}
	return v.BigInt()
}

// licenseTypeToOutput converts an SDK LicenseType into its ABI counterpart.
func licenseTypeToOutput(lt licensestypes.LicenseType) LicenseTypeOutput {
	return LicenseTypeOutput{
		Id:            lt.Id,
		Transferrable: lt.Transferrable,
		MaxSupply:     bigIntFromCosmosInt(lt.MaxSupply),
		IssuedCount:   bigIntFromCosmosInt(lt.IssuedCount),
		ActiveCount:   bigIntFromCosmosInt(lt.ActiveCount),
		RevokedCount:  bigIntFromCosmosInt(lt.RevokedCount),
	}
}

// licenseToOutput converts an SDK License into its ABI counterpart. Returns
// an error if the stored holder bech32 is malformed.
func licenseToOutput(l licensestypes.License) (LicenseOutput, error) {
	holder, err := bech32ToHex(l.Holder)
	if err != nil {
		return LicenseOutput{}, fmt.Errorf("license (type=%s, id=%d): %w", l.Type, l.Id, err)
	}
	return LicenseOutput{
		Id:        l.Id,
		TypeId:    l.Type,
		Holder:    holder,
		StartDate: l.StartDate,
		EndDate:   l.EndDate,
		Status:    l.Status,
	}, nil
}

// licensesToOutputs converts a slice of SDK Licenses into their ABI
// counterparts, propagating the first decode error.
func licensesToOutputs(ls []licensestypes.License) ([]LicenseOutput, error) {
	out := make([]LicenseOutput, 0, len(ls))
	for _, l := range ls {
		o, err := licenseToOutput(l)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}

// adminKeysToOutputs converts a slice of SDK AdminKeys into their ABI
// counterparts, propagating the first decode error.
func adminKeysToOutputs(aks []licensestypes.AdminKey) ([]AdminKeyOutput, error) {
	out := make([]AdminKeyOutput, 0, len(aks))
	for _, ak := range aks {
		o, err := adminKeyToOutput(ak)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}

// adminKeyToOutput converts an SDK AdminKey into its ABI counterpart. Returns
// an error if the stored admin bech32 is malformed.
func adminKeyToOutput(ak licensestypes.AdminKey) (AdminKeyOutput, error) {
	addr, err := bech32ToHex(ak.Address)
	if err != nil {
		return AdminKeyOutput{}, fmt.Errorf("admin key %s: %w", ak.Address, err)
	}
	grants := make([]AdminKeyGrantOutput, 0, len(ak.Grants))
	for _, g := range ak.Grants {
		grants = append(grants, AdminKeyGrantOutput{
			Permission:   g.Permission,
			LicenseTypes: append([]string{}, g.LicenseTypes...),
		})
	}
	return AdminKeyOutput{
		AdminAddress: addr,
		Grants:       grants,
	}, nil
}

// argToString unwraps a generic ABI argument expected to be a string.
func argToString(arg interface{}, name string) (string, error) {
	v, ok := arg.(string)
	if !ok {
		return "", fmt.Errorf("invalid type for %s: expected string, got %T", name, arg)
	}
	return v, nil
}

// argToAddress unwraps a generic ABI argument expected to be a common.Address.
func argToAddress(arg interface{}, name string) (common.Address, error) {
	v, ok := arg.(common.Address)
	if !ok {
		return common.Address{}, fmt.Errorf("invalid type for %s: expected address, got %T", name, arg)
	}
	return v, nil
}

// argToBool unwraps a generic ABI argument expected to be a bool.
func argToBool(arg interface{}, name string) (bool, error) {
	v, ok := arg.(bool)
	if !ok {
		return false, fmt.Errorf("invalid type for %s: expected bool, got %T", name, arg)
	}
	return v, nil
}

// argToBigInt unwraps a generic ABI argument expected to be a *big.Int.
func argToBigInt(arg interface{}, name string) (*big.Int, error) {
	v, ok := arg.(*big.Int)
	if !ok || v == nil {
		return nil, fmt.Errorf("invalid type for %s: expected uint256, got %T", name, arg)
	}
	return v, nil
}

// argToUint64 unwraps a generic ABI argument expected to be a uint64.
func argToUint64(arg interface{}, name string) (uint64, error) {
	v, ok := arg.(uint64)
	if !ok {
		return 0, fmt.Errorf("invalid type for %s: expected uint64, got %T", name, arg)
	}
	return v, nil
}

// argCount validates that the right number of arguments were passed.
func argCount(args []interface{}, want int) error {
	if len(args) != want {
		return fmt.Errorf(cmn.ErrInvalidNumberOfArgs, want, len(args))
	}
	return nil
}
