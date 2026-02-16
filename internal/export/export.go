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

// MaxPromptLen is the standard truncation length for prompt/title display.
const MaxPromptLen = 60

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
	OutputDir  string // injectable; defaults to ~/cclog
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
	messages, err = RedactMessages(messages)
	if err != nil {
		return nil, fmt.Errorf("secret redaction: %w", err)
	}

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
	outDir := opts.OutputDir
	if outDir == "" {
		outDir, err = DefaultOutputDir()
		if err != nil {
			return nil, fmt.Errorf("resolve output dir: %w", err)
		}
	}
	outPath := outputPath(outDir, sess.Slug, sess.ID, sess.Modified, ext)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
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
		prompt := TruncatePrompt(s.FirstPrompt, s.Summary, MaxPromptLen)

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
		if len(title) > MaxPromptLen {
			title = title[:MaxPromptLen] + "..."
		}
		return title
	}
	if sess.Slug != "" {
		return sess.Slug
	}
	return "Claude Code Session"
}

// DefaultOutputDir returns the default output directory for exported files.
func DefaultOutputDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, "cclog"), nil
}

// outputPath computes the full output file path from session metadata.
func outputPath(outDir, slug, sessionID string, modified time.Time, ext string) string {
	s := slug
	if s == "" && len(sessionID) >= 8 {
		s = sessionID[:8]
	}
	date := modified.Format("2006-01-02")
	if date == "0001-01-01" {
		date = time.Now().Format("2006-01-02")
	}
	return filepath.Join(outDir, s+"-"+date+ext)
}

// RedactMessages scans messages for secrets and replaces them with [REDACTED].
// Returns an error if the redactor cannot be initialized (security guarantee).
func RedactMessages(messages []parser.Message) ([]parser.Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	redactor, err := scanner.NewRedactor()
	if err != nil {
		return nil, fmt.Errorf("initialize secret scanner: %w", err)
	}

	// Copy to avoid mutating caller's slice.
	out := make([]parser.Message, len(messages))
	copy(out, messages)

	for i := range out {
		out[i].TextContent = redactor.Redact(out[i].TextContent)
		if len(out[i].ToolCalls) > 0 {
			tcs := make([]parser.ToolCall, len(out[i].ToolCalls))
			copy(tcs, out[i].ToolCalls)
			for j := range tcs {
				tcs[j].Description = redactor.Redact(tcs[j].Description)
			}
			out[i].ToolCalls = tcs
		}
	}
	return out, nil
}

// ApplyTextBoundaries filters messages by matching from/to text (case-insensitive).
func ApplyTextBoundaries(messages []parser.Message, fromText, toText string) []parser.Message {
	if fromText == "" && toText == "" {
		return messages
	}

	startIdx := 0
	endIdx := len(messages)

	if fromText != "" {
		lower := strings.ToLower(fromText)
		for i, msg := range messages {
			if strings.Contains(strings.ToLower(msg.TextContent), lower) {
				startIdx = i
				break
			}
		}
	}

	if toText != "" {
		lower := strings.ToLower(toText)
		for i := len(messages) - 1; i >= startIdx; i-- {
			if strings.Contains(strings.ToLower(messages[i].TextContent), lower) {
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

// TruncatePrompt returns the best available display text for a session.
// Prefers summary (AI-generated, more descriptive) over firstPrompt.
func TruncatePrompt(prompt, summary string, max int) string {
	s := summary
	if s == "" {
		s = prompt
	}
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
