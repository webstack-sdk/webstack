package types

// PrecompileAddress is the default EVM address at which the licenses precompile
// is registered. It lives in the application-reserved range so it cannot collide
// with the upstream cosmos/evm static precompiles (0x...0100, 0x...0400, 0x...08xx).
//
// Operators that want to register the precompile at a different address can do so
// when wiring it into the EVM keeper; this value is only consulted by the default
// registration helper.
const PrecompileAddress = "0x0000000000000000000000000000000000001900"
