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

// maxPromptSearchLen is the max text length per prompt entry in search results.
const maxPromptSearchLen = 200

// ExportOpts configures an export operation.
// SessionID accepts a single ID or comma-separated IDs for multi-session export.
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

// SearchOpts configures a prompt search operation.
type SearchOpts struct {
	ClaudeDir string
	Project   string
	Limit     int // max sessions to search (default: 20)
}

// PromptEntry represents a single user prompt with session metadata.
type PromptEntry struct {
	SessionID string
	Timestamp time.Time
	Text      string
}

// SearchResult contains session metadata and their user prompts.
type SearchResult struct {
	Sessions []session.SessionInfo
	Prompts  []PromptEntry
}

// ExportResult contains the outcome of an export operation.
type ExportResult struct {
	OutputPath   string
	MessageCount int
	ByteCount    int
	GistURL      string // empty if gist not requested
}

// ExportSession runs the full export pipeline: find → parse → boundary → redact → render → write.
// SessionID accepts comma-separated IDs for multi-session export.
func ExportSession(opts ExportOpts) (*ExportResult, error) {
	claudeDir := opts.ClaudeDir
	if claudeDir == "" {
		claudeDir = session.DefaultClaudeDir()
	}

	if opts.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	// Split comma-separated session IDs.
	ids := splitSessionIDs(opts.SessionID)

	// Resolve all sessions and collect messages.
	var allSessions []*session.SessionInfo
	for _, id := range ids {
		sess, err := session.FindSession(claudeDir, id)
		if err != nil {
			return nil, fmt.Errorf("session %s: %w", id, err)
		}
		if sess.FilePath == "" {
			return nil, fmt.Errorf("session %s has no file path", sess.ID)
		}
		allSessions = append(allSessions, sess)
	}

	// Parse and merge messages from all sessions.
	messages, err := mergeSessionMessages(allSessions)
	if err != nil {
		return nil, err
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

	// Build title from first session
	firstSess := allSessions[0]
	title := BuildTitle(firstSess)

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
	outPath := outputPath(outDir, firstSess.Slug, firstSess.ID, firstSess.Modified, ext)
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

// splitSessionIDs parses a comma-separated list of session IDs.
func splitSessionIDs(s string) []string {
	parts := strings.Split(s, ",")
	var ids []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			ids = append(ids, p)
		}
	}
	return ids
}

// mergeSessionMessages parses messages from multiple sessions and merges them
// chronologically, inserting session boundary markers between each session.
func mergeSessionMessages(sessions []*session.SessionInfo) ([]parser.Message, error) {
	var merged []parser.Message

	for i, sess := range sessions {
		msgs, err := parser.ParseFile(sess.FilePath, 0, 0)
		if err != nil {
			return nil, fmt.Errorf("parse session %s: %w", ShortID(sess.ID), err)
		}

		// Insert a boundary between sessions (not before the first).
		if i > 0 {
			merged = append(merged, parser.Message{
				Type:      "file-history-snapshot",
				Timestamp: msgs[0].Timestamp,
			})
		}

		merged = append(merged, msgs...)
	}

	return merged, nil
}

// SearchPrompts returns user prompts across sessions for Claude to reason over.
func SearchPrompts(opts SearchOpts) (*SearchResult, error) {
	claudeDir := opts.ClaudeDir
	if claudeDir == "" {
		claudeDir = session.DefaultClaudeDir()
	}

	sessions, err := ListSessions(claudeDir, opts.Project)
	if err != nil {
		return nil, err
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > len(sessions) {
		limit = len(sessions)
	}
	sessions = sessions[:limit]

	var prompts []PromptEntry
	for _, sess := range sessions {
		if sess.FilePath == "" {
			continue
		}

		messages, err := parser.ParseFile(sess.FilePath, 0, 0)
		if err != nil {
			continue // skip unparseable sessions
		}

		for _, msg := range messages {
			if (msg.Type == "user" || msg.Type == "queue-operation") && msg.TextContent != "" {
				text := msg.TextContent
				if len(text) > maxPromptSearchLen {
					text = text[:maxPromptSearchLen] + "..."
				}
				prompts = append(prompts, PromptEntry{
					SessionID: sess.ID,
					Timestamp: msg.Timestamp,
					Text:      text,
				})
			}
		}
	}

	return &SearchResult{Sessions: sessions, Prompts: prompts}, nil
}

// FormatPromptSearch formats search results grouped by session for Claude to scan.
func FormatPromptSearch(result *SearchResult) string {
	// Build session lookup.
	sessMap := make(map[string]*session.SessionInfo, len(result.Sessions))
	for i := range result.Sessions {
		sessMap[result.Sessions[i].ID] = &result.Sessions[i]
	}

	// Group prompts by session.
	type sessionPrompts struct {
		sess    *session.SessionInfo
		prompts []PromptEntry
	}
	bySession := make(map[string]*sessionPrompts)
	var order []string
	for _, p := range result.Prompts {
		sp, ok := bySession[p.SessionID]
		if !ok {
			sp = &sessionPrompts{sess: sessMap[p.SessionID]}
			bySession[p.SessionID] = sp
			order = append(order, p.SessionID)
		}
		sp.prompts = append(sp.prompts, p)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d user prompts across %d sessions:\n\n", len(result.Prompts), len(order)))

	for _, id := range order {
		sp := bySession[id]
		project := filepath.Base(sp.sess.Project)
		if project == "" || project == "." {
			project = "-"
		}
		summary := TruncatePrompt(sp.sess.FirstPrompt, sp.sess.Summary, MaxPromptLen)
		sb.WriteString(fmt.Sprintf("Session: %s (%s, %s) %q\n",
			ShortID(id), sp.sess.Modified.Format("2006-01-02"), project, summary))

		for _, p := range sp.prompts {
			ts := "         "
			if !p.Timestamp.IsZero() {
				ts = p.Timestamp.Format("15:04:05") + " "
			}
			sb.WriteString(fmt.Sprintf("  %s[user] %s\n", ts, p.Text))
		}
		sb.WriteString("\n")
	}

	return sb.String()
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
// The short session ID is always included to prevent collisions when
// multiple sessions share the same slug and date.
func outputPath(outDir, slug, sessionID string, modified time.Time, ext string) string {
	short := ShortID(sessionID)
	name := short
	if slug != "" {
		name = slug + "-" + short
	}
	date := modified.Format("2006-01-02")
	if date == "0001-01-01" {
		date = time.Now().Format("2006-01-02")
	}
	return filepath.Join(outDir, name+"-"+date+ext)
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
