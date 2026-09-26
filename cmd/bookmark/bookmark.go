package main

import (
	"context"
	"fmt"
	"strings"

	"bookmark-cli/internal/api"
	"bookmark-cli/internal/cli"
	"github.com/spf13/cobra"
)

func newBookmarkCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bookmark",
		Short: "Bookmark operations",
	}
	cmd.AddCommand(newBookmarkListCmd(opts))
	cmd.AddCommand(newBookmarkAddCmd(opts))
	cmd.AddCommand(newBookmarkGetCmd(opts))
	cmd.AddCommand(newBookmarkUpdateCmd(opts))
	cmd.AddCommand(newBookmarkDeleteCmd(opts))
	return cmd
}

func newBookmarkListCmd(opts *rootOptions) *cobra.Command {
	var tags []string
	var page int
	var size int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List bookmarks",
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := loadDeps(opts, true)
			if err != nil {
				return err
			}
			bookmarks, err := deps.API.ListBookmarks(context.Background(), tags, page, size)
			if err != nil {
				return err
			}
			return cli.PrintJSON(bookmarks)
		},
	}
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "filter by tag (repeatable)")
	cmd.Flags().IntVar(&page, "page", 1, "page number")
	cmd.Flags().IntVar(&size, "size", 10, "page size")
	return cmd
}

func newBookmarkAddCmd(opts *rootOptions) *cobra.Command {
	var url string
	var memo string
	var tags []string

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a bookmark",
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := loadDeps(opts, true)
			if err != nil {
				return err
			}
			if strings.TrimSpace(url) == "" {
				return fmt.Errorf("url is required")
			}
			if len(tags) == 0 {
				return fmt.Errorf("at least one tag is required")
			}

			bookmark, err := deps.API.AddBookmark(context.Background(), api.AddBookmarkRequest{
				URL:  url,
				Memo: memo,
				Tags: tags,
			})
			if err != nil {
				return err
			}
			return cli.PrintJSON(bookmark)
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "bookmark URL")
	cmd.Flags().StringVar(&memo, "memo", "", "bookmark memo")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "bookmark tag (repeatable)")
	return cmd
}

func newBookmarkUpdateCmd(opts *rootOptions) *cobra.Command {
	var memo string
	var tags []string

	cmd := &cobra.Command{
		Use:   "update <hashed_id>",
		Short: "Update a bookmark",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := loadDeps(opts, true)
			if err != nil {
				return err
			}

			var memoPtr *string
			if cmd.Flags().Changed("memo") {
				m := memo
				memoPtr = &m
			}

			hasTags := cmd.Flags().Changed("tag")
			if hasTags && len(tags) == 0 {
				return fmt.Errorf("at least one tag is required when --tag is specified")
			}

			if memoPtr == nil && !hasTags {
				return fmt.Errorf("specify at least one field to update: --memo or --tag")
			}

			bookmark, err := deps.API.UpdateBookmark(context.Background(), args[0], memoPtr, tags, hasTags)
			if err != nil {
				return err
			}
			return cli.PrintJSON(bookmark)
		},
	}

	cmd.Flags().StringVar(&memo, "memo", "", "new memo")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "new tag list (repeatable)")
	return cmd
}

func newBookmarkDeleteCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <hashed_id>",
		Short: "Delete a bookmark",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := loadDeps(opts, true)
			if err != nil {
				return err
			}
			if err := deps.API.DeleteBookmark(context.Background(), args[0]); err != nil {
				return err
			}
			cli.PrintLine("Deleted.")
			return nil
		},
	}
	return cmd
}

func newBookmarkGetCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use: "get <hashed_id>", Short: "Get a bookmark", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, err := loadDeps(opts, true)
			if err != nil {
				return err
			}
			bookmark, err := deps.API.GetBookmark(context.Background(), args[0])
			if err != nil {
				return err
			}
			return cli.PrintJSON(bookmark)
		},
	}
}
