package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:       DefaultParams(),
		LicenseTypes: []LicenseType{},
		Licenses:     []License{},
		AdminKeys:    []AdminKey{},
	}
}

func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	typeIDs := make(map[string]struct{})
	for _, lt := range gs.LicenseTypes {
		if _, exists := typeIDs[lt.Id]; exists {
			return fmt.Errorf("duplicate license type id: %s", lt.Id)
		}
		typeIDs[lt.Id] = struct{}{}
	}

	licenseKeys := make(map[string]struct{})
	for _, l := range gs.Licenses {
		key := fmt.Sprintf("%s/%d", l.Type, l.Id)
		if _, exists := licenseKeys[key]; exists {
			return fmt.Errorf("duplicate license (type=%s, id=%d)", l.Type, l.Id)
		}
		licenseKeys[key] = struct{}{}

		if _, exists := typeIDs[l.Type]; !exists {
			return fmt.Errorf("license (type=%s, id=%d) references unknown license type", l.Type, l.Id)
		}

		if l.Status != "active" && l.Status != "revoked" {
			return fmt.Errorf("license (type=%s, id=%d) has invalid status %q: must be \"active\" or \"revoked\"", l.Type, l.Id, l.Status)
		}

		if _, err := sdk.AccAddressFromBech32(l.Holder); err != nil {
			return fmt.Errorf("license (type=%s, id=%d) has invalid holder address %q: %w", l.Type, l.Id, l.Holder, err)
		}
	}

	adminAddrs := make(map[string]struct{})
	for _, ak := range gs.AdminKeys {
		if _, exists := adminAddrs[ak.Address]; exists {
			return fmt.Errorf("duplicate admin key address: %s", ak.Address)
		}
		adminAddrs[ak.Address] = struct{}{}

		if _, err := sdk.AccAddressFromBech32(ak.Address); err != nil {
			return fmt.Errorf("admin key has invalid address %q: %w", ak.Address, err)
		}
	}

	return nil
}
