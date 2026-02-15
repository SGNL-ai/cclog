package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sgnl-ai/cclog/internal/export"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSession creates a hermetic test session. Returns (claudeDir, sessionID).
func testSession(t *testing.T) (string, string) {
	t.Helper()

	claudeDir := t.TempDir()
	projectDir := filepath.Join(claudeDir, "-Users-test-projects-myapp")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	sessionID := "deadbeef-1234-5678-abcd-ef0123456789"
	jsonl := `{"type":"user","sessionId":"` + sessionID + `","uuid":"u1","parentUuid":null,"slug":"test-session","timestamp":"2026-02-14T10:00:00.000Z","message":{"role":"user","content":"Hello, please help me fix a bug"}}
{"type":"assistant","sessionId":"` + sessionID + `","uuid":"u2","parentUuid":"u1","timestamp":"2026-02-14T10:00:05.000Z","message":{"role":"assistant","content":[{"type":"text","text":"Sure, I can help."}]}}
`
	jsonlPath := filepath.Join(projectDir, sessionID+".jsonl")
	require.NoError(t, os.WriteFile(jsonlPath, []byte(jsonl), 0o644))

	type entry struct {
		SessionID    string `json:"sessionId"`
		FullPath     string `json:"fullPath"`
		FirstPrompt  string `json:"firstPrompt"`
		MessageCount int    `json:"messageCount"`
		Created      string `json:"created"`
		Modified     string `json:"modified"`
		ProjectPath  string `json:"projectPath"`
	}
	idx := struct {
		Version int     `json:"version"`
		Entries []entry `json:"entries"`
	}{
		Version: 1,
		Entries: []entry{{
			SessionID:    sessionID,
			FullPath:     jsonlPath,
			FirstPrompt:  "Hello, please help me fix a bug",
			MessageCount: 2,
			Created:      "2026-02-14T10:00:00.000Z",
			Modified:     "2026-02-14T10:00:05.000Z",
			ProjectPath:  "/Users/test/projects/myapp",
		}},
	}
	data, _ := json.Marshal(idx)
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "sessions-index.json"), data, 0o644))

	return claudeDir, sessionID
}

// --- pickSession ---

func TestPickSession_ValidSelection(t *testing.T) {
	claudeDir, sessionID := testSession(t)

	stdin := strings.NewReader("1\n")
	sess, err := pickSession(claudeDir, stdin)
	require.NoError(t, err)
	assert.Equal(t, sessionID, sess.ID)
}

func TestPickSession_InvalidSelection_NotANumber(t *testing.T) {
	claudeDir, _ := testSession(t)

	stdin := strings.NewReader("abc\n")
	_, err := pickSession(claudeDir, stdin)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid selection")
}

func TestPickSession_InvalidSelection_OutOfRange(t *testing.T) {
	claudeDir, _ := testSession(t)

	stdin := strings.NewReader("999\n")
	_, err := pickSession(claudeDir, stdin)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid selection")
}

func TestPickSession_InvalidSelection_Zero(t *testing.T) {
	claudeDir, _ := testSession(t)

	stdin := strings.NewReader("0\n")
	_, err := pickSession(claudeDir, stdin)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid selection")
}

func TestPickSession_NoSessions(t *testing.T) {
	stdin := strings.NewReader("1\n")
	_, err := pickSession(t.TempDir(), stdin)
	require.Error(t, err)
}

// --- rootCmd ---

func TestRootCmd_HasSubcommands(t *testing.T) {
	cmd := rootCmd()
	assert.NotNil(t, cmd)

	var names []string
	for _, sub := range cmd.Commands() {
		names = append(names, sub.Name())
	}
	assert.Contains(t, names, "export")
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "setup")
	assert.Contains(t, names, "serve")
}

func TestRootCmd_Version(t *testing.T) {
	cmd := rootCmd()
	assert.Equal(t, version, cmd.Version)
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
	assert.NotNil(t, cmd.Flags().Lookup("from-text"))
	assert.NotNil(t, cmd.Flags().Lookup("to-text"))
	assert.NotNil(t, cmd.Flags().Lookup("gist"))
	assert.NotNil(t, cmd.Flags().Lookup("public"))
}

func TestListCmd_Flags(t *testing.T) {
	cmd := listCmd()
	assert.NotNil(t, cmd.Flags().Lookup("project"))
}

// --- runExport (hermetic) ---

func TestRunExport_WithSessionID(t *testing.T) {
	claudeDir, sessionID := testSession(t)
	outDir := t.TempDir()

	// Temporarily override DefaultClaudeDir — we pass sessionID directly
	// and ExportSession uses ClaudeDir if set (but runExport doesn't expose it).
	// We use the ExportOpts.ClaudeDir via export package directly.
	result, err := export.ExportSession(export.ExportOpts{
		SessionID: sessionID,
		Format:    "md",
		All:       true,
		ClaudeDir: claudeDir,
		OutputDir: outDir,
	})
	require.NoError(t, err)
	assert.Contains(t, result.OutputPath, ".md")
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

// --- runList ---

func TestRunList_NoSessions(t *testing.T) {
	// With a nonexistent project, should print "No sessions found."
	err := runList(listOpts{project: "/nonexistent/path"})
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

// --- TruncatePrompt via export ---

func TestTruncatePrompt_UsedConsistently(t *testing.T) {
	long := "This is a very long prompt that exceeds the standard limit and should be truncated"
	result := export.TruncatePrompt(long, "", export.MaxPromptLen)
	assert.Contains(t, result, "...")
	assert.LessOrEqual(t, len(result), export.MaxPromptLen+3)
}
