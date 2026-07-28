package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ValidateCanonicalAddress checks that addr is a valid bech32 account address
// **in its canonical encoding**.
//
// Decoding successfully is not enough. BIP-173 permits an all-uppercase
// encoding of the same payload, and the bech32 library normalizes rather than
// rejects it (cosmos/btcutil bech32.Normalize lowercases before verifying the
// checksum), so "TSC1ABC..." and "tsc1abc..." decode to identical bytes.
// Signature verification compares those bytes, so both forms authenticate as
// the same account — but this module keys state on the address *string*, so
// the two encodings would be two distinct identities in the store.
//
// That difference is load-bearing: node addresses and activation addresses are
// single-use, and it is the presence of their record that burns them
// (Nodes.Has / ActivationKeys.Has). Accepting a non-canonical encoding would
// let a caller re-activate a deactivated node or re-authorize a deauthorized
// activation address simply by changing the case. Requiring the canonical form
// collapses each account to exactly one key.
func ValidateCanonicalAddress(field, addr string) error {
	decoded, err := sdk.AccAddressFromBech32(addr)
	if err != nil {
		return ErrInvalidAddress.Wrapf("invalid %s address: %s", field, err)
	}
	// Re-encoding always produces the canonical (lowercase, correct-prefix)
	// form, so a mismatch means the input was a non-canonical alias.
	if decoded.String() != addr {
		return ErrInvalidAddress.Wrapf("%s address %q is not in canonical form", field, addr)
	}
	return nil
}
