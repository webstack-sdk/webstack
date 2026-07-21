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
	cdc.RegisterConcrete(&MsgGrantPermissions{}, "webstack/x/licenses/MsgGrantPermissions", nil)
	cdc.RegisterConcrete(&MsgRevokePermissions{}, "webstack/x/licenses/MsgRevokePermissions", nil)
	cdc.RegisterConcrete(&MsgIssueLicenses{}, "webstack/x/licenses/MsgIssueLicenses", nil)
	cdc.RegisterConcrete(&MsgRevokeLicenses{}, "webstack/x/licenses/MsgRevokeLicenses", nil)
	cdc.RegisterConcrete(&MsgTransferLicense{}, "webstack/x/licenses/MsgTransferLicense", nil)
	cdc.RegisterConcrete(&MsgUpdateLicenseType{}, "webstack/x/licenses/MsgUpdateLicenseType", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgCreateLicenseType{},
		&MsgGrantPermissions{},
		&MsgRevokePermissions{},
		&MsgIssueLicenses{},
		&MsgRevokeLicenses{},
		&MsgTransferLicense{},
		&MsgUpdateLicenseType{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
