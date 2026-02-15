package export

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sgnl-ai/cclog/internal/parser"
	"github.com/sgnl-ai/cclog/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- BuildTitle ---

func TestBuildTitle_Summary(t *testing.T) {
	sess := &session.SessionInfo{Summary: "My Summary", FirstPrompt: "prompt", Slug: "slug"}
	assert.Equal(t, "My Summary", BuildTitle(sess))
}

func TestBuildTitle_FirstPrompt(t *testing.T) {
	sess := &session.SessionInfo{FirstPrompt: "Hello world", Slug: "slug"}
	assert.Equal(t, "Hello world", BuildTitle(sess))
}

func TestBuildTitle_LongPromptTruncated(t *testing.T) {
	long := "This is a very long prompt that exceeds sixty characters in length and should be truncated"
	sess := &session.SessionInfo{FirstPrompt: long}
	title := BuildTitle(sess)
	assert.LessOrEqual(t, len(title), MaxPromptLen+3)
	assert.Greater(t, len(title), MaxPromptLen)
	assert.Contains(t, title, "...")
}

func TestBuildTitle_Slug(t *testing.T) {
	sess := &session.SessionInfo{Slug: "my-slug"}
	assert.Equal(t, "my-slug", BuildTitle(sess))
}

func TestBuildTitle_Default(t *testing.T) {
	sess := &session.SessionInfo{}
	assert.Equal(t, "Claude Code Session", BuildTitle(sess))
}

// --- DefaultOutputDir ---

func TestDefaultOutputDir(t *testing.T) {
	dir, err := DefaultOutputDir()
	require.NoError(t, err)
	home, _ := os.UserHomeDir()
	assert.Equal(t, filepath.Join(home, "cclog"), dir)
}

// --- outputPath ---

func TestOutputPath_WithSlug(t *testing.T) {
	mod := time.Date(2025, 2, 14, 0, 0, 0, 0, time.UTC)
	path := outputPath("/tmp/out", "my-session", "abcdefgh", mod, ".html")
	assert.Equal(t, "/tmp/out/my-session-2025-02-14.html", path)
}

func TestOutputPath_WithoutSlug(t *testing.T) {
	mod := time.Date(2025, 2, 14, 0, 0, 0, 0, time.UTC)
	path := outputPath("/tmp/out", "", "abcdefgh12345", mod, ".md")
	assert.Equal(t, "/tmp/out/abcdefgh-2025-02-14.md", path)
}

func TestOutputPath_ZeroTime(t *testing.T) {
	path := outputPath("/tmp/out", "slug", "abcdefgh", time.Time{}, ".html")
	today := time.Now().Format("2006-01-02")
	assert.Equal(t, "/tmp/out/slug-"+today+".html", path)
}

// --- ShortID ---

func TestShortID_Long(t *testing.T) {
	assert.Equal(t, "abcdefgh", ShortID("abcdefgh-1234-5678"))
}

func TestShortID_Short(t *testing.T) {
	assert.Equal(t, "abc", ShortID("abc"))
}

func TestShortID_ExactlyEight(t *testing.T) {
	assert.Equal(t, "abcdefgh", ShortID("abcdefgh"))
}

// --- TruncatePrompt ---

func TestTruncatePrompt_Short(t *testing.T) {
	assert.Equal(t, "hello", TruncatePrompt("hello", "", 60))
}

func TestTruncatePrompt_Long(t *testing.T) {
	long := "This is a very long prompt that exceeds sixty characters in length"
	result := TruncatePrompt(long, "", 60)
	assert.Equal(t, long[:60]+"...", result)
}

func TestTruncatePrompt_FallsBackToSummary(t *testing.T) {
	assert.Equal(t, "summary", TruncatePrompt("", "summary", 60))
}

func TestTruncatePrompt_BothEmpty(t *testing.T) {
	assert.Equal(t, "", TruncatePrompt("", "", 60))
}

// --- ApplyTextBoundaries ---

func TestApplyTextBoundaries_NoFilter(t *testing.T) {
	msgs := []parser.Message{
		{TextContent: "first"},
		{TextContent: "second"},
		{TextContent: "third"},
	}

	result := ApplyTextBoundaries(msgs, "", "")
	require.Len(t, result, 3)
}

