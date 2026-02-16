package main

import (
	"fmt"

	"github.com/sgnl-ai/cclog/internal/export"
	"github.com/spf13/cobra"
)

type searchOpts struct {
	project string
	limit   int
}

func searchCmd() *cobra.Command {
	var opts searchOpts

	cmd := &cobra.Command{
		Use:   "search-prompts",
		Short: "Search user prompts across sessions",
		Long:  "Displays all user prompts grouped by session, useful for finding conversation boundaries before multi-session export.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(opts)
		},
	}

	cmd.Flags().StringVar(&opts.project, "project", "", "filter by project path")
	cmd.Flags().IntVar(&opts.limit, "limit", 20, "max sessions to search")

	return cmd
}

func runSearch(opts searchOpts) error {
	result, err := export.SearchPrompts(export.SearchOpts{
		Project: opts.project,
		Limit:   opts.limit,
	})
	if err != nil {
		return err
	}

	fmt.Print(export.FormatPromptSearch(result))
	return nil
}
