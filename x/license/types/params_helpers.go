package types

import sdk "github.com/cosmos/cosmos-sdk/types"

func DefaultParams() Params {
	return Params{
		Owner: "",
	}
}

func (p Params) Validate() error {
	if p.Owner == "" {
		return ErrInvalidSigner.Wrap("licenses module owner must be set")
	}
	_, err := sdk.AccAddressFromBech32(p.Owner)
	if err != nil {
		return ErrInvalidSigner.Wrapf("invalid owner address: %s", err)
	}
	return nil
}
