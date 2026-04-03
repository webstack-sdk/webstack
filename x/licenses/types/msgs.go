package types

import sdk "github.com/cosmos/cosmos-sdk/types"

var (
	_ sdk.Msg = &MsgUpdateParams{}
	_ sdk.Msg = &MsgCreateLicenseType{}
	_ sdk.Msg = &MsgSetAdminKey{}
	_ sdk.Msg = &MsgRemoveAdminKey{}
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
	return nil
}

func (msg *MsgUpdateLicenseType) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return ErrInvalidSigner.Wrapf("invalid owner address: %s", err)
	}
	if msg.Id == "" {
		return ErrEmptyLicenseTypeID
	}
	return nil
}

func (msg *MsgSetAdminKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return ErrInvalidSigner.Wrapf("invalid owner address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return ErrInvalidSigner.Wrapf("invalid admin address: %s", err)
	}
	return nil
}

func (msg *MsgRemoveAdminKey) ValidateBasic() error {
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
	for _, entry := range msg.Entries {
		if _, err := sdk.AccAddressFromBech32(entry.Holder); err != nil {
			return ErrEmptyHolder.Wrapf("invalid holder address in batch entry: %s", err)
		}
	}
	return nil
}
