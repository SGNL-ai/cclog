package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sgnl-ai/cclog/internal/gist"
	"github.com/sgnl-ai/cclog/internal/parser"
	"github.com/sgnl-ai/cclog/internal/renderer"
	"github.com/sgnl-ai/cclog/internal/scanner"
	"github.com/sgnl-ai/cclog/internal/session"
)

// ExportOpts configures an export operation.
type ExportOpts struct {
	SessionID  string
	Format     string // "html" or "md"
	All        bool   // skip boundary filtering
	FromText   string // text boundary start
	ToText     string // text boundary end
	Gist       bool
	GistPublic bool
	ClaudeDir  string // injectable; defaults to session.DefaultClaudeDir()
}

// ExportResult contains the outcome of an export operation.
type ExportResult struct {
	OutputPath   string
	MessageCount int
	ByteCount    int
	GistURL      string // empty if gist not requested
}

// ExportSession runs the full export pipeline: find → parse → boundary → redact → render → write.
func ExportSession(opts ExportOpts) (*ExportResult, error) {
	claudeDir := opts.ClaudeDir
	if claudeDir == "" {
		claudeDir = session.DefaultClaudeDir()
	}

	if opts.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	sess, err := session.FindSession(claudeDir, opts.SessionID)
	if err != nil {
		return nil, err
	}

	if sess.FilePath == "" {
		return nil, fmt.Errorf("session %s has no file path", sess.ID)
	}

	messages, err := parser.ParseFile(sess.FilePath, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}

	// Apply text boundaries (unless --all)
	if !opts.All {
		messages = ApplyTextBoundaries(messages, opts.FromText, opts.ToText)
	}

	// Redact secrets
	messages = RedactMessages(messages)

	// Build title
	title := BuildTitle(sess)

	// Render
	format := opts.Format
	if format == "" {
		format = "html"
	}

	var output []byte
	ext := ".html"
	switch format {
	case "md", "markdown":
		output, err = renderer.RenderMarkdown(renderer.Options{Messages: messages, Title: title})
		ext = ".md"
	default:
		output, err = renderer.RenderHTML(renderer.Options{Messages: messages, Title: title})
	}
	if err != nil {
		return nil, fmt.Errorf("render: %w", err)
	}

	// Write output
	outPath := OutputPath(sess.Slug, sess.ID, sess.Modified, ext)
	outDir := filepath.Dir(outPath)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	if err := os.WriteFile(outPath, output, 0o644); err != nil {
		return nil, fmt.Errorf("write output: %w", err)
	}

	result := &ExportResult{
		OutputPath:   outPath,
		MessageCount: len(messages),
		ByteCount:    len(output),
	}

	// Optionally publish to gist
	if opts.Gist {
		url, err := gist.Publish(outPath, title, opts.GistPublic)
		if err != nil {
			return nil, fmt.Errorf("publish gist: %w", err)
		}
		result.GistURL = url
	}

	return result, nil
}

// ListSessions returns sessions, optionally filtered by project.
func ListSessions(claudeDir, project string) ([]session.SessionInfo, error) {
	if claudeDir == "" {
		claudeDir = session.DefaultClaudeDir()
	}

	if project != "" {
		return session.DiscoverForProject(claudeDir, project)
	}
	return session.Discover(claudeDir)
}

// FormatSessionList formats a list of sessions as a human-readable string.
func FormatSessionList(sessions []session.SessionInfo, limit int) string {
	if limit <= 0 {
		limit = 20
	}
	if limit > len(sessions) {
		limit = len(sessions)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d sessions (showing %d):\n\n", len(sessions), limit))

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
			project = "-"
		}

		sb.WriteString(fmt.Sprintf("- %s | %s | %s | %s\n", s.ID, date, project, prompt))
	}

	return sb.String()
}

// BuildTitle constructs a title for the exported document from session metadata.
func BuildTitle(sess *session.SessionInfo) string {
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

// OutputDir returns the default output directory for exported files.
func OutputDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "cclog")
}

// OutputPath computes the full output file path from session metadata.
func OutputPath(slug, sessionID string, modified time.Time, ext string) string {
	s := slug
	if s == "" && len(sessionID) >= 8 {
		s = sessionID[:8]
	}
	date := modified.Format("2006-01-02")
	if date == "0001-01-01" {
		date = time.Now().Format("2006-01-02")
	}
	return filepath.Join(OutputDir(), s+"-"+date+ext)
}

// RedactMessages scans messages for secrets and replaces them with [REDACTED].
func RedactMessages(messages []parser.Message) []parser.Message {
	redactor, err := scanner.NewRedactor()
	if err != nil {
		return messages
	}

	for i := range messages {
		messages[i].TextContent = redactor.Redact(messages[i].TextContent)
		for j := range messages[i].ToolCalls {
			messages[i].ToolCalls[j].Description = redactor.Redact(messages[i].ToolCalls[j].Description)
		}
	}
	return messages
}

// ApplyTextBoundaries filters messages by matching from/to text (case-insensitive).
func ApplyTextBoundaries(messages []parser.Message, fromText, toText string) []parser.Message {
	if fromText == "" && toText == "" {
		return messages
	}

	startIdx := 0
	endIdx := len(messages)

	if fromText != "" {
		for i, msg := range messages {
			if strings.Contains(strings.ToLower(msg.TextContent), strings.ToLower(fromText)) {
				startIdx = i
				break
			}
		}
	}

	if toText != "" {
		for i := len(messages) - 1; i >= startIdx; i-- {
			if strings.Contains(strings.ToLower(messages[i].TextContent), strings.ToLower(toText)) {
				endIdx = i + 1
				break
			}
		}
	}

	return messages[startIdx:endIdx]
}

// ShortID returns the first 8 characters of a session ID.
func ShortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
