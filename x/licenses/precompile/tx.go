package licensesprecompile

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/vm"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	licensestypes "github.com/webstack-sdk/webstack/x/licenses/types"
)

// Transaction method names. Must match the function names in LicensesI.sol / abi.json.
const (
	CreateLicenseTypeMethod         = "createLicenseType"
	UpdateLicenseTypeMethod         = "updateLicenseType"
	GrantAdminPermissionsMethod     = "grantAdminPermissions"
	RevokeAdminKeyPermissionsMethod = "revokeAdminKeyPermissions"
	IssueLicensesMethod             = "issueLicenses"
	RevokeLicensesMethod            = "revokeLicenses"
	TransferLicenseMethod           = "transferLicense"
)

// CreateLicenseType handles the createLicenseType ABI method.
func (p Precompile) CreateLicenseType(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if err := argCount(args, 3); err != nil {
		return nil, err
	}
	id, err := argToString(args[0], "id")
	if err != nil {
		return nil, err
	}
	transferrable, err := argToBool(args[1], "transferrable")
	if err != nil {
		return nil, err
	}
	maxSupply, err := argToBigInt(args[2], "maxSupply")
	if err != nil {
		return nil, err
	}

	owner, err := hexToBech32(p.addrCdc, contract.Caller())
	if err != nil {
		return nil, err
	}

	msg := &licensestypes.MsgCreateLicenseType{
		Owner:         owner,
		Id:            id,
		Transferrable: transferrable,
		MaxSupply:     math.NewIntFromBigInt(maxSupply),
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if _, err := p.msgServer.CreateLicenseType(ctx, msg); err != nil {
		return nil, err
	}

	if err := p.EmitLicenseTypeCreated(ctx, stateDB, id, transferrable, maxSupply); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

// UpdateLicenseType handles the updateLicenseType ABI method.
func (p Precompile) UpdateLicenseType(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if err := argCount(args, 3); err != nil {
		return nil, err
	}
	id, err := argToString(args[0], "id")
	if err != nil {
		return nil, err
	}
	transferrable, err := argToBool(args[1], "transferrable")
	if err != nil {
		return nil, err
	}
	maxSupply, err := argToBigInt(args[2], "maxSupply")
	if err != nil {
		return nil, err
	}

	owner, err := hexToBech32(p.addrCdc, contract.Caller())
	if err != nil {
		return nil, err
	}

	msg := &licensestypes.MsgUpdateLicenseType{
		Owner:         owner,
		Id:            id,
		Transferrable: transferrable,
		MaxSupply:     math.NewIntFromBigInt(maxSupply),
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if _, err := p.msgServer.UpdateLicenseType(ctx, msg); err != nil {
		return nil, err
	}

	if err := p.EmitLicenseTypeUpdated(ctx, stateDB, id, transferrable, maxSupply); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

// GrantAdminPermissions handles the grantAdminPermissions ABI method.
func (p Precompile) GrantAdminPermissions(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if err := argCount(args, 2); err != nil {
		return nil, err
	}
	adminHex, err := argToAddress(args[0], "admin")
	if err != nil {
		return nil, err
	}

	rawGrants, ok := args[1].([]AdminKeyGrantArg)
	if !ok {
		return nil, fmt.Errorf("invalid type for grants: expected AdminKeyGrant[], got %T", args[1])
	}

	owner, err := hexToBech32(p.addrCdc, contract.Caller())
	if err != nil {
		return nil, err
	}
	adminBech, err := hexToBech32(p.addrCdc, adminHex)
	if err != nil {
		return nil, err
	}

	grants := make([]licensestypes.AdminKeyGrant, 0, len(rawGrants))
	for i, g := range rawGrants {
		perm, err := licensestypes.ParsePermission(g.Permission)
		if err != nil {
			return nil, fmt.Errorf("grant %d: %w", i, err)
		}
		grants = append(grants, licensestypes.AdminKeyGrant{
			Permission:   perm,
			LicenseTypes: append([]string{}, g.LicenseTypes...),
		})
	}

	msg := &licensestypes.MsgGrantAdminPermissions{
		Owner:   owner,
		Address: adminBech,
		Grants:  grants,
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if _, err := p.msgServer.GrantAdminPermissions(ctx, msg); err != nil {
		return nil, err
	}

	if err := p.EmitAdminPermissionsGranted(ctx, stateDB, adminHex); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

// RevokeAdminKeyPermissions handles the revokeAdminKeyPermissions ABI method.
func (p Precompile) RevokeAdminKeyPermissions(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if err := argCount(args, 2); err != nil {
		return nil, err
	}
	adminHex, err := argToAddress(args[0], "admin")
	if err != nil {
		return nil, err
	}

	rawPerms, ok := args[1].([]AdminKeyPermissionArg)
	if !ok {
		return nil, fmt.Errorf("invalid type for permissions: expected AdminKeyPermission[], got %T", args[1])
	}

	owner, err := hexToBech32(p.addrCdc, contract.Caller())
	if err != nil {
		return nil, err
	}
	adminBech, err := hexToBech32(p.addrCdc, adminHex)
	if err != nil {
		return nil, err
	}

	perms := make([]licensestypes.AdminKeyPermission, 0, len(rawPerms))
	for i, pp := range rawPerms {
		perm, err := licensestypes.ParsePermission(pp.Permission)
		if err != nil {
			return nil, fmt.Errorf("pair %d: %w", i, err)
		}
		perms = append(perms, licensestypes.AdminKeyPermission{
			LicenseTypeId: pp.LicenseTypeId,
			Permission:    perm,
		})
	}

	msg := &licensestypes.MsgRevokeAdminKeyPermissions{
		Owner:       owner,
		Address:     adminBech,
		Permissions: perms,
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if _, err := p.msgServer.RevokeAdminKeyPermissions(ctx, msg); err != nil {
		return nil, err
	}

	if err := p.EmitAdminKeyPermissionsRevoked(ctx, stateDB, adminHex); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

// IssueLicenses handles the issueLicenses ABI method.
func (p Precompile) IssueLicenses(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if err := argCount(args, 1); err != nil {
		return nil, err
	}

	rawEntries, ok := args[0].([]IssueLicenseEntryArg)
	if !ok {
		return nil, fmt.Errorf("invalid type for entries: expected IssueLicenseEntry[], got %T", args[0])
	}

	issuer, err := hexToBech32(p.addrCdc, contract.Caller())
	if err != nil {
		return nil, err
	}

	entries := make([]licensestypes.IssueLicenseEntry, 0, len(rawEntries))
	for i, e := range rawEntries {
		holder, err := hexToBech32(p.addrCdc, e.Holder)
		if err != nil {
			return nil, fmt.Errorf("entry %d: %w", i, err)
		}
		entries = append(entries, licensestypes.IssueLicenseEntry{
			LicenseTypeId: e.LicenseTypeId,
			Holder:        holder,
			StartDate:     e.StartDate,
			EndDate:       e.EndDate,
			Count:         e.Count,
		})
	}

	msg := &licensestypes.MsgIssueLicenses{
		Issuer:  issuer,
		Entries: entries,
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	res, err := p.msgServer.IssueLicenses(ctx, msg)
	if err != nil {
		return nil, err
	}

	for _, e := range rawEntries {
		if err := p.EmitLicenseIssued(ctx, stateDB, contract.Caller(), e.Holder, e.LicenseTypeId, e.Count); err != nil {
			return nil, err
		}
	}

	return method.Outputs.Pack(res.Ids)
}

// RevokeLicenses handles the revokeLicenses ABI method.
func (p Precompile) RevokeLicenses(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if err := argCount(args, 3); err != nil {
		return nil, err
	}
	licenseTypeID, err := argToString(args[0], "licenseTypeId")
	if err != nil {
		return nil, err
	}
	holderHex, err := argToAddress(args[1], "holder")
	if err != nil {
		return nil, err
	}
	count, err := argToUint64(args[2], "count")
	if err != nil {
		return nil, err
	}

	revoker, err := hexToBech32(p.addrCdc, contract.Caller())
	if err != nil {
		return nil, err
	}
	holder, err := hexToBech32(p.addrCdc, holderHex)
	if err != nil {
		return nil, err
	}

	msg := &licensestypes.MsgRevokeLicenses{
		Revoker:       revoker,
		LicenseTypeId: licenseTypeID,
		Holder:        holder,
		Count:         count,
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	res, err := p.msgServer.RevokeLicenses(ctx, msg)
	if err != nil {
		return nil, err
	}

	emitted := count
	if emitted == 0 {
		emitted = uint64(len(res.Ids))
	}
	if err := p.EmitLicenseRevoked(ctx, stateDB, contract.Caller(), holderHex, licenseTypeID, emitted); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(res.Ids)
}

// TransferLicense handles the transferLicense ABI method.
func (p Precompile) TransferLicense(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if err := argCount(args, 3); err != nil {
		return nil, err
	}
	licenseTypeID, err := argToString(args[0], "licenseTypeId")
	if err != nil {
		return nil, err
	}
	id, err := argToUint64(args[1], "id")
	if err != nil {
		return nil, err
	}
	recipientHex, err := argToAddress(args[2], "recipient")
	if err != nil {
		return nil, err
	}

	holder, err := hexToBech32(p.addrCdc, contract.Caller())
	if err != nil {
		return nil, err
	}
	recipient, err := hexToBech32(p.addrCdc, recipientHex)
	if err != nil {
		return nil, err
	}

	msg := &licensestypes.MsgTransferLicense{
		Holder:        holder,
		LicenseTypeId: licenseTypeID,
		Id:            id,
		Recipient:     recipient,
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if _, err := p.msgServer.TransferLicense(ctx, msg); err != nil {
		return nil, err
	}

	if err := p.EmitLicenseTransferred(ctx, stateDB, contract.Caller(), recipientHex, licenseTypeID, id); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}
