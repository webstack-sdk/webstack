package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"github.com/webstack-sdk/webstack/x/licenses/types"
)

// GetTxCmd returns the transaction commands for this module.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdGrantAdminPermissions())
	cmd.AddCommand(CmdRevokeAdminKeyPermissions())
	cmd.AddCommand(CmdIssueLicenses())
	cmd.AddCommand(CmdRevokeLicenses())

	return cmd
}

// CmdGrantAdminPermissions returns a command to grant admin key permissions for an address.
//
// Usage:
//
//	grant-admin-permissions [address] [permissions] [license-types]
//
// Where [permissions] is a comma-delimited list (e.g. "issue,revoke") and
// [license-types] is a comma-delimited list of license type IDs.
// One AdminKeyGrant is created per permission, each sharing the same list of license types.
// The owner is taken from --from.
func CmdGrantAdminPermissions() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grant-admin-permissions [address] [permissions] [license-types]",
		Short: "Grant admin key permissions for an address",
		Long: `Grant admin key permissions for a given address. The module owner (--from) must sign.

[permissions]    Comma-delimited list of permissions to grant. Valid values: issue, revoke.
[license-types]  Comma-delimited list of license type IDs these permissions apply to.

One grant is created per permission, each covering all specified license types.

Grants are MERGED with any existing grants for the address — previously
granted permissions and license types are preserved. To remove specific
(license-type, permission) pairs, use revoke-admin-key-permissions.

Example:
  webstackd tx licenses grant-admin-permissions webstack1abc... issue,revoke node.license,validator.license \
    --from owner --gas auto --gas-adjustment 1.5 --fees 100000aatom -y`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			address := args[0]
			if _, err := sdk.AccAddressFromBech32(address); err != nil {
				return fmt.Errorf("invalid address %q: %w", address, err)
			}

			permissions := strings.Split(args[1], ",")
			licenseTypes := strings.Split(args[2], ",")

			for i, p := range permissions {
				permissions[i] = strings.TrimSpace(p)
			}
			for i, lt := range licenseTypes {
				licenseTypes[i] = strings.TrimSpace(lt)
			}

			grants := make([]types.AdminKeyGrant, 0, len(permissions))
			for _, perm := range permissions {
				if perm == "" {
					continue
				}
				if !types.IsValidPermission(perm) {
					return fmt.Errorf("invalid permission %q: must be one of %s", perm, strings.Join(types.Permissions, ", "))
				}
				grants = append(grants, types.AdminKeyGrant{
					Permission:   perm,
					LicenseTypes: licenseTypes,
				})
			}

			if len(grants) == 0 {
				return fmt.Errorf("at least one permission must be specified")
			}

			msg := &types.MsgGrantAdminPermissions{
				Owner:   clientCtx.GetFromAddress().String(),
				Address: address,
				Grants:  grants,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdRevokeAdminKeyPermissions returns a command to remove specific
// (license-type, permission) pairs from an admin key.
//
// Usage:
//
//	revoke-admin-key-permissions [address] [pair1] [pair2] ...
//
// Each pair is colon-delimited: license-type-id:permission-name.
// Pairs that aren't currently granted are silently ignored. If the resulting
// admin key has no remaining grants, the entry is deleted entirely.
// The owner is taken from --from.
func CmdRevokeAdminKeyPermissions() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke-admin-key-permissions [address] [license-type:permission ...]",
		Short: "Revoke specific (license-type, permission) pairs from an admin key",
		Long: `Revoke specific (license-type, permission) pairs from an admin key.
The module owner (--from) must sign.

Each pair after the address is colon-delimited:
  license-type-id:permission-name

Valid permissions: issue, revoke.

Pairs that aren't currently granted are silently ignored. A grant whose
license types become empty is dropped; if no grants remain, the entire
admin key entry is deleted.

Example:
  webstackd tx licenses revoke-admin-key-permissions webstack1abc... \
    node.license:issue validator.license:revoke \
    --from owner --gas auto --fees 100000aatom -y`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			address := args[0]
			if _, err := sdk.AccAddressFromBech32(address); err != nil {
				return fmt.Errorf("invalid address %q: %w", address, err)
			}

			permissions := make([]types.AdminKeyPermission, 0, len(args)-1)
			for i, arg := range args[1:] {
				parts := strings.SplitN(arg, ":", 2)
				if len(parts) != 2 {
					return fmt.Errorf("pair %d: expected format license-type:permission, got %q", i, arg)
				}
				lt := strings.TrimSpace(parts[0])
				perm := strings.TrimSpace(parts[1])
				if lt == "" || perm == "" {
					return fmt.Errorf("pair %d: license-type and permission must both be non-empty (got %q)", i, arg)
				}
				permissions = append(permissions, types.AdminKeyPermission{
					LicenseTypeId: lt,
					Permission:    perm,
				})
			}

			msg := &types.MsgRevokeAdminKeyPermissions{
				Owner:       clientCtx.GetFromAddress().String(),
				Address:     address,
				Permissions: permissions,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdIssueLicenses returns a command to issue licenses to one or more holders,
// across one or more license types, in a single transaction.
//
// Usage:
//
//	issue-licenses [entry1] [entry2] ...
//
// Each entry is colon-delimited: license_type_id:holder:count:start_date[:end_date]
// The end_date is optional (omit or leave empty after the fourth colon).
// The issuer is taken from --from.
func CmdIssueLicenses() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue-licenses [entries...]",
		Short: "Issue licenses to one or more holders in a single transaction",
		Long: `Issue licenses in a single transaction. Each entry can target a different
license type and holder. The issuer (--from) must have "issue" permission for
every referenced license type.

Each entry is colon-delimited:
  license_type_id:holder:count:start_date[:end_date]

The end_date is optional. If omitted, the license has no expiry.

Example:
  webstackd tx licenses issue-licenses \
    node.license:webstack1abc...:1:2025-01-01:2026-01-01 \
    validator.license:webstack1def...:3:2025-01-01 \
    --from admin --gas auto --fees 100000aatom -y`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			entries := make([]types.IssueLicenseEntry, 0, len(args))
			for i, arg := range args {
				parts := strings.SplitN(arg, ":", 5)
				if len(parts) < 4 {
					return fmt.Errorf("entry %d: expected format license_type_id:holder:count:start_date[:end_date], got %q", i, arg)
				}

				licenseTypeID := strings.TrimSpace(parts[0])
				if licenseTypeID == "" {
					return fmt.Errorf("entry %d: license type id must not be empty", i)
				}

				holder := strings.TrimSpace(parts[1])
				if _, err := sdk.AccAddressFromBech32(holder); err != nil {
					return fmt.Errorf("entry %d: invalid holder address %q: %w", i, holder, err)
				}

				count, err := strconv.ParseUint(strings.TrimSpace(parts[2]), 10, 64)
				if err != nil {
					return fmt.Errorf("entry %d: invalid count %q: %w", i, parts[2], err)
				}

				startDate := strings.TrimSpace(parts[3])
				var endDate string
				if len(parts) == 5 {
					endDate = strings.TrimSpace(parts[4])
				}

				entries = append(entries, types.IssueLicenseEntry{
					LicenseTypeId: licenseTypeID,
					Holder:        holder,
					Count:         count,
					StartDate:     startDate,
					EndDate:       endDate,
				})
			}

			msg := &types.MsgIssueLicenses{
				Issuer:  clientCtx.GetFromAddress().String(),
				Entries: entries,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdRevokeLicenses returns a command to revoke licenses for a holder.
//
// Usage:
//
//	revoke-licenses [license-type-id] [holder] [count]
//
// The most recently issued active licenses are revoked first.
// The revoker is taken from --from.
func CmdRevokeLicenses() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke-licenses [license-type-id] [holder] [count]",
		Short: "Revoke licenses for a holder, most recent first",
		Long: `Revoke active licenses for a holder. The revoker (--from) must have "revoke" permission.

The most recently issued active licenses are revoked first. Their status is set to "revoked"
and end_date is set to the current block date.

Example:
  webstackd tx licenses revoke-licenses node.license webstack1abc... 2 \
    --from admin --gas auto --fees 100000aatom -y`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			holder := args[1]
			if _, err := sdk.AccAddressFromBech32(holder); err != nil {
				return fmt.Errorf("invalid holder address %q: %w", holder, err)
			}

			count, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid count %q: %w", args[2], err)
			}

			msg := &types.MsgRevokeLicenses{
				Revoker:       clientCtx.GetFromAddress().String(),
				LicenseTypeId: args[0],
				Holder:        holder,
				Count:         count,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
