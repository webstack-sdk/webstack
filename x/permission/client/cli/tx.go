package cli

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"github.com/webstack-sdk/webstack/x/permission/types"
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

	cmd.AddCommand(CmdGrantPermissions())
	cmd.AddCommand(CmdRevokePermissions())

	return cmd
}

// CmdGrantPermissions returns a command to grant permissions to an address
// within a namespace.
//
// Usage:
//
//	grant-permissions [module] [grantee] [permissions] [scopes]
//
// Where [permissions] is a comma-delimited list (e.g. "issue,revoke") and
// [scopes] is a comma-delimited list of scope identifiers, or "-" for a
// module-wide grant in namespaces that don't scope their permissions.
// One grant entry is created per permission, each sharing the same scopes.
// The namespace owner is taken from --from.
func CmdGrantPermissions() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grant-permissions [module] [grantee] [permissions] [scopes]",
		Short: "Grant permissions to an address within a namespace",
		Long: `Grant permissions to a given address within a module's namespace. The
namespace owner (--from) must sign.

[module]       The consuming module's name (e.g. "license").
[grantee]      The address receiving the grants.
[permissions]  Comma-delimited list of permissions to grant. Valid values are
               whatever the module registered; query them with:
               webstackd query permission module [module]
[scopes]       Comma-delimited list of scope identifiers these permissions
               apply to (e.g. license type IDs), or "-" for a module-wide
               grant in namespaces that don't scope their permissions.

One grant entry is created per permission, each covering all specified scopes.

Grants are MERGED with any existing grants for the grantee — previously
granted permissions and scopes are preserved. To remove specific
(permission, scope) pairs, use revoke-permissions.

Example:
  webstackd tx permission grant-permissions license webstack1abc... issue,revoke node.license,validator.license \
    --from owner --gas auto --gas-adjustment 1.5 --fees 100000aatom -y`,
		Args: cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			module := strings.TrimSpace(args[0])

			grantee := args[1]
			if _, err := sdk.AccAddressFromBech32(grantee); err != nil {
				return fmt.Errorf("invalid grantee address %q: %w", grantee, err)
			}

			permissions := strings.Split(args[2], ",")
			for i, p := range permissions {
				permissions[i] = strings.TrimSpace(p)
			}

			// "-" requests a module-wide grant: an empty scope list.
			var scopes []string
			if strings.TrimSpace(args[3]) != "-" {
				scopes = strings.Split(args[3], ",")
				for i, s := range scopes {
					scopes[i] = strings.TrimSpace(s)
				}
			}

			grants := make([]types.PermissionScopes, 0, len(permissions))
			for _, perm := range permissions {
				if perm == "" {
					continue
				}
				grants = append(grants, types.PermissionScopes{
					Permission: perm,
					Scopes:     scopes,
				})
			}

			if len(grants) == 0 {
				return fmt.Errorf("at least one permission must be specified")
			}

			msg := &types.MsgGrantPermissions{
				Owner:   clientCtx.GetFromAddress().String(),
				Module:  module,
				Grantee: grantee,
				Grants:  grants,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdRevokePermissions returns a command to remove specific
// (permission, scope) pairs from a grantee within a namespace.
//
// Usage:
//
//	revoke-permissions [module] [grantee] [pair1] [pair2] ...
//
// Each pair is colon-delimited: permission:scope. A trailing colon (or a bare
// permission with no colon) targets the module-wide (empty scope) grant.
// Pairs that aren't currently granted are silently ignored.
// The namespace owner is taken from --from.
func CmdRevokePermissions() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke-permissions [module] [grantee] [permission:scope ...]",
		Short: "Revoke specific (permission, scope) pairs from a grantee",
		Long: `Revoke specific (permission, scope) pairs from an address within a module's
namespace. The namespace owner (--from) must sign.

Each pair after the grantee is colon-delimited:
  permission:scope

A bare permission with no colon targets the module-wide (empty scope) grant.

Pairs that aren't currently granted are silently ignored.

Example:
  webstackd tx permission revoke-permissions license webstack1abc... \
    issue:node.license revoke:validator.license \
    --from owner --gas auto --fees 100000aatom -y`,
		Args: cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			module := strings.TrimSpace(args[0])

			grantee := args[1]
			if _, err := sdk.AccAddressFromBech32(grantee); err != nil {
				return fmt.Errorf("invalid grantee address %q: %w", grantee, err)
			}

			permissions := make([]types.PermissionScope, 0, len(args)-2)
			for i, arg := range args[2:] {
				parts := strings.SplitN(arg, ":", 2)
				perm := strings.TrimSpace(parts[0])
				if perm == "" {
					return fmt.Errorf("pair %d: permission must be non-empty (got %q)", i, arg)
				}
				var scope string
				if len(parts) == 2 {
					scope = strings.TrimSpace(parts[1])
				}
				permissions = append(permissions, types.PermissionScope{
					Permission: perm,
					Scope:      scope,
				})
			}

			msg := &types.MsgRevokePermissions{
				Owner:       clientCtx.GetFromAddress().String(),
				Module:      module,
				Grantee:     grantee,
				Permissions: permissions,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
