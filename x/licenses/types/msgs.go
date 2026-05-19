package types

import (
	"fmt"
	"time"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateDates validates start_date and end_date strings in YYYY-MM-DD form.
// start_date is required; end_date is optional and, if present, must not be
// before start_date.
func ValidateDates(startDate, endDate string) error {
	if startDate == "" {
		return fmt.Errorf("start_date is required")
	}
	sd, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return fmt.Errorf("invalid start_date %q: must be YYYY-MM-DD format", startDate)
	}
	if endDate != "" {
		ed, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			return fmt.Errorf("invalid end_date %q: must be YYYY-MM-DD format", endDate)
		}
		if ed.Before(sd) {
			return fmt.Errorf("end_date %s must not be before start_date %s", endDate, startDate)
		}
	}
	return nil
}

var (
	_ sdk.Msg = &MsgUpdateParams{}
	_ sdk.Msg = &MsgCreateLicenseType{}
	_ sdk.Msg = &MsgGrantAdminPermissions{}
	_ sdk.Msg = &MsgRevokeAdminKeyPermissions{}
	_ sdk.Msg = &MsgIssueLicense{}
	_ sdk.Msg = &MsgRevokeLicense{}
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
	if len(msg.Grants) > MaxAdminGrants {
		return fmt.Errorf("grants length %d exceeds max %d", len(msg.Grants), MaxAdminGrants)
	}
	for i, g := range msg.Grants {
		if len(g.LicenseTypes) > MaxAdminGrants {
			return fmt.Errorf("grant %d license_types length %d exceeds max %d", i, len(g.LicenseTypes), MaxAdminGrants)
		}
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
	if len(msg.Permissions) > MaxAdminGrants {
		return fmt.Errorf("permissions length %d exceeds max %d", len(msg.Permissions), MaxAdminGrants)
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
	if msg.Count == 0 {
		return ErrInvalidCount.Wrap("count must be greater than zero")
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
	if msg.Count == 0 {
		return ErrInvalidCount.Wrap("count must be greater than zero")
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
