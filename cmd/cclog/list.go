package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/sgnl-ai/cclog/internal/export"
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
		prompt := export.TruncatePrompt(s.FirstPrompt, s.Summary, export.MaxPromptLen)
		date := s.Modified.Format("2006-01-02 15:04")
		project := filepath.Base(s.Project)
		if project == "" || project == "." {
			project = "-"
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
			export.ShortID(s.ID), date, s.MessageCount, project, prompt)
	}

	return w.Flush()
}
