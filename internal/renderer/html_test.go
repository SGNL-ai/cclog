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
	assert.Contains(t, html, "queued")
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

func ts(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
