package licensesprecompile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestABIMethodsPresent asserts that every method/event the dispatcher references
// is declared in abi.json. If this fires the embedded ABI has drifted out of sync
// with the Go code (or vice versa).
func TestABIMethodsPresent(t *testing.T) {
	wantMethods := []string{
		// transactions
		CreateLicenseTypeMethod,
		UpdateLicenseTypeMethod,
		GrantAdminPermissionsMethod,
		RevokeAdminKeyPermissionsMethod,
		IssueLicensesMethod,
		RevokeLicensesMethod,
		TransferLicenseMethod,
		// queries
		ParamsMethod,
		PermissionsMethod,
		LicenseTypeMethod,
		LicenseTypesMethod,
		LicenseMethod,
		LicensesByTypeMethod,
		LicensesByHolderMethod,
		LicensesByHolderAndTypeMethod,
		AdminKeyMethod,
		AdminKeysMethod,
		AdminKeysByLicenseTypeMethod,
	}
	for _, name := range wantMethods {
		_, ok := ABI.Methods[name]
		require.Truef(t, ok, "method %q missing from ABI", name)
	}

	wantEvents := []string{
		EventTypeLicenseTypeCreated,
		EventTypeLicenseTypeUpdated,
		EventTypeAdminPermissionsGranted,
		EventTypeAdminKeyPermissionsRevoked,
		EventTypeLicenseIssued,
		EventTypeLicenseRevoked,
		EventTypeLicenseTransferred,
	}
	for _, name := range wantEvents {
		_, ok := ABI.Events[name]
		require.Truef(t, ok, "event %q missing from ABI", name)
	}
}

// TestIsTransaction asserts that write methods are classified as transactions
// and read methods are not. This is what gates state-changing calls in readonly mode.
func TestIsTransaction(t *testing.T) {
	p := Precompile{}

	txMethods := []string{
		CreateLicenseTypeMethod, UpdateLicenseTypeMethod,
		GrantAdminPermissionsMethod, RevokeAdminKeyPermissionsMethod,
		IssueLicensesMethod, RevokeLicensesMethod,
		TransferLicenseMethod,
	}
	for _, name := range txMethods {
		m := ABI.Methods[name]
		require.Truef(t, p.IsTransaction(&m), "%s should be a transaction", name)
	}

	queryMethods := []string{
		ParamsMethod, PermissionsMethod,
		LicenseTypeMethod, LicenseTypesMethod,
		LicenseMethod, LicensesByTypeMethod,
		LicensesByHolderMethod, LicensesByHolderAndTypeMethod,
		AdminKeyMethod, AdminKeysMethod, AdminKeysByLicenseTypeMethod,
	}
	for _, name := range queryMethods {
		m := ABI.Methods[name]
		require.Falsef(t, p.IsTransaction(&m), "%s should be a query, not a transaction", name)
	}
}
