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
	// When set, scopes must be non-empty and pass the check.
	ScopeExists ScopeExistsFn
}

// Validate checks the spec is well-formed: at least one permission, each
// permission a valid name, no duplicates.
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

// SortedPermissions returns the vocabulary in ascending order without
// mutating the spec.
func (s NamespaceSpec) SortedPermissions() []string {
	out := make([]string, len(s.Permissions))
	copy(out, s.Permissions)
	sort.Strings(out)
	return out
}
