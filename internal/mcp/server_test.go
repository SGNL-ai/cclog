package mcp

import (
	"testing"

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
	result, err := HandleList(ListInput{Limit: 10000})
	require.NoError(t, err)
	assert.Contains(t, result, "Found")
}

func TestHandleList_DefaultLimit(t *testing.T) {
	result, err := HandleList(ListInput{Limit: 0})
	require.NoError(t, err)
	assert.Contains(t, result, "Found")
}

func TestHandleList_WithProject(t *testing.T) {
	result, err := HandleList(ListInput{Project: "/nonexistent/path"})
	require.NoError(t, err)
	assert.Contains(t, result, "showing 0")
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
	assert.Contains(t, exported, "messages")
	assert.Contains(t, exported, "bytes")
}

func TestHandleExport_RealSession_Markdown(t *testing.T) {
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
}

func TestHandleExport_DefaultFormatIsHTML(t *testing.T) {
	sessions, err := session.Discover(session.DefaultClaudeDir())
	if err != nil || len(sessions) == 0 {
		t.Skip("No sessions available for integration test")
	}

	exported, err := HandleExport(ExportInput{
		SessionID: sessions[0].ID,
		Format:    "",
		All:       true,
	})
	require.NoError(t, err)
	assert.Contains(t, exported, ".html")
}
