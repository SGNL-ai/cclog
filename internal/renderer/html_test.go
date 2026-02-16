package renderer

import (
	"testing"
	"time"

	"github.com/sgnl-ai/cclog/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderHTML_EmptyMessages(t *testing.T) {
	result, err := RenderHTML(Options{Title: "Test", Messages: nil})
	require.NoError(t, err)

	html := string(result)
	assert.Contains(t, html, "<title>Test</title>")
	assert.Contains(t, html, "</html>")
}

func TestRenderHTML_DefaultTitle(t *testing.T) {
	result, err := RenderHTML(Options{Messages: nil})
	require.NoError(t, err)
	assert.Contains(t, string(result), "<title>Claude Code Session</title>")
}

func TestRenderHTML_UserMessage(t *testing.T) {
	msgs := []parser.Message{
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:00:00Z"), TextContent: "Hello world"},
	}

	result, err := RenderHTML(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)

	html := string(result)
	assert.Contains(t, html, `class="message user"`)
	assert.Contains(t, html, "Hello world")
	assert.Contains(t, html, "07:00:00")
}

func TestRenderHTML_AssistantMessage(t *testing.T) {
	msgs := []parser.Message{
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:00:05Z"), TextContent: "I can help with that."},
	}

	result, err := RenderHTML(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)

	html := string(result)
	assert.Contains(t, html, `class="message assistant"`)
	assert.Contains(t, html, "I can help with that.")
}

func TestRenderHTML_ToolCalls(t *testing.T) {
	msgs := []parser.Message{
		{
			Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:00:05Z"),
			TextContent: "Let me check.",
			ToolCalls:   []parser.ToolCall{{Name: "Bash", Description: "Run tests"}},
		},
	}

	result, err := RenderHTML(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)

	html := string(result)
	assert.Contains(t, html, `class="tool-call"`)
	assert.Contains(t, html, "● Bash")
	assert.Contains(t, html, "Run tests")
}

func TestRenderHTML_Boundary(t *testing.T) {
	msgs := []parser.Message{
		{Type: "file-history-snapshot", Timestamp: ts("2026-02-13T07:01:00Z")},
	}

	result, err := RenderHTML(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)

	html := string(result)
	assert.Contains(t, html, `class="boundary"`)
	assert.Contains(t, html, "session boundary")
}

func TestRenderHTML_QueueOperation(t *testing.T) {
	msgs := []parser.Message{
		{Type: "queue-operation", TextContent: "also do this", Timestamp: ts("2026-02-13T08:00:00Z")},
	}

	result, err := RenderHTML(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)

	html := string(result)
	assert.Contains(t, html, "also do this")
	assert.Contains(t, html, `class="message user"`)
}

func TestRenderHTML_SkipsEmptyUserMessage(t *testing.T) {
	msgs := []parser.Message{
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:00:00Z"), TextContent: ""},
	}

	result, err := RenderHTML(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)

	html := string(result)
	assert.NotContains(t, html, `class="message user"`)
}

func TestRenderHTML_SkipsEmptyAssistantMessage(t *testing.T) {
	msgs := []parser.Message{
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:00:05Z"), TextContent: ""},
	}

	result, err := RenderHTML(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)

	html := string(result)
	assert.NotContains(t, html, `class="message assistant"`)
}

func TestRenderHTML_HTMLEscaping(t *testing.T) {
	msgs := []parser.Message{
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:00:00Z"), TextContent: `<script>alert("xss")</script>`},
	}

	result, err := RenderHTML(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)

	html := string(result)
	assert.NotContains(t, html, "<script>")
	assert.Contains(t, html, "&lt;script&gt;")
}

func TestRenderHTML_ParagraphBreaks(t *testing.T) {
	msgs := []parser.Message{
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:00:00Z"), TextContent: "Para one\n\nPara two"},
	}

	result, err := RenderHTML(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)

	html := string(result)
	assert.Contains(t, html, "<p>Para one</p>")
	assert.Contains(t, html, "<p>Para two</p>")
}

