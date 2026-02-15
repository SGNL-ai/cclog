package mcp

import (
	"strings"
	"testing"

	"github.com/sgnl-ai/cclog/internal/parser"
	"github.com/sgnl-ai/cclog/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	server := NewServer()
	assert.NotNil(t, server)
}

func TestHandleExport_MissingSessionID(t *testing.T) {
	_, err := HandleExport(ExportInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session_id is required")
}

func TestHandleExport_NonexistentSession(t *testing.T) {
	_, err := HandleExport(ExportInput{SessionID: "nonexistent-session-id"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestHandleList_ReturnsResults(t *testing.T) {
	// This test uses real session data if available
	result, err := HandleList(ListInput{Limit: 5})
	require.NoError(t, err)
	assert.Contains(t, result, "Found")
	assert.Contains(t, result, "sessions")
}

func TestHandleList_WithLimit(t *testing.T) {
	result, err := HandleList(ListInput{Limit: 2})
	require.NoError(t, err)
	assert.Contains(t, result, "showing 2")
}

func TestHandleList_WithProject(t *testing.T) {
	// Non-existent project should return empty
	result, err := HandleList(ListInput{Project: "/nonexistent/path"})
	require.NoError(t, err)
	assert.Contains(t, result, "showing 0")
}

func TestApplyTextBoundaries_NoFilter(t *testing.T) {
	msgs := []parser.Message{
		{TextContent: "first"},
		{TextContent: "second"},
		{TextContent: "third"},
	}

	result := applyTextBoundaries(msgs, "", "")
	assert.Len(t, result, 3)
}

func TestApplyTextBoundaries_FromText(t *testing.T) {
	msgs := []parser.Message{
		{TextContent: "first message"},
		{TextContent: "start here please"},
		{TextContent: "third message"},
	}

	result := applyTextBoundaries(msgs, "start here", "")
	assert.Len(t, result, 2)
	assert.Equal(t, "start here please", result[0].TextContent)
}

func TestApplyTextBoundaries_ToText(t *testing.T) {
	msgs := []parser.Message{
		{TextContent: "first message"},
		{TextContent: "stop here please"},
		{TextContent: "third message"},
	}

	result := applyTextBoundaries(msgs, "", "stop here")
	assert.Len(t, result, 2)
	assert.Equal(t, "first message", result[0].TextContent)
}

func TestApplyTextBoundaries_CaseInsensitive(t *testing.T) {
	msgs := []parser.Message{
		{TextContent: "first"},
		{TextContent: "START HERE"},
		{TextContent: "third"},
	}

	result := applyTextBoundaries(msgs, "start here", "")
	assert.Len(t, result, 2)
}

func TestBuildTitle_Summary(t *testing.T) {
	sess := &session.SessionInfo{Summary: "My Summary"}
	assert.Equal(t, "My Summary", buildTitle(sess))
}

func TestBuildTitle_FirstPrompt(t *testing.T) {
	sess := &session.SessionInfo{FirstPrompt: "Hello world"}
	assert.Equal(t, "Hello world", buildTitle(sess))
}

func TestBuildTitle_LongPromptTruncated(t *testing.T) {
	long := "This is a very long prompt that exceeds sixty characters in length and should be truncated"
	sess := &session.SessionInfo{FirstPrompt: long}
	title := buildTitle(sess)
	assert.Len(t, title, 63) // 60 + "..."
}

func TestBuildTitle_Slug(t *testing.T) {
	sess := &session.SessionInfo{Slug: "my-slug"}
	assert.Equal(t, "my-slug", buildTitle(sess))
}

func TestBuildTitle_Default(t *testing.T) {
	sess := &session.SessionInfo{}
	assert.Equal(t, "Claude Code Session", buildTitle(sess))
}

func TestHandleExport_RealSession(t *testing.T) {
	// Find a real session to export
	result, err := HandleList(ListInput{Limit: 1})
	require.NoError(t, err)

	// Extract session ID from the list output
	lines := strings.Split(result, "\n")
	if len(lines) < 3 {
		t.Skip("No sessions available for integration test")
	}

	// Parse the session ID from "- <uuid> | ..."
	line := lines[2] // first session line after header
	if !strings.HasPrefix(line, "- ") {
		t.Skip("Could not parse session list output")
	}
	parts := strings.SplitN(line[2:], " | ", 2)
	sessionID := strings.TrimSpace(parts[0])

	// Export as markdown with --all
	outDir := t.TempDir()
	origOutputDir := outputDir
	// We can't override outputDir easily, so export to real ~/cclog
	exported, err := HandleExport(ExportInput{
		SessionID: sessionID,
		Format:    "md",
		All:       true,
	})
	require.NoError(t, err)
	assert.Contains(t, exported, "Exported to")
	assert.Contains(t, exported, "messages")
	_ = outDir
	_ = origOutputDir
}

func TestHandleExport_HTMLFormat(t *testing.T) {
	result, err := HandleList(ListInput{Limit: 1})
	require.NoError(t, err)

	lines := strings.Split(result, "\n")
	if len(lines) < 3 {
		t.Skip("No sessions available")
	}

	line := lines[2]
	if !strings.HasPrefix(line, "- ") {
		t.Skip("Could not parse session list")
	}
	parts := strings.SplitN(line[2:], " | ", 2)
	sessionID := strings.TrimSpace(parts[0])

	exported, err := HandleExport(ExportInput{
		SessionID: sessionID,
		Format:    "html",
		All:       true,
	})
	require.NoError(t, err)
	assert.Contains(t, exported, "Exported to")
	assert.Contains(t, exported, ".html")
}

func TestHandleExport_WithTextBoundaries(t *testing.T) {
	result, err := HandleList(ListInput{Limit: 1})
	require.NoError(t, err)

	lines := strings.Split(result, "\n")
	if len(lines) < 3 {
		t.Skip("No sessions available")
	}

	line := lines[2]
	if !strings.HasPrefix(line, "- ") {
		t.Skip("Could not parse session list")
	}
	parts := strings.SplitN(line[2:], " | ", 2)
	sessionID := strings.TrimSpace(parts[0])

	// Export with a text boundary that won't match anything — should still work
	exported, err := HandleExport(ExportInput{
		SessionID: sessionID,
		Format:    "md",
		FromText:  "zzz_nonexistent_text_zzz",
	})
	require.NoError(t, err)
	assert.Contains(t, exported, "Exported to")
}

func TestApplyTextBoundaries_BothFromAndTo(t *testing.T) {
	msgs := []parser.Message{
		{TextContent: "before"},
		{TextContent: "from here"},
		{TextContent: "middle"},
		{TextContent: "to here"},
		{TextContent: "after"},
	}

	result := applyTextBoundaries(msgs, "from here", "to here")
	assert.Len(t, result, 3)
	assert.Equal(t, "from here", result[0].TextContent)
	assert.Equal(t, "to here", result[2].TextContent)
}
