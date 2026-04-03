package config

import (
	evmconfig "github.com/cosmos/evm/config"

	clienthelpers "cosmossdk.io/client/v2/helpers"
)

const (
	// AppName is the name of the application binary.
	AppName = "webstackd"

	// DefaultNodeHomeDir is the default home directory name.
	DefaultNodeHomeDir = ".webstackd"
)

// MustGetDefaultNodeHome returns the default node home directory.
func MustGetDefaultNodeHome() string {
	defaultNodeHome, err := clienthelpers.GetNodeHomeDirectory(DefaultNodeHomeDir)
	if err != nil {
		panic(err)
	}
	return defaultNodeHome
}

// SetBech32Prefixes wraps the cosmos evm config SetBech32Prefixes.
var SetBech32Prefixes = evmconfig.SetBech32Prefixes

// SetBip44CoinType wraps the cosmos evm config SetBip44CoinType.
var SetBip44CoinType = evmconfig.SetBip44CoinType

// InitAppConfig wraps the cosmos evm config InitAppConfig.
var InitAppConfig = evmconfig.InitAppConfig

// GetChainIDFromHome wraps the cosmos evm config GetChainIDFromHome.
var GetChainIDFromHome = evmconfig.GetChainIDFromHome
