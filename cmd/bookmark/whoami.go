package main

import (
	"context"

	"bookmark-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newWhoamiCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show current logged-in user",
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := loadDeps(opts, true)
			if err != nil {
				return err
			}
			user, err := deps.API.Me(context.Background())
			if err != nil {
				return err
			}
			return cli.PrintJSON(user)
		},
	}
}
