package main

import (
	"context"

	"bookmark-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newLogoutCmd(opts rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Delete session from OS keychain",
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := loadDeps(opts, false)
			if err != nil {
				return err
			}
			if err := deps.Auth.Logout(context.Background()); err != nil {
				return err
			}
			cli.PrintLine("Logged out.")
			return nil
		},
	}
}
