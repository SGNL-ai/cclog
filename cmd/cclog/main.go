package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/sgnl-ai/cclog/internal/gist"
	mcpserver "github.com/sgnl-ai/cclog/internal/mcp"
	"github.com/sgnl-ai/cclog/internal/parser"
	"github.com/sgnl-ai/cclog/internal/renderer"
	"github.com/sgnl-ai/cclog/internal/scanner"
	"github.com/sgnl-ai/cclog/internal/session"
	"github.com/spf13/cobra"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cclog",
		Short: "Export Claude Code session transcripts as readable documents",
		Long:  "cclog reads Claude Code JSONL session transcripts and exports them as HTML or Markdown, with secret scrubbing and smart boundary detection.",
	}

	cmd.AddCommand(exportCmd())
	cmd.AddCommand(listCmd())
	cmd.AddCommand(setupCmd())
	cmd.AddCommand(serveCmd())

	return cmd
}

func exportCmd() *cobra.Command {
	var (
		sessionID string
		all       bool
		format    string
		doGist    bool
		public    bool
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export a session transcript",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExport(exportOpts{
				sessionID: sessionID,
				all:       all,
				format:    format,
				gist:      doGist,
				public:    public,
			})
		},
	}

	cmd.Flags().StringVar(&sessionID, "session", "", "session ID to export")
	cmd.Flags().BoolVar(&all, "all", false, "skip boundary detection, export everything")
	cmd.Flags().StringVar(&format, "format", "html", "output format: html or md")
	cmd.Flags().BoolVar(&doGist, "gist", false, "publish to GitHub gist")
	cmd.Flags().BoolVar(&public, "public", false, "make gist public (requires --gist)")

	return cmd
}

func listCmd() *cobra.Command {
	var project string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(listOpts{project: project})
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "filter by project path")

	return cmd
}

func setupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Register cclog as an MCP server in Claude Code",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetup()
		},
	}
}

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run as MCP server (stdio transport)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe()
		},
	}
}

// --- Export command ---

type exportOpts struct {
	sessionID string
	all       bool
	format    string
	gist      bool
	public    bool
}

func runExport(opts exportOpts) error {
	claudeDir := session.DefaultClaudeDir()

	// Find the session
	var sess *session.SessionInfo
	if opts.sessionID != "" {
		s, err := session.FindSession(claudeDir, opts.sessionID)
		if err != nil {
			return err
		}
		sess = s
	} else {
		// Interactive picker
		s, err := pickSession(claudeDir)
		if err != nil {
			return err
		}
		sess = s
	}

	if sess.FilePath == "" {
		return fmt.Errorf("session %s has no file path", sess.ID)
	}

	// Parse the JSONL file
	messages, err := parser.ParseFile(sess.FilePath, 0, 0)
	if err != nil {
		return fmt.Errorf("parse session: %w", err)
	}

	// Boundary detection (unless --all)
	if !opts.all {
		messages, err = applyBoundarySelection(messages)
		if err != nil {
			return err
		}
	}

	// Redact secrets
	redactor, err := scanner.NewRedactor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not initialize secret scanner: %v\n", err)
	} else {
		for i := range messages {
			messages[i].TextContent = redactor.Redact(messages[i].TextContent)
			for j := range messages[i].ToolCalls {
				messages[i].ToolCalls[j].Description = redactor.Redact(messages[i].ToolCalls[j].Description)
			}
		}
	}

	// Build title
	title := buildTitle(sess)

	// Render
	var output []byte
	ext := ".html"
	switch opts.format {
	case "md", "markdown":
		output, err = renderer.RenderMarkdown(renderer.Options{Messages: messages, Title: title})
		ext = ".md"
	default:
		output, err = renderer.RenderHTML(renderer.Options{Messages: messages, Title: title})
	}
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}

	// Write output
	outDir := outputDir()
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	slug := sess.Slug
	if slug == "" {
		slug = sess.ID[:8]
	}
	date := sess.Modified.Format("2006-01-02")
	if date == "0001-01-01" {
		date = time.Now().Format("2006-01-02")
	}
	outPath := filepath.Join(outDir, slug+"-"+date+ext)

	if err := os.WriteFile(outPath, output, 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	fmt.Printf("Exported to %s\n", outPath)

	// Optionally publish to gist
	if opts.gist {
		url, err := gist.Publish(outPath, title, opts.public)
		if err != nil {
			return fmt.Errorf("publish gist: %w", err)
		}
		fmt.Printf("Published to %s\n", url)
	}

	return nil
}

func buildTitle(sess *session.SessionInfo) string {
	if sess.Summary != "" {
		return sess.Summary
	}
	if sess.FirstPrompt != "" {
		title := sess.FirstPrompt
		if len(title) > 60 {
			title = title[:60] + "..."
		}
		return title
	}
	if sess.Slug != "" {
		return sess.Slug
	}
	return "Claude Code Session"
}

func pickSession(claudeDir string) (*session.SessionInfo, error) {
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
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var idx int
	if _, err := fmt.Sscanf(input, "%d", &idx); err != nil || idx < 1 || idx > limit {
		return nil, fmt.Errorf("invalid selection: %s", input)
	}

	s := sessions[idx-1]
	return &s, nil
}

func applyBoundarySelection(messages []parser.Message) ([]parser.Message, error) {
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
	reader := bufio.NewReader(os.Stdin)
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

func outputDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "cclog")
}

// --- List command ---

type listOpts struct {
	project string
}

func runList(opts listOpts) error {
	claudeDir := session.DefaultClaudeDir()

	var sessions []session.SessionInfo
	var err error

	if opts.project != "" {
		sessions, err = session.DiscoverForProject(claudeDir, opts.project)
	} else {
		sessions, err = session.Discover(claudeDir)
	}
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDATE\tMSGS\tPROJECT\tPROMPT")

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

		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
			shortID(s.ID), date, s.MessageCount, project, prompt)
	}

	return w.Flush()
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// --- Setup command ---

func runSetup() error {
	// Find cclog binary path
	cclogPath, err := exec.LookPath("cclog")
	if err != nil {
		// Use the current executable
		cclogPath, err = os.Executable()
		if err != nil {
			return fmt.Errorf("could not find cclog binary: %w", err)
		}
	}

	cmd := exec.Command("claude", "mcp", "add",
		"--transport", "stdio",
		"--scope", "user",
		"cclog", "--",
		cclogPath, "serve",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to register MCP server: %w", err)
	}

	fmt.Println("\ncclog MCP server registered successfully!")
	fmt.Println("You can now ask Claude to export session transcripts.")
	return nil
}

// --- Serve command ---

func runServe() error {
	return mcpserver.Run(context.Background())
}
