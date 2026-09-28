package main

import (
	"context"
	"fmt"

	"bookmark-cli/internal/api"
	"bookmark-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newUserCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "user", Short: "User operations"}
	cmd.AddCommand(newUserAddCmd(opts), newUserGetCmd(opts), newUserListCmd(opts), newUserUpdateCmd(opts))
	return cmd
}

func newUserAddCmd(opts *rootOptions) *cobra.Command {
	var authority int
	var passwordStdin bool
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("authority") {
				return fmt.Errorf("--authority is required")
			}
			if err := api.ValidateAddUserOptions(args[0], authority); err != nil {
				return err
			}
			password, err := readPassword(passwordStdin)
			if err != nil {
				return err
			}
			defer zeroBytes(password)
			deps, err := loadDeps(opts, true)
			if err != nil {
				return err
			}
			user, err := deps.API.AddUser(context.Background(), api.AddUserRequest{
				Name: args[0], Password: string(password), Authority: authority,
			})
			if err != nil {
				return err
			}
			return cli.PrintJSON(user)
		},
	}
	cmd.Flags().IntVar(&authority, "authority", 0, "user authority (0, 1, 2, or 9)")
	cmd.Flags().BoolVar(&passwordStdin, "password-stdin", false, "read password from stdin")
	return cmd
}

func newUserGetCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <name>",
		Short: "Get a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := loadDeps(opts, true)
			if err != nil {
				return err
			}
			user, err := deps.API.GetUser(context.Background(), args[0])
			if err != nil {
				return err
			}
			return cli.PrintJSON(user)
		},
	}
}

func newUserListCmd(opts *rootOptions) *cobra.Command {
	var page, size int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List users",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := loadDeps(opts, true)
			if err != nil {
				return err
			}
			users, err := deps.API.ListUsers(context.Background(), page, size)
			if err != nil {
				return err
			}
			return cli.PrintJSON(users)
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "page number")
	cmd.Flags().IntVar(&size, "size", 10, "page size")
	return cmd
}

func newUserUpdateCmd(opts *rootOptions) *cobra.Command {
	var newName string
	var authority int
	var disabled bool
	var password, passwordStdin bool
	cmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Update a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if password && passwordStdin {
				return fmt.Errorf("--password and --password-stdin cannot be used together")
			}
			var req api.UpdateUserRequest
			if cmd.Flags().Changed("name") {
				req.Name = &newName
			}
			if cmd.Flags().Changed("authority") {
				req.Authority = &authority
			}
			if cmd.Flags().Changed("disabled") {
				req.Disabled = &disabled
			}
			if req.Name == nil && req.Authority == nil && req.Disabled == nil && !password && !passwordStdin {
				return fmt.Errorf("specify at least one field to update")
			}
			if password || passwordStdin {
				pw, err := readPassword(passwordStdin)
				if err != nil {
					return err
				}
				defer zeroBytes(pw)
				value := string(pw)
				req.Password = &value
			}
			deps, err := loadDeps(opts, true)
			if err != nil {
				return err
			}
			user, err := deps.API.UpdateUser(context.Background(), args[0], req)
			if err != nil {
				return err
			}
			return cli.PrintJSON(user)
		},
	}
	cmd.Flags().StringVar(&newName, "name", "", "new user name")
	cmd.Flags().IntVar(&authority, "authority", 0, "new authority (0, 1, 2, or 9)")
	cmd.Flags().BoolVar(&disabled, "disabled", false, "set disabled state (true or false)")
	cmd.Flags().BoolVar(&password, "password", false, "prompt for new password")
	cmd.Flags().BoolVar(&passwordStdin, "password-stdin", false, "read new password from stdin")
	return cmd
}
