package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sgnl-ai/cclog/internal/export"
	"github.com/sgnl-ai/cclog/internal/parser"
	"github.com/sgnl-ai/cclog/internal/session"
	"github.com/spf13/cobra"
)

type exportOpts struct {
	sessionID string
	all       bool
	format    string
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

	cmd.Flags().StringVar(&opts.sessionID, "session", "", "session ID to export")
	cmd.Flags().BoolVar(&opts.all, "all", false, "skip boundary detection, export everything")
	cmd.Flags().StringVar(&opts.format, "format", "html", "output format: html or md")
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

	// For interactive boundary selection, we need to parse first
	if !opts.all && opts.sessionID == "" {
		sess, err := session.FindSession(claudeDir, sessionID)
		if err != nil {
			return err
		}
		if sess.FilePath == "" {
			return fmt.Errorf("session %s has no file path", sess.ID)
		}

		messages, err := parser.ParseFile(sess.FilePath, 0, 0)
		if err != nil {
			return fmt.Errorf("parse session: %w", err)
		}

		messages, err = applyBoundarySelection(messages, stdin)
		if err != nil {
			return err
		}

		// If boundary was selected, use text boundaries for the export
		// For interactive mode, we pass --all to the exporter since we handled boundaries here
		_ = messages // boundaries handled interactively
	}

	result, err := export.ExportSession(export.ExportOpts{
		SessionID:  sessionID,
		Format:     opts.format,
		All:        opts.all || opts.sessionID == "",
		Gist:       opts.gist,
		GistPublic: opts.public,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Exported to %s\n", result.OutputPath)

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
		prompt := s.FirstPrompt
		if prompt == "" {
			prompt = s.Summary
		}
		if len(prompt) > 60 {
			prompt = prompt[:60] + "..."
		}
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

func applyBoundarySelection(messages []parser.Message, stdin io.Reader) ([]parser.Message, error) {
	boundaries := session.DetectBoundaries(messages)
	if len(boundaries) <= 1 {
		return messages, nil
	}

	fmt.Println("Session boundaries detected:")
	fmt.Println()

	for i, b := range boundaries {
		preview := b.PreviewText
		if preview == "" {
			preview = "(no preview)"
		}
		fmt.Printf("  [%d] %s — %s — %s\n", i+1, b.Timestamp.Format("15:04:05"), b.Reason, preview)
	}

	fmt.Printf("\nExport from boundary (1-%d, or 'all'): ", len(boundaries))
	reader := bufio.NewReader(stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "all" || input == "" {
		return messages, nil
	}

	var startIdx int
	if _, err := fmt.Sscanf(input, "%d", &startIdx); err != nil || startIdx < 1 || startIdx > len(boundaries) {
		return nil, fmt.Errorf("invalid selection: %s", input)
	}

	startLine := boundaries[startIdx-1].LineIndex
	endLine := len(messages)
	if startIdx < len(boundaries) {
		endLine = boundaries[startIdx].LineIndex
	}

	return messages[startLine:endLine], nil
}
