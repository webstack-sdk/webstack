package types

import (
	"fmt"

	"cosmossdk.io/collections"
)

const (
	ModuleName   = "permission"
	StoreKey     = ModuleName
	RouterKey    = ModuleName
	QuerierRoute = ModuleName
)

// MaxGrants bounds the per-message slice lengths for grant operations: the
// top-level Grants/Permissions lists on MsgGrantPermissions /
// MsgRevokePermissions, and the inner Scopes slice within each grant.
const MaxGrants = 100

var (
	NamespacePrefix = collections.NewPrefix(1)
	GrantPrefix     = collections.NewPrefix(2)
)

// ValidateName validates a module or permission name: non-empty, lowercase
// alphanumeric plus '.', '_' and '-'. The character set deliberately excludes
// the ',' and ':' delimiters used at the CLI boundary.
func ValidateName(kind, s string) error {
	if s == "" {
		return fmt.Errorf("%s must not be empty", kind)
	}
	for _, c := range s {
		switch {
		case c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9':
		case c == '.' || c == '_' || c == '-':
		default:
			return fmt.Errorf("invalid %s %q: only lowercase letters, digits, '.', '_' and '-' are allowed", kind, s)
		}
	}
	return nil
}
