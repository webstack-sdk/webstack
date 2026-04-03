package cli

import (
	"fmt"
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

	cmd.AddCommand(CmdSetAdminKey())
	cmd.AddCommand(CmdBatchIssueLicense())

	return cmd
}

// CmdSetAdminKey returns a command to set admin key grants for an address.
//
// Usage:
//
//	set-admin-key [address] [permissions] [license-types]
//
// Where [permissions] is a comma-delimited list (e.g. "issue,revoke") and
// [license-types] is a comma-delimited list of license type IDs.
// One AdminKeyGrant is created per permission, each sharing the same list of license types.
// The owner is taken from --from.
func CmdSetAdminKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-admin-key [address] [permissions] [license-types]",
		Short: "Set admin key grants for an address",
		Long: `Set admin key grants for a given address. The module owner (--from) must sign.

[permissions]    Comma-delimited list of permissions to grant. Valid values: issue, revoke, update.
[license-types]  Comma-delimited list of license type IDs these permissions apply to.

One grant is created per permission, each covering all specified license types.

Example:
  webstackd tx licenses set-admin-key cosmos1abc... issue,revoke node.license,validator.license \
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
				grants = append(grants, types.AdminKeyGrant{
					Permission:   perm,
					LicenseTypes: licenseTypes,
				})
			}

			if len(grants) == 0 {
				return fmt.Errorf("at least one permission must be specified")
			}

			msg := &types.MsgSetAdminKey{
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

// CmdBatchIssueLicense returns a command to issue licenses to multiple holders in one tx.
//
// Usage:
//
//	batch-issue-license [license-type-id] [holder1:start:end] [holder2:start:end] ...
//
// Each entry is colon-delimited: holder_address:start_date:end_date
// The end_date is optional (omit or leave empty after the second colon).
// The issuer is taken from --from.
func CmdBatchIssueLicense() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch-issue-license [license-type-id] [entries...]",
		Short: "Issue licenses to multiple holders in a single transaction",
		Long: `Issue licenses to multiple holders in a single transaction. The issuer (--from) must have "issue" permission.

Each entry after the license type ID is colon-delimited:
  holder_address:start_date:end_date

The end_date is optional. If omitted, the license has no expiry.

Example:
  webstackd tx licenses batch-issue-license node.license \
    cosmos1abc...:2025-01-01:2026-01-01 \
    cosmos1def...:2025-01-01 \
    --from admin --gas auto --fees 100000aatom -y`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			licenseTypeID := args[0]
			entries := make([]types.BatchIssueLicenseEntry, 0, len(args)-1)

			for i, arg := range args[1:] {
				parts := strings.SplitN(arg, ":", 3)
				if len(parts) < 2 {
					return fmt.Errorf("entry %d: expected format holder:start_date[:end_date], got %q", i, arg)
				}

				holder := strings.TrimSpace(parts[0])
				if _, err := sdk.AccAddressFromBech32(holder); err != nil {
					return fmt.Errorf("entry %d: invalid holder address %q: %w", i, holder, err)
				}

				startDate := strings.TrimSpace(parts[1])
				var endDate string
				if len(parts) == 3 {
					endDate = strings.TrimSpace(parts[2])
				}

				entries = append(entries, types.BatchIssueLicenseEntry{
					Holder:    holder,
					StartDate: startDate,
					EndDate:   endDate,
				})
			}

			msg := &types.MsgBatchIssueLicense{
				Issuer:        clientCtx.GetFromAddress().String(),
				LicenseTypeId: licenseTypeID,
				Entries:       entries,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
