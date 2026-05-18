package licensesprecompile

const (
	// ErrCallerIsNotIssuer is returned when an issueLicense call originates from
	// an address that is not the issuer declared in the message.
	ErrCallerIsNotIssuer = "caller %s is not the declared issuer %s"
	// ErrCallerIsNotRevoker is returned when a revokeLicense call originates from
	// an address that is not the revoker declared in the message.
	ErrCallerIsNotRevoker = "caller %s is not the declared revoker %s"
	// ErrCallerIsNotHolder is returned when a transferLicense call originates from
	// an address that is not the current license holder.
	ErrCallerIsNotHolder = "caller %s is not the current holder %s"
	// ErrCallerIsNotOwner is returned when an owner-gated call originates from a
	// non-owner address.
	ErrCallerIsNotOwner = "caller %s is not the module owner %s"
)
