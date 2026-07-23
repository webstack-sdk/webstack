package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Namespaces: []Namespace{},
		Grants:     []Grant{},
	}
}

// Validate performs stateless genesis validation: shape and referential
// integrity within the genesis document itself. Checks that need the
// in-process namespace registry (permission vocabulary, scope existence)
// happen in the keeper's InitGenesis, after wiring.
func (gs GenesisState) Validate() error {
	modules := make(map[string]struct{}, len(gs.Namespaces))
	for _, ns := range gs.Namespaces {
		if err := ValidateName("module", ns.Module); err != nil {
			return fmt.Errorf("namespace: %w", err)
		}
		if _, dup := modules[ns.Module]; dup {
			return fmt.Errorf("duplicate namespace for module %q", ns.Module)
		}
		modules[ns.Module] = struct{}{}

		if _, err := sdk.AccAddressFromBech32(ns.Owner); err != nil {
			return fmt.Errorf("namespace %q has invalid owner address %q: %w", ns.Module, ns.Owner, err)
		}
	}

	grantKeys := make(map[string]struct{}, len(gs.Grants))
	for i, g := range gs.Grants {
		if _, exists := modules[g.Module]; !exists {
			return fmt.Errorf("grant %d references unknown namespace %q", i, g.Module)
		}
		if _, err := sdk.AccAddressFromBech32(g.Grantee); err != nil {
			return fmt.Errorf("grant %d has invalid grantee address %q: %w", i, g.Grantee, err)
		}
		if err := ValidateName("permission", g.Permission); err != nil {
			return fmt.Errorf("grant %d: %w", i, err)
		}

		key := fmt.Sprintf("%s/%s/%s/%s", g.Module, g.Grantee, g.Permission, g.Scope)
		if _, dup := grantKeys[key]; dup {
			return fmt.Errorf("duplicate grant (module=%s, grantee=%s, permission=%s, scope=%s)", g.Module, g.Grantee, g.Permission, g.Scope)
		}
		grantKeys[key] = struct{}{}
	}

	return nil
}