func TestRenderHTML_ToolCallWithoutDescription(t *testing.T) {
	msgs := []parser.Message{
		{
			Type: "assistant", Role: "assistant",
			ToolCalls: []parser.ToolCall{{Name: "Read"}},
		},
	}

	result, err := RenderHTML(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)

	html := string(result)
	assert.Contains(t, html, "● Read</div>")
}

func TestRenderHTML_SelfContained(t *testing.T) {
	result, err := RenderHTML(Options{Title: "Test"})
	require.NoError(t, err)

	html := string(result)
	assert.Contains(t, html, "<!DOCTYPE html>")
	assert.Contains(t, html, "<style>")
	assert.Contains(t, html, "</html>")
}

func TestEscapeAndFormat(t *testing.T) {
	assert.Equal(t, "<p>hello</p>", escapeAndFormat("hello"))
	assert.Contains(t, escapeAndFormat("a\n\nb"), "<p>a</p>")
	assert.Contains(t, escapeAndFormat("a\n\nb"), "<p>b</p>")
	assert.Contains(t, escapeAndFormat("a\nb"), "<br>")
}

func TestRenderHTML_QueueOperationEmpty(t *testing.T) {
	msgs := []parser.Message{
		{Type: "queue-operation", TextContent: ""},
	}

	result, err := RenderHTML(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)
	assert.NotContains(t, string(result), "queued")
}

func TestRenderHTML_TitleXSSPrevention(t *testing.T) {
	result, err := RenderHTML(Options{Title: `<script>alert("xss")</script>`})
	require.NoError(t, err)

	html := string(result)
	assert.NotContains(t, html, "<script>alert")
	assert.Contains(t, html, "&lt;script&gt;")
}

func TestRenderHTML_FullConversation(t *testing.T) {
	msgs := []parser.Message{
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:00:00Z"), TextContent: "Fix the bug"},
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:00:05Z"), TextContent: "Looking at it.", ToolCalls: []parser.ToolCall{{Name: "Read", Description: "/src/main.go"}}},
		{Type: "file-history-snapshot", Timestamp: ts("2026-02-13T07:01:00Z")},
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:01:01Z"), TextContent: "Any progress?"},
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:01:10Z"), TextContent: "Fixed it."},
	}

	result, err := RenderHTML(Options{Title: "Bug Fix", Messages: msgs})
	require.NoError(t, err)

	html := string(result)
	// Verify structural elements appear in correct order
	userIdx := indexOf(html, `class="message user"`)
	assistantIdx := indexOf(html, `class="message assistant"`)
	boundaryIdx := indexOf(html, `class="boundary"`)
	require.True(t, userIdx > 0, "should have user message")
	require.True(t, assistantIdx > userIdx, "assistant should come after user")
	require.True(t, boundaryIdx > assistantIdx, "boundary should come after assistant")

	// Verify content is within its message context
	assert.Contains(t, html, "Fix the bug")
	assert.Contains(t, html, "Looking at it.")
	assert.Contains(t, html, "● Read")
	assert.Contains(t, html, "Any progress?")
	assert.Contains(t, html, "Fixed it.")
}

func TestRenderHTML_BoundaryWithoutTimestamp(t *testing.T) {
	msgs := []parser.Message{
		{Type: "file-history-snapshot"}, // zero timestamp
	}

	result, err := RenderHTML(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)
	html := string(result)
	assert.Contains(t, html, "session boundary")
	// Should not have a timestamp span
	assert.NotContains(t, html, "0001-01-01")
}

func TestRenderHTML_AssistantToolCallsOnlyNoText(t *testing.T) {
	msgs := []parser.Message{
		{Type: "assistant", Role: "assistant", ToolCalls: []parser.ToolCall{
			{Name: "Bash", Description: "go test ./..."},
			{Name: "Read", Description: "/foo.go"},
		}},
	}

	result, err := RenderHTML(Options{Title: "Test", Messages: msgs})
	require.NoError(t, err)
	html := string(result)
	assert.Contains(t, html, `class="message assistant"`)
	assert.Contains(t, html, "● Bash")
	assert.Contains(t, html, "● Read")
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func ts(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
