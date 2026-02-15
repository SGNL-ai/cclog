package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/sgnl-ai/cclog/internal/export"
	"github.com/sgnl-ai/cclog/internal/session"
	"github.com/spf13/cobra"
)

type listOpts struct {
	project string
}

func listCmd() *cobra.Command {
	var opts listOpts

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(opts)
		},
	}

	cmd.Flags().StringVar(&opts.project, "project", "", "filter by project path")

	return cmd
}

func runList(opts listOpts) error {
	sessions, err := export.ListSessions("", opts.project)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tDATE\tMSGS\tPROJECT\tPROMPT")

	for _, s := range sessions {
		prompt := s.FirstPrompt
		if prompt == "" {
			prompt = s.Summary
		}
		if len(prompt) > 50 {
			prompt = prompt[:50] + "..."
		}

		date := s.Modified.Format("2006-01-02 15:04")
		project := filepath.Base(s.Project)
		if project == "" || project == "." {
			project = "-"
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
			shortID(s.ID), date, s.MessageCount, project, prompt)
	}

	return w.Flush()
}

func shortID(id string) string {
	return export.ShortID(id)
}

// formatListRow is extracted for testability.
func formatListRow(s session.SessionInfo) string {
	prompt := s.FirstPrompt
	if prompt == "" {
		prompt = s.Summary
	}
	if len(prompt) > 50 {
		prompt = prompt[:50] + "..."
	}

	date := s.Modified.Format("2006-01-02 15:04")
	project := filepath.Base(s.Project)
	if project == "" || project == "." {
		project = "-"
	}

	return fmt.Sprintf("%s\t%s\t%d\t%s\t%s",
		shortID(s.ID), date, s.MessageCount, project, prompt)
}
