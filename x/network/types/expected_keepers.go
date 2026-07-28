package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// LicenseKeeper is the x/license keeper surface the network module consumes.
type LicenseKeeper interface {
	// CountActiveLicenses returns the number of active licenses holder holds
	// across the given license types. When stopAt is non-zero the walk stops
	// as soon as the count reaches it, so gas is bounded by the check being
	// made rather than by holdings; stopAt zero counts everything.
	CountActiveLicenses(ctx context.Context, holder string, licenseTypes []string, stopAt uint64) (uint64, error)
}

// AccountKeeper is the x/auth keeper surface the network module consumes to
// create accounts for operator wallets, activation addresses, and node
// addresses. Pubkeys are set lazily by the stock SetPubKeyDecorator on each
// account's first signed tx, so accounts are created from the address only.
type AccountKeeper interface {
	HasAccount(ctx context.Context, addr sdk.AccAddress) bool
	NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	SetAccount(ctx context.Context, acc sdk.AccountI)
}

// BankKeeper is the x/bank keeper surface the network module consumes to
// charge the deauthorize fee.
type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
}

// PermissionKeeper is the x/permission keeper surface the network module
// consumes. The network module registers the "network" namespace at wiring
// time; grants are module-wide (no scope).
type PermissionKeeper interface {
	// Has reports whether grantee holds the (permission, scope) grant within
	// the module's namespace. A missing grant returns (false, nil); a store
	// error is surfaced.
	Has(ctx context.Context, module, grantee, permission, scope string) (bool, error)

	// IsOwner reports whether addr owns the module's namespace. A missing
	// namespace is surfaced as an error.
	IsOwner(ctx context.Context, module, addr string) (bool, error)
}
