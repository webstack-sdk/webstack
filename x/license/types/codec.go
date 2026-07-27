package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreateLicenseType{}, "webstack/x/license/MsgCreateLicenseType", nil)
	cdc.RegisterConcrete(&MsgIssueLicenses{}, "webstack/x/license/MsgIssueLicenses", nil)
	cdc.RegisterConcrete(&MsgRevokeLicenses{}, "webstack/x/license/MsgRevokeLicenses", nil)
	cdc.RegisterConcrete(&MsgTransferLicense{}, "webstack/x/license/MsgTransferLicense", nil)
	cdc.RegisterConcrete(&MsgUpdateLicenseType{}, "webstack/x/license/MsgUpdateLicenseType", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateLicenseType{},
		&MsgIssueLicenses{},
		&MsgRevokeLicenses{},
		&MsgTransferLicense{},
		&MsgUpdateLicenseType{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
