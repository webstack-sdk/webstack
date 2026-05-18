package types

import (
	"fmt"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgUpdateParams{}
	_ sdk.Msg = &MsgCreateLicenseType{}
	_ sdk.Msg = &MsgGrantAdminPermissions{}
	_ sdk.Msg = &MsgRevokeAdminKeyPermissions{}
	_ sdk.Msg = &MsgIssueLicense{}
	_ sdk.Msg = &MsgRevokeLicense{}
	_ sdk.Msg = &MsgUpdateLicense{}
	_ sdk.Msg = &MsgTransferLicense{}
	_ sdk.Msg = &MsgUpdateLicenseType{}
	_ sdk.Msg = &MsgBatchIssueLicense{}
)

func (msg *MsgUpdateParams) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return ErrInvalidSigner.Wrapf("invalid authority address: %s", err)
	}
	return msg.Params.Validate()
}

func (msg *MsgCreateLicenseType) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return ErrInvalidSigner.Wrapf("invalid owner address: %s", err)
	}
	if msg.Id == "" {
		return ErrEmptyLicenseTypeID
	}
	return ValidateMaxSupply(msg.MaxSupply)
}

func (msg *MsgUpdateLicenseType) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return ErrInvalidSigner.Wrapf("invalid owner address: %s", err)
	}
	if msg.Id == "" {
		return ErrEmptyLicenseTypeID
	}
	return ValidateMaxSupply(msg.MaxSupply)
}

func ValidateMaxSupply(v math.Int) error {
	if v.IsNil() {
		return ErrInvalidMaxSupply.Wrap("max_supply must be set")
	}
	if v.IsNegative() {
		return ErrInvalidMaxSupply.Wrapf("max_supply must not be negative, got %s", v.String())
	}
	return nil
}

func (msg *MsgGrantAdminPermissions) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return ErrInvalidSigner.Wrapf("invalid owner address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return ErrInvalidSigner.Wrapf("invalid admin address: %s", err)
	}
	return nil
}

func (msg *MsgRevokeAdminKeyPermissions) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return ErrInvalidSigner.Wrapf("invalid owner address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return ErrInvalidSigner.Wrapf("invalid admin address: %s", err)
	}
	return nil
}

func (msg *MsgIssueLicense) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Issuer); err != nil {
		return ErrInvalidSigner.Wrapf("invalid issuer address: %s", err)
	}
	if msg.LicenseTypeId == "" {
		return ErrEmptyLicenseTypeID
	}
	if _, err := sdk.AccAddressFromBech32(msg.Holder); err != nil {
		return ErrEmptyHolder.Wrapf("invalid holder address: %s", err)
	}
	return nil
}

func (msg *MsgRevokeLicense) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Revoker); err != nil {
		return ErrInvalidSigner.Wrapf("invalid revoker address: %s", err)
	}
	if msg.LicenseTypeId == "" {
		return ErrEmptyLicenseTypeID
	}
	if _, err := sdk.AccAddressFromBech32(msg.Holder); err != nil {
		return ErrEmptyHolder.Wrapf("invalid holder address: %s", err)
	}
	return nil
}

func (msg *MsgUpdateLicense) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Updater); err != nil {
		return ErrInvalidSigner.Wrapf("invalid updater address: %s", err)
	}
	if msg.LicenseTypeId == "" {
		return ErrEmptyLicenseTypeID
	}
	if msg.Status != "active" && msg.Status != "revoked" {
		return ErrInvalidLicenseStatus.Wrapf("status must be 'active' or 'revoked', got '%s'", msg.Status)
	}
	return nil
}

func (msg *MsgTransferLicense) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Holder); err != nil {
		return ErrInvalidSigner.Wrapf("invalid holder address: %s", err)
	}
	if msg.LicenseTypeId == "" {
		return ErrEmptyLicenseTypeID
	}
	if _, err := sdk.AccAddressFromBech32(msg.Recipient); err != nil {
		return ErrEmptyHolder.Wrapf("invalid recipient address: %s", err)
	}
	return nil
}

func (msg *MsgBatchIssueLicense) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Issuer); err != nil {
		return ErrInvalidSigner.Wrapf("invalid issuer address: %s", err)
	}
	if msg.LicenseTypeId == "" {
		return ErrEmptyLicenseTypeID
	}
	if len(msg.Entries) == 0 {
		return ErrEmptyBatchEntries
	}
	if len(msg.Entries) > MaxIssueBatchSize {
		return fmt.Errorf("entries length %d exceeds max batch size %d", len(msg.Entries), MaxIssueBatchSize)
	}
	for _, entry := range msg.Entries {
		if _, err := sdk.AccAddressFromBech32(entry.Holder); err != nil {
			return ErrEmptyHolder.Wrapf("invalid holder address in batch entry: %s", err)
		}
	}
	return nil
}
