package types

const (
	EventTypeCreateNamespace      = "create_namespace"
	EventTypeUpdateNamespaceOwner = "update_namespace_owner"
	EventTypeTransferOwnership    = "transfer_ownership"
	EventTypeGrantPermissions     = "grant_permissions"
	EventTypeRevokePermissions    = "revoke_permissions"

	AttributeKeyModule      = "module"
	AttributeKeyOwner       = "owner"
	AttributeKeyGrantee     = "grantee"
	AttributeKeyPermissions = "permissions"
	AttributeKeyScopes      = "scopes"
)
