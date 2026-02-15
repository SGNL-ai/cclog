package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sgnl-ai/cclog/internal/export"
	"github.com/sgnl-ai/cclog/internal/session"
	"github.com/spf13/cobra"
)

type exportOpts struct {
	sessionID string
	all       bool
	format    string
	fromText  string
	toText    string
	gist      bool
	public    bool
}

func exportCmd() *cobra.Command {
	var opts exportOpts

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export a session transcript",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExport(opts, os.Stdin)
		},
	}

	cmd.Flags().StringVar(&opts.sessionID, "session", "", "session ID (or prefix) to export")
	cmd.Flags().BoolVar(&opts.all, "all", false, "skip boundary detection, export everything")
	cmd.Flags().StringVar(&opts.format, "format", "html", "output format: html or md")
	cmd.Flags().StringVar(&opts.fromText, "from-text", "", "start export at message containing this text")
	cmd.Flags().StringVar(&opts.toText, "to-text", "", "end export at message containing this text")
	cmd.Flags().BoolVar(&opts.gist, "gist", false, "publish to GitHub gist")
	cmd.Flags().BoolVar(&opts.public, "public", false, "make gist public (requires --gist)")

	return cmd
}

func runExport(opts exportOpts, stdin io.Reader) error {
	claudeDir := session.DefaultClaudeDir()

	// Resolve session ID interactively if not provided
	sessionID := opts.sessionID
	if sessionID == "" {
		sess, err := pickSession(claudeDir, stdin)
		if err != nil {
			return err
		}
		sessionID = sess.ID
	}

	result, err := export.ExportSession(export.ExportOpts{
		SessionID:  sessionID,
		Format:     opts.format,
		All:        opts.all,
		FromText:   opts.fromText,
		ToText:     opts.toText,
		Gist:       opts.gist,
		GistPublic: opts.public,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Exported %d messages to %s\n", result.MessageCount, result.OutputPath)

	if result.GistURL != "" {
		fmt.Printf("Published to %s\n", result.GistURL)
	}

	return nil
}

func pickSession(claudeDir string, stdin io.Reader) (*session.SessionInfo, error) {
	sessions, err := session.Discover(claudeDir)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, fmt.Errorf("no sessions found")
	}

	fmt.Println("Available sessions:")
	fmt.Println()

	limit := 20
	if len(sessions) < limit {
		limit = len(sessions)
	}

	for i := 0; i < limit; i++ {
		s := sessions[i]
		prompt := export.TruncatePrompt(s.FirstPrompt, s.Summary, export.MaxPromptLen)
		date := s.Modified.Format("2006-01-02 15:04")
		project := filepath.Base(s.Project)
		if project == "" || project == "." {
			project = "unknown"
		}
		fmt.Printf("  [%d] %-16s %-12s %s\n", i+1, date, project, prompt)
	}

	fmt.Printf("\nSelect session (1-%d): ", limit)
	reader := bufio.NewReader(stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var idx int
	if _, err := fmt.Sscanf(input, "%d", &idx); err != nil || idx < 1 || idx > limit {
		return nil, fmt.Errorf("invalid selection: %s", input)
	}

	s := sessions[idx-1]
	return &s, nil
}
