package main

import (
	"context"

	"bookmark-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newAPIVersionCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "api-version",
		Short: "Show the API version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			deps, err := loadDeps(opts, true)
			if err != nil {
				return err
			}
			version, err := deps.API.Version(context.Background())
			if err != nil {
				return err
			}
			return cli.PrintJSON(version)
		},
	}
}
