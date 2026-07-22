package licenseprecompile

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	cmn "github.com/cosmos/evm/precompiles/common"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Event names. Must match the event names in LicenseI.sol / abi.json.
const (
	EventTypeLicenseTypeCreated = "LicenseTypeCreated"
	EventTypeLicenseTypeUpdated = "LicenseTypeUpdated"
	EventTypePermissionsGranted = "PermissionsGranted"
	EventTypePermissionsRevoked = "PermissionsRevoked"
	EventTypeLicenseIssued      = "LicenseIssued"
	EventTypeLicenseRevoked     = "LicenseRevoked"
	EventTypeLicenseTransferred = "LicenseTransferred"
)

// emitLog writes an EVM log entry to the stateDB.
func (p Precompile) emitLog(ctx sdk.Context, stateDB vm.StateDB, topics []common.Hash, data []byte) {
	stateDB.AddLog(&ethtypes.Log{
		Address:     p.Address(),
		Topics:      topics,
		Data:        data,
		BlockNumber: uint64(ctx.BlockHeight()), //nolint:gosec // G115: block height is non-negative.
	})
}

// packArgs ABI-encodes the non-indexed event inputs.
func packArgs(event abi.Event, values ...interface{}) ([]byte, error) {
	nonIndexed := event.Inputs.NonIndexed()
	return nonIndexed.Pack(values...)
}

// EmitLicenseTypeCreated emits the LicenseTypeCreated event.
func (p Precompile) EmitLicenseTypeCreated(ctx sdk.Context, stateDB vm.StateDB, id string, transferrable bool, maxSupply *big.Int) error {
	event := p.Events[EventTypeLicenseTypeCreated]

	idTopic, err := cmn.MakeTopic(id)
	if err != nil {
		return err
	}

	data, err := packArgs(event, transferrable, maxSupply)
	if err != nil {
		return err
	}

	p.emitLog(ctx, stateDB, []common.Hash{event.ID, idTopic}, data)
	return nil
}

// EmitLicenseTypeUpdated emits the LicenseTypeUpdated event.
func (p Precompile) EmitLicenseTypeUpdated(ctx sdk.Context, stateDB vm.StateDB, id string, transferrable bool, maxSupply *big.Int) error {
	event := p.Events[EventTypeLicenseTypeUpdated]

	idTopic, err := cmn.MakeTopic(id)
	if err != nil {
		return err
	}

	data, err := packArgs(event, transferrable, maxSupply)
	if err != nil {
		return err
	}

	p.emitLog(ctx, stateDB, []common.Hash{event.ID, idTopic}, data)
	return nil
}

// EmitPermissionsGranted emits the PermissionsGranted event.
func (p Precompile) EmitPermissionsGranted(ctx sdk.Context, stateDB vm.StateDB, admin common.Address) error {
	event := p.Events[EventTypePermissionsGranted]

	adminTopic, err := cmn.MakeTopic(admin)
	if err != nil {
		return err
	}

	p.emitLog(ctx, stateDB, []common.Hash{event.ID, adminTopic}, nil)
	return nil
}

// EmitPermissionsRevoked emits the PermissionsRevoked event.
func (p Precompile) EmitPermissionsRevoked(ctx sdk.Context, stateDB vm.StateDB, admin common.Address) error {
	event := p.Events[EventTypePermissionsRevoked]

	adminTopic, err := cmn.MakeTopic(admin)
	if err != nil {
		return err
	}

	p.emitLog(ctx, stateDB, []common.Hash{event.ID, adminTopic}, nil)
	return nil
}

// EmitLicenseIssued emits the LicenseIssued event.
func (p Precompile) EmitLicenseIssued(ctx sdk.Context, stateDB vm.StateDB, issuer, holder common.Address, licenseTypeID string, count uint64) error {
	event := p.Events[EventTypeLicenseIssued]

	issuerTopic, err := cmn.MakeTopic(issuer)
	if err != nil {
		return err
	}
	holderTopic, err := cmn.MakeTopic(holder)
	if err != nil {
		return err
	}

	data, err := packArgs(event, licenseTypeID, count)
	if err != nil {
		return err
	}

	p.emitLog(ctx, stateDB, []common.Hash{event.ID, issuerTopic, holderTopic}, data)
	return nil
}

// EmitLicenseRevoked emits the LicenseRevoked event.
func (p Precompile) EmitLicenseRevoked(ctx sdk.Context, stateDB vm.StateDB, revoker, holder common.Address, licenseTypeID string, count uint64) error {
	event := p.Events[EventTypeLicenseRevoked]

	revokerTopic, err := cmn.MakeTopic(revoker)
	if err != nil {
		return err
	}
	holderTopic, err := cmn.MakeTopic(holder)
	if err != nil {
		return err
	}

	data, err := packArgs(event, licenseTypeID, count)
	if err != nil {
		return err
	}

	p.emitLog(ctx, stateDB, []common.Hash{event.ID, revokerTopic, holderTopic}, data)
	return nil
}

// EmitLicenseTransferred emits the LicenseTransferred event.
func (p Precompile) EmitLicenseTransferred(ctx sdk.Context, stateDB vm.StateDB, from, to common.Address, licenseTypeID string, id uint64) error {
	event := p.Events[EventTypeLicenseTransferred]

	fromTopic, err := cmn.MakeTopic(from)
	if err != nil {
		return err
	}
	toTopic, err := cmn.MakeTopic(to)
	if err != nil {
		return err
	}

	data, err := packArgs(event, licenseTypeID, id)
	if err != nil {
		return err
	}

	p.emitLog(ctx, stateDB, []common.Hash{event.ID, fromTopic, toTopic}, data)
	return nil
}
