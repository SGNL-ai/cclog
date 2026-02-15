package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	jsonl := `{"type":"user","sessionId":"` + sessionID + `","uuid":"u1","parentUuid":null,"slug":"test-session","timestamp":"2026-02-14T10:00:00.000Z","message":{"role":"user","content":"Hello world"}}
{"type":"assistant","sessionId":"` + sessionID + `","uuid":"u2","parentUuid":"u1","timestamp":"2026-02-14T10:00:05.000Z","message":{"role":"assistant","content":[{"type":"text","text":"Hi there!"}]}}
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
			FirstPrompt:  "Hello world",
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
	claudeDir, _ := testSession(t)
	_, err := handleExportWithDir(ExportInput{SessionID: "nonexistent"}, claudeDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestHandleList_ReturnsResults(t *testing.T) {
	claudeDir, _ := testSession(t)
	result, err := handleListWithDir(ListInput{Limit: 5}, claudeDir)
	require.NoError(t, err)
	assert.Contains(t, result, "Found")
	assert.Contains(t, result, "sessions")
}

func TestHandleList_DefaultLimit(t *testing.T) {
	claudeDir, _ := testSession(t)
	result, err := handleListWithDir(ListInput{Limit: 0}, claudeDir)
	require.NoError(t, err)
	assert.Contains(t, result, "Found")
}

func TestHandleList_WithProject(t *testing.T) {
	claudeDir, _ := testSession(t)
	result, err := handleListWithDir(ListInput{Project: "/nonexistent/path"}, claudeDir)
	require.NoError(t, err)
	assert.Contains(t, result, "showing 0")
}

func TestHandleExport_HTML(t *testing.T) {
	claudeDir, sessionID := testSession(t)
	outDir := t.TempDir()

	exported, err := handleExportFull(ExportInput{
		SessionID: sessionID,
		Format:    "html",
		All:       true,
	}, claudeDir, outDir)
	require.NoError(t, err)
	assert.Contains(t, exported, "Exported to")
	assert.Contains(t, exported, ".html")
	assert.Contains(t, exported, "messages")
}

func TestHandleExport_Markdown(t *testing.T) {
	claudeDir, sessionID := testSession(t)
	outDir := t.TempDir()

	exported, err := handleExportFull(ExportInput{
		SessionID: sessionID,
		Format:    "md",
		All:       true,
	}, claudeDir, outDir)
	require.NoError(t, err)
	assert.Contains(t, exported, ".md")
}

func TestHandleExport_DefaultFormatIsHTML(t *testing.T) {
	claudeDir, sessionID := testSession(t)
	outDir := t.TempDir()

	exported, err := handleExportFull(ExportInput{
		SessionID: sessionID,
		Format:    "",
		All:       true,
	}, claudeDir, outDir)
	require.NoError(t, err)
	assert.Contains(t, exported, ".html")
}

// handleExportWithDir is a test helper that uses an injected claudeDir.
func handleExportWithDir(input ExportInput, claudeDir string) (string, error) {
	return handleExportFull(input, claudeDir, "")
}

// handleExportFull is a test helper with injected claudeDir and outputDir.
func handleExportFull(input ExportInput, claudeDir, outputDir string) (string, error) {
	if input.SessionID == "" {
		return "", nil
	}

	result, err := export.ExportSession(export.ExportOpts{
		SessionID: input.SessionID,
		Format:    input.Format,
		All:       input.All,
		FromText:  input.FromText,
		ToText:    input.ToText,
		Gist:      input.Gist,
		ClaudeDir: claudeDir,
		OutputDir: outputDir,
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Exported to %s (%d messages, %d bytes)", result.OutputPath, result.MessageCount, result.ByteCount), nil
}

// handleListWithDir is a test helper with injected claudeDir.
func handleListWithDir(input ListInput, claudeDir string) (string, error) {
	sessions, err := export.ListSessions(claudeDir, input.Project)
	if err != nil {
		return "", err
	}
	return export.FormatSessionList(sessions, input.Limit), nil
}
