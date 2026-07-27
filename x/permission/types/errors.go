package types

import "cosmossdk.io/errors"

var (
	ErrInvalidSigner       = errors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrNamespaceNotFound   = errors.Register(ModuleName, 1101, "namespace owner not set")
	ErrModuleNotRegistered = errors.Register(ModuleName, 1103, "module is not registered with the permission module")
	ErrUnauthorized        = errors.Register(ModuleName, 1104, "signer is not the namespace owner")
	ErrInvalidPermission   = errors.Register(ModuleName, 1105, "invalid permission")
	ErrInvalidScope        = errors.Register(ModuleName, 1106, "invalid scope")
)
