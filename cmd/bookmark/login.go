package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"bookmark-cli/internal/cli"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newLoginCmd(opts rootOptions) *cobra.Command {
	var username string
	var passwordStdin bool
	var scopes []string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login and store session in OS keychain",
		RunE: func(cmd *cobra.Command, _ []string) error {
			deps, err := loadDeps(opts, false)
			if err != nil {
				return err
			}

			if strings.TrimSpace(username) == "" {
				cli.PrintLine("Username:")
				r := bufio.NewReader(os.Stdin)
				line, err := r.ReadString('\n')
				if err != nil && !errors.Is(err, io.EOF) {
					return cli.WrapUserError("read username", err)
				}
				username = strings.TrimSpace(line)
			}
			if username == "" {
				return fmt.Errorf("username is required")
			}

			password, err := readPassword(passwordStdin)
			if err != nil {
				return err
			}
			defer zeroBytes(password)

			effectiveScopes := scopes
			if len(effectiveScopes) == 0 {
				effectiveScopes = deps.Scopes
			}

			if err := deps.Auth.Login(context.Background(), username, password, effectiveScopes); err != nil {
				return err
			}
			cli.PrintLine("Logged in successfully.")
			return nil
		},
	}

	cmd.Flags().StringVar(&username, "username", "", "username for login")
	cmd.Flags().BoolVar(&passwordStdin, "password-stdin", false, "read password from stdin")
	cmd.Flags().StringArrayVar(&scopes, "scope", nil, "requested OAuth scope (repeatable)")
	return cmd
}

func readPassword(passwordStdin bool) ([]byte, error) {
	if passwordStdin {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, cli.WrapUserError("read password from stdin", err)
		}
		return []byte(strings.TrimRight(string(b), "\r\n")), nil
	}

	fmt.Fprint(os.Stdout, "Password: ")
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stdout)
	if err != nil {
		return nil, cli.WrapUserError("read password", err)
	}
	return pw, nil
}

func zeroBytes(v []byte) {
	for i := range v {
		v[i] = 0
	}
}
