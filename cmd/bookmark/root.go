package main

import (
	"fmt"
	"os"

	"bookmark-cli/internal/config"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	ConfigPath string
	BaseURL    string
	TimeoutSec int
}

func newRootCmd() *cobra.Command {
	opts := rootOptions{}

	cmd := &cobra.Command{
		Use:     "bookmark",
		Short:   "CLI for bookmark-sample WebAPI",
		Version: version,
	}
	cmd.SilenceUsage = true
	cmd.SetVersionTemplate("{{.Version}}\n")

	defaultConfigPath, err := config.DefaultPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to resolve default config path:", err)
		defaultConfigPath = ""
	}

	cmd.PersistentFlags().StringVar(&opts.ConfigPath, "config", defaultConfigPath, "config file path")
	cmd.PersistentFlags().StringVar(&opts.BaseURL, "base-url", "", "override API base URL")
	cmd.PersistentFlags().IntVar(&opts.TimeoutSec, "timeout", 10, "HTTP timeout seconds")

	cmd.AddCommand(newLoginCmd(opts))
	cmd.AddCommand(newLogoutCmd(opts))
	cmd.AddCommand(newWhoamiCmd(opts))
	cmd.AddCommand(newAuthCmd(opts))
	cmd.AddCommand(newBookmarkListCmd(opts))
	cmd.AddCommand(newBookmarkAddCmd(opts))
	cmd.AddCommand(newBookmarkUpdateCmd(opts))
	cmd.AddCommand(newBookmarkDeleteCmd(opts))
	cmd.AddCommand(newVersionCmd())
	return cmd
}
