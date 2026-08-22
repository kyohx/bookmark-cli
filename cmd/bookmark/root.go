package main

import (
	"fmt"
	"os"

	"bookmark-cli/internal/config"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	ConfigPath  string
	Profile     string
	BaseURL     string
	TimeoutSec  int
	TimeoutSet  bool
	ShowProfile bool
}

func newRootCmd() *cobra.Command {
	opts := &rootOptions{}

	cmd := &cobra.Command{
		Use:     "bookmark",
		Short:   "CLI for bookmark-sample WebAPI",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			opts.TimeoutSet = cmd.Flags().Changed("timeout")
			return maybePrintProfile(cmd, opts)
		},
	}
	cmd.SilenceUsage = true
	cmd.SetVersionTemplate("{{.Version}}\n")

	defaultConfigPath, err := config.DefaultPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to resolve default config path:", err)
		defaultConfigPath = ""
	}

	cmd.PersistentFlags().StringVar(&opts.ConfigPath, "config", defaultConfigPath, "config file path")
	cmd.PersistentFlags().StringVar(&opts.Profile, "profile", config.DefaultProfile, "profile name in config file")
	cmd.PersistentFlags().StringVar(&opts.BaseURL, "base-url", "", "override API base URL")
	cmd.PersistentFlags().IntVar(&opts.TimeoutSec, "timeout", 10, "HTTP timeout seconds")
	cmd.PersistentFlags().BoolVar(&opts.ShowProfile, "show-profile", false, "print resolved profile to stderr before running the command")

	cmd.AddCommand(newLoginCmd(opts))
	cmd.AddCommand(newLogoutCmd(opts))
	cmd.AddCommand(newWhoamiCmd(opts))
	cmd.AddCommand(newAuthCmd(opts))
	cmd.AddCommand(newBookmarkListCmd(opts))
	cmd.AddCommand(newBookmarkAddCmd(opts))
	cmd.AddCommand(newBookmarkUpdateCmd(opts))
	cmd.AddCommand(newBookmarkDeleteCmd(opts))
	cmd.AddCommand(newAPIVersionCmd(opts))
	cmd.AddCommand(newVersionCmd())
	return cmd
}
