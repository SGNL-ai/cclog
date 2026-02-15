package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sgnl-ai/cclog/internal/parser"
	"github.com/sgnl-ai/cclog/internal/renderer"
	"github.com/sgnl-ai/cclog/internal/scanner"
	"github.com/sgnl-ai/cclog/internal/session"
)

// ExportInput defines the parameters for the export_transcript tool.
type ExportInput struct {
	SessionID string `json:"session_id,omitempty" jsonschema:"session ID to export"`
	Format    string `json:"format,omitempty" jsonschema:"output format: html or md (default: html)"`
	FromText  string `json:"from_text,omitempty" jsonschema:"search string to find start boundary"`
	ToText    string `json:"to_text,omitempty" jsonschema:"search string to find end boundary"`
	All       bool   `json:"all,omitempty" jsonschema:"export entire session without boundary selection"`
	Gist      bool   `json:"gist,omitempty" jsonschema:"publish to GitHub gist"`
}

// ListInput defines the parameters for the list_sessions tool.
type ListInput struct {
	Project string `json:"project,omitempty" jsonschema:"filter by project path"`
	Limit   int    `json:"limit,omitempty" jsonschema:"max number of sessions to return (default: 20)"`
}

// NewServer creates an MCP server with cclog tools registered.
func NewServer() *gomcp.Server {
	server := gomcp.NewServer(
		&gomcp.Implementation{
			Name:    "cclog",
			Version: "1.0.0",
		},
		&gomcp.ServerOptions{
			Instructions: "cclog exports Claude Code session transcripts as HTML or Markdown. Use export_transcript to export sessions and list_sessions to browse available sessions.",
		},
	)

	gomcp.AddTool(server, &gomcp.Tool{
		Name:        "export_transcript",
		Description: "Export a Claude Code session transcript as HTML or Markdown",
	}, func(ctx context.Context, req *gomcp.CallToolRequest, input ExportInput) (*gomcp.CallToolResult, any, error) {
		result, err := HandleExport(input)
		if err != nil {
			return &gomcp.CallToolResult{
				Content: []gomcp.Content{&gomcp.TextContent{Text: fmt.Sprintf("Error: %v", err)}},
				IsError: true,
			}, nil, nil
		}
		return &gomcp.CallToolResult{
			Content: []gomcp.Content{&gomcp.TextContent{Text: result}},
		}, nil, nil
	})

	gomcp.AddTool(server, &gomcp.Tool{
		Name:        "list_sessions",
		Description: "List available Claude Code sessions",
	}, func(ctx context.Context, req *gomcp.CallToolRequest, input ListInput) (*gomcp.CallToolResult, any, error) {
		result, err := HandleList(input)
		if err != nil {
			return &gomcp.CallToolResult{
				Content: []gomcp.Content{&gomcp.TextContent{Text: fmt.Sprintf("Error: %v", err)}},
				IsError: true,
			}, nil, nil
		}
		return &gomcp.CallToolResult{
			Content: []gomcp.Content{&gomcp.TextContent{Text: result}},
		}, nil, nil
	})

	return server
}

// Run starts the MCP server on the stdio transport.
func Run(ctx context.Context) error {
	server := NewServer()
	return server.Run(ctx, &gomcp.StdioTransport{})
}

// HandleExport processes an export request and returns a result string.
func HandleExport(input ExportInput) (string, error) {
	if input.SessionID == "" {
		return "", fmt.Errorf("session_id is required. Use list_sessions to find available session IDs")
	}

	claudeDir := session.DefaultClaudeDir()
	sess, err := session.FindSession(claudeDir, input.SessionID)
	if err != nil {
		return "", err
	}

	if sess.FilePath == "" {
		return "", fmt.Errorf("session %s has no file path", sess.ID)
	}

	messages, err := parser.ParseFile(sess.FilePath, 0, 0)
	if err != nil {
		return "", fmt.Errorf("parse session: %w", err)
	}

	// Apply boundary selection via text search
	if !input.All {
		messages = applyTextBoundaries(messages, input.FromText, input.ToText)
	}

	// Redact secrets
	redactor, err := scanner.NewRedactor()
	if err == nil {
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
	format := input.Format
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
		return "", fmt.Errorf("render: %w", err)
	}

	// Write output
	outDir := outputDir()
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	slug := sess.Slug
	if slug == "" && len(sess.ID) >= 8 {
		slug = sess.ID[:8]
	}
	date := sess.Modified.Format("2006-01-02")
	if date == "0001-01-01" {
		date = time.Now().Format("2006-01-02")
	}
	outPath := filepath.Join(outDir, slug+"-"+date+ext)

	if err := os.WriteFile(outPath, output, 0o644); err != nil {
		return "", fmt.Errorf("write output: %w", err)
	}

	return fmt.Sprintf("Exported to %s (%d messages, %d bytes)", outPath, len(messages), len(output)), nil
}

// HandleList processes a list request and returns a result string.
func HandleList(input ListInput) (string, error) {
	claudeDir := session.DefaultClaudeDir()

	var sessions []session.SessionInfo
	var err error

	if input.Project != "" {
		sessions, err = session.DiscoverForProject(claudeDir, input.Project)
	} else {
		sessions, err = session.Discover(claudeDir)
	}
	if err != nil {
		return "", err
	}

	limit := input.Limit
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

	return sb.String(), nil
}

func applyTextBoundaries(messages []parser.Message, fromText, toText string) []parser.Message {
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

func outputDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "cclog")
}
