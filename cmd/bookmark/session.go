package main

import (
	"fmt"

	"bookmark-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newSessionCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "session", Short: "User refresh session operations (admin only)"}
	cmd.AddCommand(newSessionListCmd(opts), newSessionRevokeCmd(opts))
	return cmd
}

func newSessionListCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list <name>",
		Short: "List a user's refresh sessions (admin only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := loadDeps(opts, true)
			if err != nil {
				return err
			}
			sessions, err := deps.API.ListUserSessions(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return cli.PrintJSONTo(cmd.OutOrStdout(), sessions)
		},
	}
}

func newSessionRevokeCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "revoke <name> <session_id>",
		Short: "Revoke a user's refresh session (admin only)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := loadDeps(opts, true)
			if err != nil {
				return err
			}
			if err := deps.API.RevokeUserSession(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Revoked.")
			return err
		},
	}
}
