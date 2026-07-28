package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccountKeeper is the x/auth keeper surface the license module consumes to
// create holder accounts on issuance, so a wallet holding only a license can
// sign its first transaction. Pubkeys are set lazily by the stock
// SetPubKeyDecorator, so accounts are created from the address only.
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	HasAccount(ctx context.Context, addr sdk.AccAddress) bool
	NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	SetAccount(ctx context.Context, acc sdk.AccountI)
}

// PermissionKeeper is the x/permission keeper surface the license module
// consumes. The license module registers the "license" namespace at wiring
// time; ownership and (permission, license type) grants live there.
type PermissionKeeper interface {
	// Has reports whether grantee holds the (permission, scope) grant within
	// the module's namespace. A missing grant returns (false, nil); a store
	// error is surfaced.
	Has(ctx context.Context, module, grantee, permission, scope string) (bool, error)

	// IsOwner reports whether addr owns the module's namespace. A missing
	// namespace is surfaced as an error.
	IsOwner(ctx context.Context, module, addr string) (bool, error)
}
