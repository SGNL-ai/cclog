package renderer

import (
	"testing"

	"github.com/sgnl-ai/cclog/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderMarkdown_EmptyMessages(t *testing.T) {
	result, err := RenderMarkdown(Options{Title: "Test", Messages: nil})
	require.NoError(t, err)
	assert.Equal(t, "# Test\n\n", string(result))
}

func TestRenderMarkdown_DefaultTitle(t *testing.T) {
	result, err := RenderMarkdown(Options{Messages: nil})
	require.NoError(t, err)
	assert.Contains(t, string(result), "# Claude Code Session")
}

func TestRenderMarkdown_UserMessage(t *testing.T) {
	msgs := []parser.Message{
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:00:00Z"), TextContent: "Hello world"},
	}

	result, err := RenderMarkdown(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)

	md := string(result)
	assert.Contains(t, md, "### User\n")
	assert.Contains(t, md, "> Hello world")
	assert.Contains(t, md, "_07:00:00_")
}

func TestRenderMarkdown_AssistantMessage(t *testing.T) {
	msgs := []parser.Message{
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:00:05Z"), TextContent: "I can help."},
	}

	result, err := RenderMarkdown(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)

	md := string(result)
	assert.Contains(t, md, "### Assistant\n")
	assert.Contains(t, md, "I can help.")
}

func TestRenderMarkdown_ToolCalls(t *testing.T) {
	msgs := []parser.Message{
		{
			Type: "assistant", Role: "assistant",
			ToolCalls: []parser.ToolCall{
				{Name: "Bash", Description: "Run tests"},
				{Name: "Read"},
			},
		},
	}

	result, err := RenderMarkdown(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)

	md := string(result)
	assert.Contains(t, md, "- `Bash` — Run tests")
	assert.Contains(t, md, "- `Read`\n")
}

func TestRenderMarkdown_Boundary(t *testing.T) {
	msgs := []parser.Message{
		{Type: "file-history-snapshot", Timestamp: ts("2026-02-13T07:01:00Z")},
	}

	result, err := RenderMarkdown(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)
	assert.Contains(t, string(result), "---")
}

func TestRenderMarkdown_QueueOperation(t *testing.T) {
	msgs := []parser.Message{
		{Type: "queue-operation", TextContent: "also do this", Timestamp: ts("2026-02-13T08:00:00Z")},
	}

	result, err := RenderMarkdown(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)

	md := string(result)
	assert.Contains(t, md, "_(queued)_")
	assert.Contains(t, md, "> also do this")
}

func TestRenderMarkdown_SkipsEmptyUserMessage(t *testing.T) {
	msgs := []parser.Message{
		{Type: "user", Role: "user", TextContent: ""},
	}

	result, err := RenderMarkdown(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)
	assert.NotContains(t, string(result), "### User")
}

func TestRenderMarkdown_SkipsEmptyAssistantMessage(t *testing.T) {
	msgs := []parser.Message{
		{Type: "assistant", Role: "assistant", TextContent: ""},
	}

	result, err := RenderMarkdown(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)
	assert.NotContains(t, string(result), "### Assistant")
}

func TestRenderMarkdown_MultilineUser(t *testing.T) {
	msgs := []parser.Message{
		{Type: "user", Role: "user", TextContent: "line one\nline two\nline three"},
	}

	result, err := RenderMarkdown(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)

	md := string(result)
	assert.Contains(t, md, "> line one\n> line two\n> line three")
}

func TestRenderMarkdown_QueueOperationEmpty(t *testing.T) {
	msgs := []parser.Message{
		{Type: "queue-operation", TextContent: ""},
	}

	result, err := RenderMarkdown(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)
	assert.NotContains(t, string(result), "queued")
}
