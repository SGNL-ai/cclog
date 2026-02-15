package mcp

import (
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
	result, err := HandleList(ListInput{Limit: 5})
	require.NoError(t, err)
	assert.Contains(t, result, "Found")
	assert.Contains(t, result, "sessions")
	assert.Contains(t, result, "showing")
}

func TestHandleList_LimitCapsToAvailable(t *testing.T) {
	// With a very high limit, should still work and show available count
	result, err := HandleList(ListInput{Limit: 10000})
	require.NoError(t, err)
	assert.Contains(t, result, "Found")
}

func TestHandleList_DefaultLimit(t *testing.T) {
	// Zero limit should default to 20
	result, err := HandleList(ListInput{Limit: 0})
	require.NoError(t, err)
	assert.Contains(t, result, "Found")
}

func TestHandleList_WithProject(t *testing.T) {
	result, err := HandleList(ListInput{Project: "/nonexistent/path"})
	require.NoError(t, err)
	assert.Contains(t, result, "showing 0")
}

func TestHandleExport_RealSession_Markdown(t *testing.T) {
	// Find a real session — skip if none available
	sessions, err := session.Discover(session.DefaultClaudeDir())
	if err != nil || len(sessions) == 0 {
		t.Skip("No sessions available for integration test")
	}

	exported, err := HandleExport(ExportInput{
		SessionID: sessions[0].ID,
		Format:    "md",
		All:       true,
	})
	require.NoError(t, err)
	assert.Contains(t, exported, "Exported to")
	assert.Contains(t, exported, ".md")
	assert.Contains(t, exported, "messages")
	assert.Contains(t, exported, "bytes")
}

func TestHandleExport_RealSession_HTML(t *testing.T) {
	sessions, err := session.Discover(session.DefaultClaudeDir())
	if err != nil || len(sessions) == 0 {
		t.Skip("No sessions available for integration test")
	}

	exported, err := HandleExport(ExportInput{
		SessionID: sessions[0].ID,
		Format:    "html",
		All:       true,
	})
	require.NoError(t, err)
	assert.Contains(t, exported, "Exported to")
	assert.Contains(t, exported, ".html")
}

func TestHandleExport_DefaultFormatIsHTML(t *testing.T) {
	sessions, err := session.Discover(session.DefaultClaudeDir())
	if err != nil || len(sessions) == 0 {
		t.Skip("No sessions available for integration test")
	}

	exported, err := HandleExport(ExportInput{
		SessionID: sessions[0].ID,
		Format:    "", // should default to html
		All:       true,
	})
	require.NoError(t, err)
	assert.Contains(t, exported, ".html")
}

func TestHandleExport_WithTextBoundaries(t *testing.T) {
	sessions, err := session.Discover(session.DefaultClaudeDir())
	if err != nil || len(sessions) == 0 {
		t.Skip("No sessions available for integration test")
	}

	// Non-matching text boundary — should still export (returns all messages since no match)
	exported, err := HandleExport(ExportInput{
		SessionID: sessions[0].ID,
		Format:    "md",
		FromText:  "zzz_nonexistent_text_zzz",
	})
	require.NoError(t, err)
	assert.Contains(t, exported, "Exported to")
}

func TestApplyTextBoundaries_NoFilter(t *testing.T) {
	msgs := []parser.Message{
		{TextContent: "first"},
		{TextContent: "second"},
		{TextContent: "third"},
	}

	result := applyTextBoundaries(msgs, "", "")
	require.Len(t, result, 3)
	assert.Equal(t, "first", result[0].TextContent)
	assert.Equal(t, "third", result[2].TextContent)
}

func TestApplyTextBoundaries_FromText(t *testing.T) {
	msgs := []parser.Message{
		{TextContent: "first message"},
		{TextContent: "start here please"},
		{TextContent: "third message"},
	}

	result := applyTextBoundaries(msgs, "start here", "")
	require.Len(t, result, 2)
	assert.Equal(t, "start here please", result[0].TextContent)
	assert.Equal(t, "third message", result[1].TextContent)
}

func TestApplyTextBoundaries_ToText(t *testing.T) {
	msgs := []parser.Message{
		{TextContent: "first message"},
		{TextContent: "stop here please"},
		{TextContent: "third message"},
	}

	result := applyTextBoundaries(msgs, "", "stop here")
	require.Len(t, result, 2)
	assert.Equal(t, "first message", result[0].TextContent)
	assert.Equal(t, "stop here please", result[1].TextContent)
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
	require.Len(t, result, 3)
	assert.Equal(t, "from here", result[0].TextContent)
	assert.Equal(t, "middle", result[1].TextContent)
	assert.Equal(t, "to here", result[2].TextContent)
}

func TestApplyTextBoundaries_CaseInsensitive(t *testing.T) {
	msgs := []parser.Message{
		{TextContent: "first"},
		{TextContent: "START HERE"},
		{TextContent: "third"},
	}

	result := applyTextBoundaries(msgs, "start here", "")
	require.Len(t, result, 2)
	assert.Equal(t, "START HERE", result[0].TextContent)
}

func TestApplyTextBoundaries_NoMatch(t *testing.T) {
	msgs := []parser.Message{
		{TextContent: "first"},
		{TextContent: "second"},
	}

	// fromText doesn't match anything — should return from index 0
	result := applyTextBoundaries(msgs, "nonexistent", "")
	assert.Len(t, result, 2, "no match should return all messages")
}

func TestBuildTitle_Summary(t *testing.T) {
	sess := &session.SessionInfo{Summary: "My Summary", FirstPrompt: "prompt", Slug: "slug"}
	assert.Equal(t, "My Summary", buildTitle(sess), "summary takes precedence")
}

func TestBuildTitle_FirstPrompt(t *testing.T) {
	sess := &session.SessionInfo{FirstPrompt: "Hello world", Slug: "slug"}
	assert.Equal(t, "Hello world", buildTitle(sess), "firstPrompt used when no summary")
}

func TestBuildTitle_LongPromptTruncated(t *testing.T) {
	long := "This is a very long prompt that exceeds sixty characters in length and should be truncated"
	sess := &session.SessionInfo{FirstPrompt: long}
	title := buildTitle(sess)
	assert.True(t, len(title) <= 63, "truncated title should be at most 63 chars")
	assert.True(t, len(title) > 60, "truncated title should include ellipsis")
	assert.Contains(t, title, "...")
}

func TestBuildTitle_Slug(t *testing.T) {
	sess := &session.SessionInfo{Slug: "my-slug"}
	assert.Equal(t, "my-slug", buildTitle(sess))
}

func TestBuildTitle_Default(t *testing.T) {
	sess := &session.SessionInfo{}
	assert.Equal(t, "Claude Code Session", buildTitle(sess))
}
