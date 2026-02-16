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
	assert.Contains(t, md, "07:00:00 Hello world")
}

func TestRenderMarkdown_AssistantMessage(t *testing.T) {
	msgs := []parser.Message{
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:00:05Z"), TextContent: "I can help."},
	}

	result, err := RenderMarkdown(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)

	md := string(result)
	assert.Contains(t, md, "### Assistant\n")
	assert.Contains(t, md, "07:00:05 I can help.")
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
	assert.Contains(t, md, "### User")
	assert.Contains(t, md, "also do this")
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
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:00:00Z"), TextContent: "line one\nline two\nline three"},
	}

	result, err := RenderMarkdown(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)

	md := string(result)
	// Timestamp on first line, content flows naturally.
	assert.Contains(t, md, "07:00:00 line one\nline two\nline three")
}

func TestRenderMarkdown_QueueOperationEmpty(t *testing.T) {
	msgs := []parser.Message{
		{Type: "queue-operation", TextContent: ""},
	}

	result, err := RenderMarkdown(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)
	assert.NotContains(t, string(result), "User")
}

func TestRenderMarkdown_GroupsConsecutiveSameRole(t *testing.T) {
	msgs := []parser.Message{
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:00:05Z"), TextContent: "First thing."},
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:00:10Z"), TextContent: "Second thing."},
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:00:15Z"), TextContent: "Third thing."},
	}

	result, err := RenderMarkdown(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)

	md := string(result)
	// Should have exactly one "### Assistant" header
	first := indexOf(md, "### Assistant")
	second := indexOf(md[first+1:], "### Assistant")
	assert.Equal(t, -1, second, "should only have one Assistant header")

	assert.Contains(t, md, "07:00:05 First thing.")
	assert.Contains(t, md, "07:00:10 Second thing.")
	assert.Contains(t, md, "07:00:15 Third thing.")
}

func TestRenderMarkdown_CodeBlocksPreserved(t *testing.T) {
	// Code fences at column 0 should render correctly in GFM.
	msgs := []parser.Message{
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:00:05Z"),
			TextContent: "Here is code:\n\n```javascript\nconst x = 1;\n```\n\nDone."},
	}

	result, err := RenderMarkdown(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)

	md := string(result)
	assert.Contains(t, md, "```javascript\nconst x = 1;\n```")
	assert.Contains(t, md, "Done.")
}

func TestRenderMarkdown_FullConversation(t *testing.T) {
	msgs := []parser.Message{
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:00:00Z"), TextContent: "Fix the bug"},
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:00:05Z"), TextContent: "On it.", ToolCalls: []parser.ToolCall{{Name: "Read", Description: "/main.go"}}},
		{Type: "file-history-snapshot"},
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:01:01Z"), TextContent: "Done?"},
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:01:10Z"), TextContent: "Yes."},
	}

	result, err := RenderMarkdown(Options{Title: "Session", Messages: msgs})
	require.NoError(t, err)

	md := string(result)
	assert.Contains(t, md, "# Session")
	assert.Contains(t, md, "Fix the bug")
	assert.Contains(t, md, "On it.")
	assert.Contains(t, md, "- `Read` — /main.go")
	assert.Contains(t, md, "---")
	assert.Contains(t, md, "Done?")
	assert.Contains(t, md, "Yes.")

	// Verify ordering
	userIdx := indexOf(md, "Fix the bug")
	assistantIdx := indexOf(md, "On it.")
	boundaryIdx := indexOf(md, "---\n")
	assert.True(t, userIdx < assistantIdx, "user should come before assistant")
	assert.True(t, assistantIdx < boundaryIdx, "assistant should come before boundary")
}

func TestRenderMarkdown_AssistantNoTimestamp(t *testing.T) {
	msgs := []parser.Message{
		{Type: "assistant", Role: "assistant", TextContent: "No timestamp here."},
	}

	result, err := RenderMarkdown(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)
	md := string(result)
	assert.Contains(t, md, "No timestamp here.")
}

// indexOf is defined in html_test.go (same package)
