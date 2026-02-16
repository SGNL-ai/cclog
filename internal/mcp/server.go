package mcp

import (
	"context"
	"fmt"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sgnl-ai/cclog/internal/export"
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

	result, err := export.ExportSession(export.ExportOpts{
		SessionID: input.SessionID,
		Format:    input.Format,
		All:       input.All,
		FromText:  input.FromText,
		ToText:    input.ToText,
		Gist:      input.Gist,
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Exported to %s (%d messages, %d bytes)", result.OutputPath, result.MessageCount, result.ByteCount), nil
}

// HandleList processes a list request and returns a result string.
func HandleList(input ListInput) (string, error) {
	sessions, err := export.ListSessions("", input.Project)
	if err != nil {
		return "", err
	}

	return export.FormatSessionList(sessions, input.Limit), nil
}
