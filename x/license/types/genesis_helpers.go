package types

import (
	"fmt"
	"time"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:        DefaultParams(),
		LicenseTypes:  []LicenseType{},
		Licenses:      []License{},
		Permissions:   []AddressPermissions{},
		LicenseCounts: []LicenseCount{},
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

		if err := ValidateMaxSupply(lt.MaxSupply); err != nil {
			return fmt.Errorf("license type %s: %w", lt.Id, err)
		}
		if err := validateNonNegativeCounter(lt.IssuedCount, "issued_count"); err != nil {
			return fmt.Errorf("license type %s: %w", lt.Id, err)
		}
		if err := validateNonNegativeCounter(lt.ActiveCount, "active_count"); err != nil {
			return fmt.Errorf("license type %s: %w", lt.Id, err)
		}
		if err := validateNonNegativeCounter(lt.RevokedCount, "revoked_count"); err != nil {
			return fmt.Errorf("license type %s: %w", lt.Id, err)
		}
	}

	// Pass 1: detect duplicate licenses. Done before per-license validation so
	// the duplicate error fires regardless of the offending record's date or
	// holder fields.
	licenseKeys := make(map[string]struct{})
	for _, l := range gs.Licenses {
		key := fmt.Sprintf("%s/%d", l.Type, l.Id)
		if _, exists := licenseKeys[key]; exists {
			return fmt.Errorf("duplicate license (type=%s, id=%d)", l.Type, l.Id)
		}
		licenseKeys[key] = struct{}{}
	}

	// Pass 2: per-license validation + per-type tally of active/revoked and
	// the highest id seen per type (for the counter invariant below).
	activeByType := make(map[string]uint64)
	revokedByType := make(map[string]uint64)
	maxIDByType := make(map[string]uint64)
	for _, l := range gs.Licenses {
		if _, exists := typeIDs[l.Type]; !exists {
			return fmt.Errorf("license (type=%s, id=%d) references unknown license type", l.Type, l.Id)
		}

		switch l.Status {
		case StatusActive:
			activeByType[l.Type]++
			if l.RevokedDate != "" {
				return fmt.Errorf("license (type=%s, id=%d) is active but has revoked_date %q", l.Type, l.Id, l.RevokedDate)
			}
		case StatusRevoked:
			revokedByType[l.Type]++
			if l.RevokedDate == "" {
				return fmt.Errorf("license (type=%s, id=%d) is revoked but has no revoked_date", l.Type, l.Id)
			}
			if _, err := time.Parse("2006-01-02", l.RevokedDate); err != nil {
				return fmt.Errorf("license (type=%s, id=%d) has invalid revoked_date %q: must be YYYY-MM-DD format", l.Type, l.Id, l.RevokedDate)
			}
		default:
			return fmt.Errorf("license (type=%s, id=%d) has invalid status %q", l.Type, l.Id, l.Status.String())
		}

		if l.Id > maxIDByType[l.Type] {
			maxIDByType[l.Type] = l.Id
		}

		if _, err := sdk.AccAddressFromBech32(l.Holder); err != nil {
			return fmt.Errorf("license (type=%s, id=%d) has invalid holder address %q: %w", l.Type, l.Id, l.Holder, err)
		}

		if err := ValidateDates(l.StartDate, l.EndDate); err != nil {
			return fmt.Errorf("license (type=%s, id=%d): %w", l.Type, l.Id, err)
		}
	}

	// Pass 3: per-type counter invariants must agree with the license set.
	for _, lt := range gs.LicenseTypes {
		wantActive := math.NewIntFromUint64(activeByType[lt.Id])
		wantRevoked := math.NewIntFromUint64(revokedByType[lt.Id])
		wantIssued := wantActive.Add(wantRevoked)

		if !lt.ActiveCount.Equal(wantActive) {
			return fmt.Errorf("license type %s: active_count %s does not match %s active license(s) in genesis", lt.Id, lt.ActiveCount.String(), wantActive.String())
		}
		if !lt.RevokedCount.Equal(wantRevoked) {
			return fmt.Errorf("license type %s: revoked_count %s does not match %s revoked license(s) in genesis", lt.Id, lt.RevokedCount.String(), wantRevoked.String())
		}
		if !lt.IssuedCount.Equal(wantIssued) {
			return fmt.Errorf("license type %s: issued_count %s does not match %s license(s) in genesis (active+revoked)", lt.Id, lt.IssuedCount.String(), wantIssued.String())
		}
	}

	// Pass 4: the per-type id sequence must exist for every type that has
	// licenses and must never be below the highest existing id — otherwise
	// the next issuance would overwrite an imported license.
	countByType := make(map[string]uint64)
	for _, lc := range gs.LicenseCounts {
		if _, exists := typeIDs[lc.LicenseTypeId]; !exists {
			return fmt.Errorf("license count references unknown license type %q", lc.LicenseTypeId)
		}
		if _, dup := countByType[lc.LicenseTypeId]; dup {
			return fmt.Errorf("duplicate license count for license type %q", lc.LicenseTypeId)
		}
		countByType[lc.LicenseTypeId] = lc.Count
	}
	for typeID, maxID := range maxIDByType {
		count, ok := countByType[typeID]
		if !ok {
			return fmt.Errorf("license type %s has licenses but no license count entry", typeID)
		}
		if count < maxID {
			return fmt.Errorf("license type %s: license count %d is below the highest license id %d", typeID, count, maxID)
		}
	}

	adminAddrs := make(map[string]struct{})
	for _, ak := range gs.Permissions {
		if _, exists := adminAddrs[ak.Address]; exists {
			return fmt.Errorf("duplicate permissions address: %s", ak.Address)
		}
		adminAddrs[ak.Address] = struct{}{}

		if _, err := sdk.AccAddressFromBech32(ak.Address); err != nil {
			return fmt.Errorf("permissions entry has invalid address %q: %w", ak.Address, err)
		}

		for i, g := range ak.Grants {
			if !g.Permission.IsValid() {
				return fmt.Errorf("permissions entry %s grant %d: invalid permission %q", ak.Address, i, g.Permission.String())
			}
			if len(g.LicenseTypes) == 0 {
				return fmt.Errorf("permissions entry %s grant %d (permission %q): must include at least one license type", ak.Address, i, g.Permission.Short())
			}
			for _, ltID := range g.LicenseTypes {
				if _, exists := typeIDs[ltID]; !exists {
					return fmt.Errorf("permissions entry %s grant %d (permission %q): unknown license type %q", ak.Address, i, g.Permission.Short(), ltID)
				}
			}
		}
	}

	return nil
}

// validateNonNegativeCounter rejects nil or negative math.Int counters that
// would otherwise panic on later arithmetic.
func validateNonNegativeCounter(v math.Int, name string) error {
	if v.IsNil() {
		return fmt.Errorf("%s must be set", name)
	}
	if v.IsNegative() {
		return fmt.Errorf("%s must not be negative, got %s", name, v.String())
	}
	return nil
}
