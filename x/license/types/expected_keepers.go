package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
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
