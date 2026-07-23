package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgCreateNamespace{}
	_ sdk.Msg = &MsgUpdateNamespaceOwner{}
	_ sdk.Msg = &MsgTransferOwnership{}
	_ sdk.Msg = &MsgGrantPermissions{}
	_ sdk.Msg = &MsgRevokePermissions{}
)

func (msg *MsgCreateNamespace) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return ErrInvalidSigner.Wrapf("invalid authority address: %s", err)
	}
	if err := ValidateName("module", msg.Module); err != nil {
		return err
	}
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	return nil
}

func (msg *MsgUpdateNamespaceOwner) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return ErrInvalidSigner.Wrapf("invalid authority address: %s", err)
	}
	if err := ValidateName("module", msg.Module); err != nil {
		return err
	}
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	return nil
}

func (msg *MsgTransferOwnership) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return ErrInvalidSigner.Wrapf("invalid owner address: %s", err)
	}
	if err := ValidateName("module", msg.Module); err != nil {
		return err
	}
	if _, err := sdk.AccAddressFromBech32(msg.NewOwner); err != nil {
		return fmt.Errorf("invalid new owner address: %w", err)
	}
	return nil
}

func (msg *MsgGrantPermissions) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return ErrInvalidSigner.Wrapf("invalid owner address: %s", err)
	}
	if err := ValidateName("module", msg.Module); err != nil {
		return err
	}
	if _, err := sdk.AccAddressFromBech32(msg.Grantee); err != nil {
		return fmt.Errorf("invalid grantee address: %w", err)
	}
	if len(msg.Grants) == 0 {
		return fmt.Errorf("grants must not be empty")
	}
	if len(msg.Grants) > MaxGrants {
		return fmt.Errorf("grants length %d exceeds max %d", len(msg.Grants), MaxGrants)
	}
	for i, g := range msg.Grants {
		if err := ValidateName("permission", g.Permission); err != nil {
			return fmt.Errorf("grant %d: %w", i, err)
		}
		if len(g.Scopes) > MaxGrants {
			return fmt.Errorf("grant %d scopes length %d exceeds max %d", i, len(g.Scopes), MaxGrants)
		}
	}
	return nil
}

func (msg *MsgRevokePermissions) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return ErrInvalidSigner.Wrapf("invalid owner address: %s", err)
	}
	if err := ValidateName("module", msg.Module); err != nil {
		return err
	}
	if _, err := sdk.AccAddressFromBech32(msg.Grantee); err != nil {
		return fmt.Errorf("invalid grantee address: %w", err)
	}
	if len(msg.Permissions) == 0 {
		return fmt.Errorf("permissions must not be empty")
	}
	if len(msg.Permissions) > MaxGrants {
		return fmt.Errorf("permissions length %d exceeds max %d", len(msg.Permissions), MaxGrants)
	}
	for i, p := range msg.Permissions {
		if err := ValidateName("permission", p.Permission); err != nil {
			return fmt.Errorf("pair %d: %w", i, err)
		}
	}
	return nil
}
