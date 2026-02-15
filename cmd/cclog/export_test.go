package main

import (
	"strings"
	"testing"
	"time"

	"github.com/sgnl-ai/cclog/internal/parser"
	"github.com/sgnl-ai/cclog/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- pickSession ---

func TestPickSession_ValidSelection(t *testing.T) {
	claudeDir := session.DefaultClaudeDir()
	sessions, err := session.Discover(claudeDir)
	if err != nil || len(sessions) == 0 {
		t.Skip("No sessions available for test")
	}

	stdin := strings.NewReader("1\n")
	sess, err := pickSession(claudeDir, stdin)
	require.NoError(t, err)
	assert.Equal(t, sessions[0].ID, sess.ID)
}

func TestPickSession_InvalidSelection_NotANumber(t *testing.T) {
	claudeDir := session.DefaultClaudeDir()
	sessions, err := session.Discover(claudeDir)
	if err != nil || len(sessions) == 0 {
		t.Skip("No sessions available for test")
	}

	stdin := strings.NewReader("abc\n")
	_, err = pickSession(claudeDir, stdin)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid selection")
}

func TestPickSession_InvalidSelection_OutOfRange(t *testing.T) {
	claudeDir := session.DefaultClaudeDir()
	sessions, err := session.Discover(claudeDir)
	if err != nil || len(sessions) == 0 {
		t.Skip("No sessions available for test")
	}

	stdin := strings.NewReader("999\n")
	_, err = pickSession(claudeDir, stdin)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid selection")
}

func TestPickSession_InvalidSelection_Zero(t *testing.T) {
	claudeDir := session.DefaultClaudeDir()
	sessions, err := session.Discover(claudeDir)
	if err != nil || len(sessions) == 0 {
		t.Skip("No sessions available for test")
	}

	stdin := strings.NewReader("0\n")
	_, err = pickSession(claudeDir, stdin)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid selection")
}

func TestPickSession_NoSessions(t *testing.T) {
	stdin := strings.NewReader("1\n")
	_, err := pickSession("/nonexistent/path", stdin)
	require.Error(t, err)
}

// --- applyBoundarySelection ---

func TestApplyBoundarySelection_SingleBoundary(t *testing.T) {
	// With only 1 boundary (or fewer), should return messages unchanged
	msgs := []parser.Message{
		{Type: "user", Role: "user", TextContent: "hello", Timestamp: time.Now()},
	}

	stdin := strings.NewReader("1\n")
	result, err := applyBoundarySelection(msgs, stdin)
	require.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestApplyBoundarySelection_AllInput(t *testing.T) {
	now := time.Now()
	msgs := []parser.Message{
		{Type: "user", Role: "user", TextContent: "first", Timestamp: now},
		{Type: "user", Role: "user", TextContent: "second", Timestamp: now.Add(2 * time.Hour)},
	}

	stdin := strings.NewReader("all\n")
	result, err := applyBoundarySelection(msgs, stdin)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestApplyBoundarySelection_EmptyInput(t *testing.T) {
	now := time.Now()
	msgs := []parser.Message{
		{Type: "user", Role: "user", TextContent: "first", Timestamp: now},
		{Type: "user", Role: "user", TextContent: "second", Timestamp: now.Add(2 * time.Hour)},
	}

	stdin := strings.NewReader("\n")
	result, err := applyBoundarySelection(msgs, stdin)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestApplyBoundarySelection_SelectBoundary(t *testing.T) {
	now := time.Now()
	msgs := []parser.Message{
		{Type: "user", Role: "user", TextContent: "old work", Timestamp: now},
		{Type: "user", Role: "user", TextContent: "new work", Timestamp: now.Add(2 * time.Hour)},
	}

	stdin := strings.NewReader("2\n")
	result, err := applyBoundarySelection(msgs, stdin)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "new work", result[0].TextContent)
}

func TestApplyBoundarySelection_InvalidSelection(t *testing.T) {
	now := time.Now()
	msgs := []parser.Message{
		{Type: "user", Role: "user", TextContent: "first", Timestamp: now},
		{Type: "user", Role: "user", TextContent: "second", Timestamp: now.Add(2 * time.Hour)},
	}

	stdin := strings.NewReader("xyz\n")
	_, err := applyBoundarySelection(msgs, stdin)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid selection")
}

func TestApplyBoundarySelection_OutOfRangeSelection(t *testing.T) {
	now := time.Now()
	msgs := []parser.Message{
		{Type: "user", Role: "user", TextContent: "first", Timestamp: now},
		{Type: "user", Role: "user", TextContent: "second", Timestamp: now.Add(2 * time.Hour)},
	}

	stdin := strings.NewReader("99\n")
	_, err := applyBoundarySelection(msgs, stdin)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid selection")
}

// --- shortID ---

func TestShortID_Long(t *testing.T) {
	assert.Equal(t, "abcdefgh", shortID("abcdefgh-1234-5678"))
}

func TestShortID_Short(t *testing.T) {
	assert.Equal(t, "abc", shortID("abc"))
}

// --- formatListRow ---

func TestFormatListRow_Basic(t *testing.T) {
	s := session.SessionInfo{
		ID:           "abcdefgh-1234",
		FirstPrompt:  "hello world",
		Modified:     time.Date(2025, 2, 14, 10, 30, 0, 0, time.UTC),
		Project:      "/path/to/project",
		MessageCount: 42,
	}

	row := formatListRow(s)
	assert.Contains(t, row, "abcdefgh")
	assert.Contains(t, row, "2025-02-14 10:30")
	assert.Contains(t, row, "42")
	assert.Contains(t, row, "project")
	assert.Contains(t, row, "hello world")
}

func TestFormatListRow_LongPromptTruncated(t *testing.T) {
	s := session.SessionInfo{
		ID:          "abcdefgh-1234",
		FirstPrompt: "This is a very long prompt that exceeds fifty characters and should be truncated at some point",
		Modified:    time.Date(2025, 2, 14, 0, 0, 0, 0, time.UTC),
	}

	row := formatListRow(s)
	assert.Contains(t, row, "...")
}

func TestFormatListRow_FallsBackToSummary(t *testing.T) {
	s := session.SessionInfo{
		ID:       "abcdefgh-1234",
		Summary:  "summary text",
		Modified: time.Date(2025, 2, 14, 0, 0, 0, 0, time.UTC),
	}

	row := formatListRow(s)
	assert.Contains(t, row, "summary text")
}

func TestFormatListRow_EmptyProject(t *testing.T) {
	s := session.SessionInfo{
		ID:       "abcdefgh-1234",
		Modified: time.Date(2025, 2, 14, 0, 0, 0, 0, time.UTC),
	}

	row := formatListRow(s)
	assert.Contains(t, row, "-")
}

// --- rootCmd ---

func TestRootCmd_HasSubcommands(t *testing.T) {
	cmd := rootCmd()
	assert.NotNil(t, cmd)

	names := make([]string, 0)
	for _, sub := range cmd.Commands() {
		names = append(names, sub.Name())
	}
	assert.Contains(t, names, "export")
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "setup")
	assert.Contains(t, names, "serve")
}

func TestRootCmd_Help(t *testing.T) {
	cmd := rootCmd()
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestExportCmd_Flags(t *testing.T) {
	cmd := exportCmd()
	assert.NotNil(t, cmd.Flags().Lookup("session"))
	assert.NotNil(t, cmd.Flags().Lookup("all"))
	assert.NotNil(t, cmd.Flags().Lookup("format"))
	assert.NotNil(t, cmd.Flags().Lookup("gist"))
	assert.NotNil(t, cmd.Flags().Lookup("public"))
}

func TestListCmd_Flags(t *testing.T) {
	cmd := listCmd()
	assert.NotNil(t, cmd.Flags().Lookup("project"))
}

// --- runExport ---

func TestRunExport_WithSessionID(t *testing.T) {
	sessions, err := session.Discover(session.DefaultClaudeDir())
	if err != nil || len(sessions) == 0 {
		t.Skip("No sessions available for integration test")
	}

	stdin := strings.NewReader("")
	err = runExport(exportOpts{
		sessionID: sessions[0].ID,
		all:       true,
		format:    "md",
	}, stdin)
	require.NoError(t, err)
}

func TestRunExport_WithSessionID_HTML(t *testing.T) {
	sessions, err := session.Discover(session.DefaultClaudeDir())
	if err != nil || len(sessions) == 0 {
		t.Skip("No sessions available for integration test")
	}

	stdin := strings.NewReader("")
	err = runExport(exportOpts{
		sessionID: sessions[0].ID,
		all:       true,
		format:    "html",
	}, stdin)
	require.NoError(t, err)
}

func TestRunExport_NonexistentSession(t *testing.T) {
	stdin := strings.NewReader("")
	err := runExport(exportOpts{
		sessionID: "nonexistent-session-id",
		all:       true,
	}, stdin)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunExport_InteractivePickSession(t *testing.T) {
	sessions, err := session.Discover(session.DefaultClaudeDir())
	if err != nil || len(sessions) == 0 {
		t.Skip("No sessions available for integration test")
	}

	// Pick first session interactively
	stdin := strings.NewReader("1\n")
	err = runExport(exportOpts{
		all:    true,
		format: "md",
	}, stdin)
	require.NoError(t, err)
}

func TestRunExport_InteractiveWithBoundarySelection(t *testing.T) {
	sessions, err := session.Discover(session.DefaultClaudeDir())
	if err != nil || len(sessions) == 0 {
		t.Skip("No sessions available for integration test")
	}

	// Pick first session + "all" for boundary selection
	stdin := strings.NewReader("1\nall\n")
	err = runExport(exportOpts{
		all:    false, // triggers boundary detection
		format: "md",
	}, stdin)
	require.NoError(t, err)
}

// --- runList ---

func TestRunList_AllSessions(t *testing.T) {
	err := runList(listOpts{})
	require.NoError(t, err)
}

func TestRunList_WithProject(t *testing.T) {
	err := runList(listOpts{project: "/nonexistent/path"})
	// Should not error, just shows "No sessions found."
	require.NoError(t, err)
}

// --- serveCmd / setupCmd construction ---

func TestServeCmd(t *testing.T) {
	cmd := serveCmd()
	assert.Equal(t, "serve", cmd.Name())
}

func TestSetupCmd(t *testing.T) {
	cmd := setupCmd()
	assert.Equal(t, "setup", cmd.Name())
}
