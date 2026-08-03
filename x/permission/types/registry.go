package types

import (
	"context"
	"fmt"
	"sort"
)

// ScopeExistsFn reports whether a scope identifier refers to an existing
// resource in the consuming module (e.g. a license type id). It is consulted
// at grant time and during genesis import.
type ScopeExistsFn func(ctx context.Context, scope string) (bool, error)

// NamespaceSpec is the in-process registration a consuming module supplies at
// wiring time: its valid permission vocabulary and, optionally, a scope
// validator. It is static configuration, not state — every node registers the
// same specs during app construction, so consulting them is deterministic.
type NamespaceSpec struct {
	// Permissions is the full set of permission names the module uses.
	Permissions []string

	// ScopeExists validates scope identifiers at grant time. When nil, scopes
	// are unconstrained opaque strings and may be empty (module-wide grants).
	// When set, scopes must be non-empty and pass the check — except for the
	// permissions named in Unscoped.
	ScopeExists ScopeExistsFn

	// Unscoped names the permissions that are module-wide even though the
	// namespace otherwise scopes. Grants for these must carry the empty scope
	// and ScopeExists is not consulted for them.
	//
	// It exists for permissions that have no resource to point at — the right
	// to create the very resources the other permissions are scoped to cannot
	// name one, since it does not exist yet. Without this, such a permission
	// is inexpressible in a scoped namespace.
	//
	// Only meaningful alongside a non-nil ScopeExists: a namespace with no
	// scope validator is already module-wide throughout.
	Unscoped []string
}

// Validate checks the spec is well-formed: at least one permission, each
// permission a valid name, no duplicates, and every unscoped permission a
// member of the vocabulary that the namespace actually scopes.
func (s NamespaceSpec) Validate() error {
	if len(s.Permissions) == 0 {
		return fmt.Errorf("namespace spec must declare at least one permission")
	}
	seen := make(map[string]struct{}, len(s.Permissions))
	for _, p := range s.Permissions {
		if err := ValidateName("permission", p); err != nil {
			return err
		}
		if _, dup := seen[p]; dup {
			return fmt.Errorf("duplicate permission %q", p)
		}
		seen[p] = struct{}{}
	}

	if len(s.Unscoped) == 0 {
		return nil
	}
	// Both remaining checks reject specs where Unscoped changes nothing, on
	// the grounds that declaring it means the author expected it to.
	if s.ScopeExists == nil {
		return fmt.Errorf("unscoped permissions declared without a scope validator: every permission in this namespace is already module-wide")
	}
	unscoped := make(map[string]struct{}, len(s.Unscoped))
	for _, p := range s.Unscoped {
		if _, ok := seen[p]; !ok {
			return fmt.Errorf("unscoped permission %q is not in the namespace vocabulary", p)
		}
		if _, dup := unscoped[p]; dup {
			return fmt.Errorf("duplicate unscoped permission %q", p)
		}
		unscoped[p] = struct{}{}
	}
	if len(unscoped) == len(seen) {
		return fmt.Errorf("all %d permissions are declared unscoped: drop ScopeExists instead", len(seen))
	}
	return nil
}

// HasPermission reports whether p is in the spec's vocabulary.
func (s NamespaceSpec) HasPermission(p string) bool {
	for _, sp := range s.Permissions {
		if sp == p {
			return true
		}
	}
	return false
}

// IsUnscoped reports whether p is granted module-wide despite the namespace
// carrying a scope validator. Always false when Unscoped is empty, which
// includes every namespace that does not scope at all.
func (s NamespaceSpec) IsUnscoped(p string) bool {
	for _, up := range s.Unscoped {
		if up == p {
			return true
		}
	}
	return false
}

// SortedPermissions returns the vocabulary in ascending order without
// mutating the spec.
func (s NamespaceSpec) SortedPermissions() []string {
	out := make([]string, len(s.Permissions))
	copy(out, s.Permissions)
	sort.Strings(out)
	return out
}
