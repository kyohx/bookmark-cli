package main

import (
	"context"
	"errors"
	"time"

	"bookmark-cli/internal/cli"
	"bookmark-cli/internal/session"
	"github.com/spf13/cobra"
)

type authStatus struct {
	LoggedIn  bool   `json:"logged_in"`
	TokenType string `json:"token_type,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
	Expired   bool   `json:"expired,omitempty"`
}

func newAuthCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Auth-related commands",
	}
	cmd.AddCommand(newAuthStatusCmd(opts))
	return cmd
}

func newAuthStatusCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show local auth session status",
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := loadDeps(opts, false)
			if err != nil {
				return err
			}
			token, err := deps.Auth.CurrentToken(context.Background())
			if err != nil {
				if errors.Is(err, session.ErrNotFound) {
					return cli.PrintJSON(authStatus{LoggedIn: false})
				}
				return err
			}

			st := authStatus{
				LoggedIn:  true,
				TokenType: token.TokenType,
			}
			if !token.Expiry.IsZero() {
				st.ExpiresAt = token.Expiry.UTC().Format(time.RFC3339)
				st.Expired = token.Expiry.Before(time.Now().UTC())
			}
			return cli.PrintJSON(st)
		},
	}
}