func TestApplyTextBoundaries_FromText(t *testing.T) {
	msgs := []parser.Message{
		{TextContent: "first message"},
		{TextContent: "start here please"},
		{TextContent: "third message"},
	}

	result := ApplyTextBoundaries(msgs, "start here", "")
	require.Len(t, result, 2)
	assert.Equal(t, "start here please", result[0].TextContent)
}

func TestApplyTextBoundaries_ToText(t *testing.T) {
	msgs := []parser.Message{
		{TextContent: "first message"},
		{TextContent: "stop here please"},
		{TextContent: "third message"},
	}

	result := ApplyTextBoundaries(msgs, "", "stop here")
	require.Len(t, result, 2)
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

	result := ApplyTextBoundaries(msgs, "from here", "to here")
	require.Len(t, result, 3)
	assert.Equal(t, "from here", result[0].TextContent)
	assert.Equal(t, "to here", result[2].TextContent)
}

func TestApplyTextBoundaries_CaseInsensitive(t *testing.T) {
	msgs := []parser.Message{
		{TextContent: "first"},
		{TextContent: "START HERE"},
		{TextContent: "third"},
	}

	result := ApplyTextBoundaries(msgs, "start here", "")
	require.Len(t, result, 2)
	assert.Equal(t, "START HERE", result[0].TextContent)
}

func TestApplyTextBoundaries_NoMatch(t *testing.T) {
	msgs := []parser.Message{
		{TextContent: "first"},
		{TextContent: "second"},
	}

	result := ApplyTextBoundaries(msgs, "nonexistent", "")
	assert.Len(t, result, 2, "no match should return all messages")
}

// --- RedactMessages ---

func TestRedactMessages_NoSecrets(t *testing.T) {
	msgs := []parser.Message{
		{TextContent: "plain text"},
		{TextContent: "another message"},
	}

	result, err := RedactMessages(msgs)
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "plain text", result[0].TextContent)
	assert.Equal(t, "another message", result[1].TextContent)
}

