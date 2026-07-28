package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgCreateOperatorAccount{}
	_ sdk.Msg = &MsgAuthorizeActivationKey{}
	_ sdk.Msg = &MsgDeauthorizeActivationKey{}
	_ sdk.Msg = &MsgActivateNode{}
	_ sdk.Msg = &MsgDeactivateNode{}
	_ sdk.Msg = &MsgUpdateNodeStatus{}
	_ sdk.Msg = &MsgUpdateParams{}
)

// GaslessMessages returns the msg type URLs this module admits with a zero
// fee, for composing an app's gasless allowlist. Consuming chains append
// their own module's gasless msgs (e.g. attestation msgs).
func GaslessMessages() []string {
	return []string{
		sdk.MsgTypeURL(&MsgAuthorizeActivationKey{}),
		sdk.MsgTypeURL(&MsgActivateNode{}),
		sdk.MsgTypeURL(&MsgDeactivateNode{}),
		sdk.MsgTypeURL(&MsgUpdateNodeStatus{}),
	}
}

func (msg *MsgCreateOperatorAccount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Admin); err != nil {
		return ErrInvalidSigner.Wrapf("invalid admin address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Wallet); err != nil {
		return ErrInvalidSigner.Wrapf("invalid wallet address: %s", err)
	}
	return nil
}

func (msg *MsgAuthorizeActivationKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return ErrInvalidSigner.Wrapf("invalid operator address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.ActivationAddress); err != nil {
		return ErrInvalidSigner.Wrapf("invalid activation address: %s", err)
	}
	if msg.ActivationAddress == msg.Operator {
		return ErrSelfAuthorization
	}
	return nil
}

func (msg *MsgDeauthorizeActivationKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return ErrInvalidSigner.Wrapf("invalid operator address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.ActivationAddress); err != nil {
		return ErrInvalidSigner.Wrapf("invalid activation address: %s", err)
	}
	return nil
}

func (msg *MsgActivateNode) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.ActivationAddress); err != nil {
		return ErrInvalidSigner.Wrapf("invalid activation address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return ErrInvalidSigner.Wrapf("invalid operator address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.NodeAddress); err != nil {
		return ErrInvalidSigner.Wrapf("invalid node address: %s", err)
	}
	if msg.NodeType == "" {
		return ErrInvalidNodeType.Wrap("node type must not be empty")
	}
	return nil
}

func (msg *MsgDeactivateNode) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return ErrInvalidSigner.Wrapf("invalid signer address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.NodeAddress); err != nil {
		return ErrInvalidSigner.Wrapf("invalid node address: %s", err)
	}
	return nil
}

func (msg *MsgUpdateNodeStatus) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.NodeAddress); err != nil {
		return ErrInvalidSigner.Wrapf("invalid node address: %s", err)
	}
	return msg.Payload.Validate()
}

// Validate bounds the payload fields. The chain treats the contents as
// opaque; only sizes are checked.
func (p NodeStatusPayload) Validate() error {
	for name, v := range map[string]string{
		"device": p.Device, "os": p.Os, "hostname": p.Hostname,
		"node_info": p.NodeInfo, "memory": p.Memory, "cpu": p.Cpu,
		"storage": p.Storage,
	} {
		if len(v) > MaxStatusFieldLen {
			return ErrInvalidStatusPayload.Wrapf("%s exceeds %d bytes", name, MaxStatusFieldLen)
		}
	}
	if len(p.Workloads) > MaxStatusWorkloads {
		return ErrInvalidStatusPayload.Wrapf("workloads length %d exceeds %d entries", len(p.Workloads), MaxStatusWorkloads)
	}
	for i, w := range p.Workloads {
		if len(w) > MaxStatusFieldLen {
			return ErrInvalidStatusPayload.Wrapf("workloads[%d] exceeds %d bytes", i, MaxStatusFieldLen)
		}
	}
	return nil
}

func (msg *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return ErrInvalidSigner.Wrapf("invalid authority address: %s", err)
	}
	return msg.Params.Validate()
}
