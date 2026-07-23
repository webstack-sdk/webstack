package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreateNamespace{}, "webstack/x/permission/MsgCreateNamespace", nil)
	cdc.RegisterConcrete(&MsgUpdateNamespaceOwner{}, "webstack/x/permission/MsgUpdateNamespaceOwner", nil)
	cdc.RegisterConcrete(&MsgTransferOwnership{}, "webstack/x/permission/MsgTransferOwnership", nil)
	cdc.RegisterConcrete(&MsgGrantPermissions{}, "webstack/x/permission/MsgGrantPermissions", nil)
	cdc.RegisterConcrete(&MsgRevokePermissions{}, "webstack/x/permission/MsgRevokePermissions", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateNamespace{},
		&MsgUpdateNamespaceOwner{},
		&MsgTransferOwnership{},
		&MsgGrantPermissions{},
		&MsgRevokePermissions{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
