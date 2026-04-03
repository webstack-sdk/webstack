package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgUpdateParams{}, "webstack/x/licenses/MsgUpdateParams", nil)
	cdc.RegisterConcrete(&MsgCreateLicenseType{}, "webstack/x/licenses/MsgCreateLicenseType", nil)
	cdc.RegisterConcrete(&MsgSetAdminKey{}, "webstack/x/licenses/MsgSetAdminKey", nil)
	cdc.RegisterConcrete(&MsgRemoveAdminKey{}, "webstack/x/licenses/MsgRemoveAdminKey", nil)
	cdc.RegisterConcrete(&MsgIssueLicense{}, "webstack/x/licenses/MsgIssueLicense", nil)
	cdc.RegisterConcrete(&MsgRevokeLicense{}, "webstack/x/licenses/MsgRevokeLicense", nil)
	cdc.RegisterConcrete(&MsgUpdateLicense{}, "webstack/x/licenses/MsgUpdateLicense", nil)
	cdc.RegisterConcrete(&MsgTransferLicense{}, "webstack/x/licenses/MsgTransferLicense", nil)
	cdc.RegisterConcrete(&MsgUpdateLicenseType{}, "webstack/x/licenses/MsgUpdateLicenseType", nil)
	cdc.RegisterConcrete(&MsgBatchIssueLicense{}, "webstack/x/licenses/MsgBatchIssueLicense", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgCreateLicenseType{},
		&MsgSetAdminKey{},
		&MsgRemoveAdminKey{},
		&MsgIssueLicense{},
		&MsgRevokeLicense{},
		&MsgUpdateLicense{},
		&MsgTransferLicense{},
		&MsgUpdateLicenseType{},
		&MsgBatchIssueLicense{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