func TestRedactMessages_EmptySlice(t *testing.T) {
	result, err := RedactMessages(nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestRedactMessages_DoesNotMutateInput(t *testing.T) {
	original := []parser.Message{
		{TextContent: "hello"},
		{TextContent: "world"},
	}

	result, err := RedactMessages(original)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	// Verify original is unchanged (same values, different slice).
	assert.Equal(t, "hello", original[0].TextContent)
	assert.Equal(t, "world", original[1].TextContent)
}

// --- FormatSessionList ---

func TestFormatSessionList_Empty(t *testing.T) {
	result := FormatSessionList(nil, 20)
	assert.Contains(t, result, "Found 0 sessions (showing 0)")
}

func TestFormatSessionList_DefaultLimit(t *testing.T) {
	sessions := makeSessions(5)
	result := FormatSessionList(sessions, 0)
	assert.Contains(t, result, "showing 5")
}

func TestFormatSessionList_LimitLargerThanAvailable(t *testing.T) {
	sessions := makeSessions(3)
	result := FormatSessionList(sessions, 100)
	assert.Contains(t, result, "showing 3")
}

func TestFormatSessionList_LimitSmallerThanAvailable(t *testing.T) {
	sessions := makeSessions(10)
	result := FormatSessionList(sessions, 3)
	assert.Contains(t, result, "Found 10 sessions (showing 3)")
}

func TestFormatSessionList_IncludesSessionData(t *testing.T) {
	sessions := []session.SessionInfo{{
		ID:          "sess-001",
		FirstPrompt: "hello world",
		Modified:    time.Date(2025, 2, 14, 10, 30, 0, 0, time.UTC),
		Project:     "/path/to/project",
	}}

	result := FormatSessionList(sessions, 20)
	assert.Contains(t, result, "sess-001")
	assert.Contains(t, result, "2025-02-14 10:30")
	assert.Contains(t, result, "project")
	assert.Contains(t, result, "hello world")
}

func TestFormatSessionList_EmptyProjectShowsDash(t *testing.T) {
	sessions := []session.SessionInfo{{
		ID:       "sess-001",
		Modified: time.Date(2025, 2, 14, 0, 0, 0, 0, time.UTC),
	}}

	result := FormatSessionList(sessions, 20)
	assert.Contains(t, result, "| - |")
}

// --- ListSessions (hermetic) ---

func TestListSessions_WithNonexistentProject(t *testing.T) {
	claudeDir, _ := testSession(t)
	sessions, err := ListSessions(claudeDir, "/nonexistent/path")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestListSessions_ReturnsResults(t *testing.T) {
	claudeDir, _ := testSession(t)
	sessions, err := ListSessions(claudeDir, "")
	require.NoError(t, err)
	assert.NotEmpty(t, sessions)
}

// --- ExportSession (hermetic) ---

func TestExportSession_MissingSessionID(t *testing.T) {
	_, err := ExportSession(ExportOpts{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session_id is required")
}

func TestExportSession_NonexistentSession(t *testing.T) {
	claudeDir, _ := testSession(t)
	_, err := ExportSession(ExportOpts{SessionID: "nonexistent", ClaudeDir: claudeDir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestExportSession_HTML(t *testing.T) {
	claudeDir, sessionID := testSession(t)
	outDir := t.TempDir()

	result, err := ExportSession(ExportOpts{
		SessionID: sessionID,
		Format:    "html",
		All:       true,
		ClaudeDir: claudeDir,
		OutputDir: outDir,
	})
	require.NoError(t, err)
	assert.Contains(t, result.OutputPath, ".html")
	assert.Greater(t, result.MessageCount, 0)
	assert.Greater(t, result.ByteCount, 0)

	// Verify file was written.
	_, err = os.Stat(result.OutputPath)
	require.NoError(t, err)
}

func TestExportSession_Markdown(t *testing.T) {
	claudeDir, sessionID := testSession(t)
	outDir := t.TempDir()

	result, err := ExportSession(ExportOpts{
		SessionID: sessionID,
		Format:    "md",
		All:       true,
		ClaudeDir: claudeDir,
		OutputDir: outDir,
	})
	require.NoError(t, err)
	assert.Contains(t, result.OutputPath, ".md")
	assert.Greater(t, result.MessageCount, 0)
}

func TestExportSession_DefaultFormatIsHTML(t *testing.T) {
	claudeDir, sessionID := testSession(t)
	outDir := t.TempDir()

	result, err := ExportSession(ExportOpts{
		SessionID: sessionID,
		All:       true,
		ClaudeDir: claudeDir,
		OutputDir: outDir,
	})
	require.NoError(t, err)
	assert.Contains(t, result.OutputPath, ".html")
}

func TestExportSession_WithTextBoundaries(t *testing.T) {
	claudeDir, sessionID := testSession(t)
	outDir := t.TempDir()

	result, err := ExportSession(ExportOpts{
		SessionID: sessionID,
		Format:    "md",
		FromText:  "error is in",
		ClaudeDir: claudeDir,
		OutputDir: outDir,
	})
	require.NoError(t, err)
	assert.Contains(t, result.OutputPath, ".md")
	// Should have fewer messages since we filtered.
	assert.Greater(t, result.MessageCount, 0)
}

func TestExportSession_PrefixMatch(t *testing.T) {
	claudeDir, _ := testSession(t)
	outDir := t.TempDir()

	result, err := ExportSession(ExportOpts{
		SessionID: "deadbeef",
		All:       true,
		ClaudeDir: claudeDir,
		OutputDir: outDir,
	})
	require.NoError(t, err)
	assert.Contains(t, result.OutputPath, ".html")
}

// --- helpers ---

func makeSessions(n int) []session.SessionInfo {
	sessions := make([]session.SessionInfo, n)
	for i := range sessions {
		sessions[i] = session.SessionInfo{
			ID:          fmt.Sprintf("sess-%03d", i),
			FirstPrompt: fmt.Sprintf("prompt %d", i),
			Modified:    time.Date(2025, 2, 14, 0, 0, 0, 0, time.UTC),
		}
	}
	return sessions
}
