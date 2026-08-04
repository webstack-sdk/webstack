package types

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultParams returns the module's default parameters. Defaults are
// chain-neutral: license_types is empty, which fail-closes activation (no
// counted licenses means a limit of zero), and deauthorize_fee is empty
// (charge disabled) because this module cannot name a chain's denom.
// Consuming chains seed their real values in an upgrade handler or genesis.
func DefaultParams() Params {
	return Params{
		LicenseTypes:              []string{},
		ActivationLimitMultiplier: 3,
		SpamLimitMultiplier:       9,
		MaxActivationKeys:         5,
		RecentKeyLimit:            10,
		RecentKeyWindow:           24 * time.Hour,
		StatusDailyLimit:          10,
		DeauthorizeFee:            sdk.NewCoins(),
		MaxGaslessGas:             300_000,
		MaxGaslessMsgs:            10,
		MaxGaslessTxBytes:         50_000,
	}
}

// Validate checks the parameter set is well-formed.
func (p Params) Validate() error {
	if err := validateTypeList("license_types", p.LicenseTypes); err != nil {
		return err
	}
	if p.ActivationLimitMultiplier == 0 {
		return fmt.Errorf("activation_limit_multiplier must be positive")
	}
	if p.SpamLimitMultiplier < p.ActivationLimitMultiplier {
		return fmt.Errorf("spam_limit_multiplier (%d) must not be below activation_limit_multiplier (%d)", p.SpamLimitMultiplier, p.ActivationLimitMultiplier)
	}
	if p.MaxActivationKeys == 0 {
		return fmt.Errorf("max_activation_keys must be positive")
	}
	if p.RecentKeyLimit == 0 {
		return fmt.Errorf("recent_key_limit must be positive")
	}
	if p.RecentKeyWindow <= 0 {
		return fmt.Errorf("recent_key_window must be positive, got %s", p.RecentKeyWindow)
	}
	if p.StatusDailyLimit == 0 {
		return fmt.Errorf("status_daily_limit must be positive")
	}
	if err := p.DeauthorizeFee.Validate(); err != nil {
		return fmt.Errorf("invalid deauthorize_fee: %w", err)
	}
	if p.MaxGaslessGas == 0 {
		return fmt.Errorf("max_gasless_gas must be positive")
	}
	if p.MaxGaslessMsgs == 0 {
		return fmt.Errorf("max_gasless_msgs must be positive")
	}
	if p.MaxGaslessTxBytes == 0 {
		return fmt.Errorf("max_gasless_tx_bytes must be positive")
	}
	return nil
}

func validateTypeList(name string, list []string) error {
	seen := make(map[string]struct{}, len(list))
	for _, entry := range list {
		if entry == "" {
			return fmt.Errorf("%s must not contain empty entries", name)
		}
		if _, dup := seen[entry]; dup {
			return fmt.Errorf("%s contains duplicate entry %q", name, entry)
		}
		seen[entry] = struct{}{}
	}
	return nil
}
