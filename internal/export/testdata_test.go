package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// testSession creates a temp directory with a synthetic JSONL session file
// and sessions-index.json. Returns (claudeDir, sessionID, cleanup).
func testSession(t *testing.T) (string, string) {
	t.Helper()

	claudeDir := t.TempDir()
	projectDir := filepath.Join(claudeDir, "-Users-test-projects-myapp")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	sessionID := "deadbeef-1234-5678-abcd-ef0123456789"

	// Write a JSONL session file with realistic content.
	jsonl := `{"type":"user","sessionId":"` + sessionID + `","uuid":"u1","parentUuid":null,"slug":"test-session","timestamp":"2026-02-14T10:00:00.000Z","message":{"role":"user","content":"Hello, please help me fix a bug"}}
{"type":"assistant","sessionId":"` + sessionID + `","uuid":"u2","parentUuid":"u1","timestamp":"2026-02-14T10:00:05.000Z","message":{"role":"assistant","content":[{"type":"text","text":"Sure, I can help with that. Let me look at the code."}]}}
{"type":"user","sessionId":"` + sessionID + `","uuid":"u3","parentUuid":"u2","timestamp":"2026-02-14T10:01:00.000Z","message":{"role":"user","content":"The error is in main.go line 42"}}
{"type":"assistant","sessionId":"` + sessionID + `","uuid":"u4","parentUuid":"u3","timestamp":"2026-02-14T10:01:10.000Z","message":{"role":"assistant","content":[{"type":"text","text":"I see the issue. The variable is not initialized before use."}]}}
`

	jsonlPath := filepath.Join(projectDir, sessionID+".jsonl")
	require.NoError(t, os.WriteFile(jsonlPath, []byte(jsonl), 0o644))

	// Write sessions-index.json.
	type indexEntry struct {
		SessionID    string `json:"sessionId"`
		FullPath     string `json:"fullPath"`
		FirstPrompt  string `json:"firstPrompt"`
		Summary      string `json:"summary"`
		MessageCount int    `json:"messageCount"`
		Created      string `json:"created"`
		Modified     string `json:"modified"`
		ProjectPath  string `json:"projectPath"`
	}
	type sessionsIndex struct {
		Version int          `json:"version"`
		Entries []indexEntry `json:"entries"`
	}

	idx := sessionsIndex{
		Version: 1,
		Entries: []indexEntry{{
			SessionID:    sessionID,
			FullPath:     jsonlPath,
			FirstPrompt:  "Hello, please help me fix a bug",
			Summary:      "Bug fix session",
			MessageCount: 4,
			Created:      "2026-02-14T10:00:00.000Z",
			Modified:     "2026-02-14T10:01:10.000Z",
			ProjectPath:  "/Users/test/projects/myapp",
		}},
	}
	data, _ := json.Marshal(idx)
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "sessions-index.json"), data, 0o644))

	return claudeDir, sessionID
}
