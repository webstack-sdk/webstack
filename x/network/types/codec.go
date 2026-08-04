package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreateOperatorAccount{}, "webstack/x/network/MsgCreateOperatorAccount", nil)
	cdc.RegisterConcrete(&MsgCreateNodeType{}, "webstack/x/network/MsgCreateNodeType", nil)
	cdc.RegisterConcrete(&MsgAuthorizeActivationKey{}, "webstack/x/network/MsgAuthorizeActivationKey", nil)
	cdc.RegisterConcrete(&MsgDeauthorizeActivationKey{}, "webstack/x/network/MsgDeauthorizeActivationKey", nil)
	cdc.RegisterConcrete(&MsgActivateNode{}, "webstack/x/network/MsgActivateNode", nil)
	cdc.RegisterConcrete(&MsgDeactivateNode{}, "webstack/x/network/MsgDeactivateNode", nil)
	cdc.RegisterConcrete(&MsgUpdateNodeStatus{}, "webstack/x/network/MsgUpdateNodeStatus", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "webstack/x/network/MsgUpdateParams", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateOperatorAccount{},
		&MsgCreateNodeType{},
		&MsgAuthorizeActivationKey{},
		&MsgDeauthorizeActivationKey{},
		&MsgActivateNode{},
		&MsgDeactivateNode{},
		&MsgUpdateNodeStatus{},
		&MsgUpdateParams{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
