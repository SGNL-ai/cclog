package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscover_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	sessions, err := Discover(dir)
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestDiscover_WithSessionsIndex(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "-Users-test-projects-myapp")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	idx := sessionsIndex{
		Version: 1,
		Entries: []indexEntry{
			{
				SessionID:    "sess-123",
				FullPath:     filepath.Join(projectDir, "sess-123.jsonl"),
				FirstPrompt:  "Fix the bug",
				Summary:      "Bug fix session",
				MessageCount: 42,
				Created:      "2026-02-13T07:00:00.000Z",
				Modified:     "2026-02-13T08:00:00.000Z",
				ProjectPath:  "/Users/test/projects/myapp",
			},
		},
		OriginalPath: "/Users/test/projects/myapp",
	}
	data, _ := json.Marshal(idx)
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "sessions-index.json"), data, 0o644))

	// Create dummy JSONL so discoverFromJSONL doesn't also pick it up
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "sess-123.jsonl"), []byte{}, 0o644))

	sessions, err := Discover(dir)
	require.NoError(t, err)
	require.Len(t, sessions, 1)

	s := sessions[0]
	assert.Equal(t, "sess-123", s.ID)
	assert.Equal(t, "/Users/test/projects/myapp", s.Project)
	assert.Equal(t, "Fix the bug", s.FirstPrompt)
	assert.Equal(t, "Bug fix session", s.Summary)
	assert.Equal(t, 42, s.MessageCount)
}

func TestDiscover_WithJSONLOnly(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "-Users-test-projects-myapp")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	// Write a minimal JSONL file
	line := `{"type":"user","sessionId":"sess-456","uuid":"u1","slug":"test-slug","timestamp":"2026-02-13T07:12:33.727Z","message":{"role":"user","content":"Hello world"}}` + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "sess-456.jsonl"), []byte(line), 0o644))

	sessions, err := Discover(dir)
	require.NoError(t, err)
	require.Len(t, sessions, 1)

	s := sessions[0]
	assert.Equal(t, "sess-456", s.ID)
	assert.Equal(t, "test-slug", s.Slug)
	assert.Equal(t, "Hello world", s.FirstPrompt)
}

func TestDiscover_SortsByModifiedDesc(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "-Users-test-projects-myapp")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	idx := sessionsIndex{
		Version: 1,
		Entries: []indexEntry{
			{
				SessionID: "old-sess",
				FullPath:  filepath.Join(projectDir, "old-sess.jsonl"),
				Created:   "2026-02-10T07:00:00.000Z",
				Modified:  "2026-02-10T08:00:00.000Z",
			},
			{
				SessionID: "new-sess",
				FullPath:  filepath.Join(projectDir, "new-sess.jsonl"),
				Created:   "2026-02-13T07:00:00.000Z",
				Modified:  "2026-02-13T08:00:00.000Z",
			},
		},
	}
	data, _ := json.Marshal(idx)
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "sessions-index.json"), data, 0o644))

	// Create dummy files
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "old-sess.jsonl"), []byte{}, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "new-sess.jsonl"), []byte{}, 0o644))

	sessions, err := Discover(dir)
	require.NoError(t, err)
	require.Len(t, sessions, 2)
	assert.Equal(t, "new-sess", sessions[0].ID)
	assert.Equal(t, "old-sess", sessions[1].ID)
}

func TestDiscoverForProject(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "-Users-test-projects-myapp")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	idx := sessionsIndex{
		Version: 1,
		Entries: []indexEntry{
			{SessionID: "s1", FullPath: filepath.Join(projectDir, "s1.jsonl"), ProjectPath: "/Users/test/projects/myapp", Modified: "2026-02-13T08:00:00.000Z"},
			{SessionID: "s2", FullPath: filepath.Join(projectDir, "s2.jsonl"), ProjectPath: "/Users/test/projects/other", Modified: "2026-02-13T08:00:00.000Z"},
		},
	}
	data, _ := json.Marshal(idx)
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "sessions-index.json"), data, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "s1.jsonl"), []byte{}, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "s2.jsonl"), []byte{}, 0o644))

	sessions, err := DiscoverForProject(dir, "/Users/test/projects/myapp")
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "s1", sessions[0].ID)
}

func TestFindSession(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "-Users-test-projects-myapp")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	idx := sessionsIndex{
		Version: 1,
		Entries: []indexEntry{
			{SessionID: "target", FullPath: filepath.Join(projectDir, "target.jsonl"), FirstPrompt: "found me", Modified: "2026-02-13T08:00:00.000Z"},
		},
	}
	data, _ := json.Marshal(idx)
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "sessions-index.json"), data, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "target.jsonl"), []byte{}, 0o644))

	s, err := FindSession(dir, "target")
	require.NoError(t, err)
	assert.Equal(t, "found me", s.FirstPrompt)
}

func TestFindSession_NotFound(t *testing.T) {
	dir := t.TempDir()

	_, err := FindSession(dir, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestFindSession_PrefixMatch(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "-Users-test-projects-myapp")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	fullID := "abcdef12-3456-7890"
	idx := sessionsIndex{
		Version: 1,
		Entries: []indexEntry{
			{SessionID: fullID, FullPath: filepath.Join(projectDir, fullID+".jsonl"), FirstPrompt: "found by prefix", Modified: "2026-02-13T08:00:00.000Z"},
		},
	}
	data, _ := json.Marshal(idx)
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "sessions-index.json"), data, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, fullID+".jsonl"), []byte{}, 0o644))

	s, err := FindSession(dir, "abcdef12")
	require.NoError(t, err)
	assert.Equal(t, "found by prefix", s.FirstPrompt)
	assert.Equal(t, fullID, s.ID)
}

func TestFindSession_AmbiguousPrefix(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "-Users-test-projects-myapp")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	idx := sessionsIndex{
		Version: 1,
		Entries: []indexEntry{
			{SessionID: "abc-111", FullPath: filepath.Join(projectDir, "abc-111.jsonl"), Modified: "2026-02-13T08:00:00.000Z"},
			{SessionID: "abc-222", FullPath: filepath.Join(projectDir, "abc-222.jsonl"), Modified: "2026-02-13T09:00:00.000Z"},
		},
	}
	data, _ := json.Marshal(idx)
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "sessions-index.json"), data, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "abc-111.jsonl"), []byte{}, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "abc-222.jsonl"), []byte{}, 0o644))

	_, err := FindSession(dir, "abc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous")
	assert.Contains(t, err.Error(), "2 matches")
}

func TestDiscover_NonexistentDir(t *testing.T) {
	_, err := Discover("/nonexistent/path/that/does/not/exist")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read claude dir")
}

func TestIsInternalPrompt(t *testing.T) {
	assert.True(t, isInternalPrompt("[Request interrupted by user for tool use]"))
	assert.True(t, isInternalPrompt("<local-command-caveat>Caveat: The messages below"))
	assert.True(t, isInternalPrompt("<command-name>/fast</command-name>"))
	assert.False(t, isInternalPrompt("Hello, please help me fix a bug"))
	assert.False(t, isInternalPrompt(""))
}

func TestDecodeProjectPath(t *testing.T) {
	assert.Equal(t, "/Users/erikgustavson/projects/cclog", decodeProjectPath("-Users-erikgustavson-projects-cclog"))
	assert.Equal(t, "", decodeProjectPath(""))
	// Note: decodeProjectPath treats ALL dashes as path separators.
	// This is correct for Claude's encoding but means project names with dashes get mangled.
	assert.Equal(t, "/a/b", decodeProjectPath("-a-b"))
}

func TestTruncatePrompt(t *testing.T) {
	assert.Equal(t, "hello", truncatePrompt("hello", 100))
	assert.Equal(t, "first line", truncatePrompt("first line\nsecond line", 100))
	assert.Equal(t, "ab...", truncatePrompt("abcde", 2))
}

func TestPeekSession_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")
	require.NoError(t, os.WriteFile(path, []byte{}, 0o644))

	slug, prompt := peekSession(path)
	assert.Empty(t, slug)
	assert.Empty(t, prompt)
}

func TestPeekSession_WithContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")
	content := `{"type":"file-history-snapshot","messageId":"m1","snapshot":{}}
{"type":"user","sessionId":"abc","uuid":"u1","slug":"my-slug","timestamp":"2026-02-13T07:12:33.727Z","message":{"role":"user","content":"Build a CLI tool"}}
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	slug, prompt := peekSession(path)
	assert.Equal(t, "my-slug", slug)
	assert.Equal(t, "Build a CLI tool", prompt)
}

func TestPeekSession_ArrayContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")
	content := `{"type":"user","sessionId":"abc","uuid":"u1","slug":"arr-slug","timestamp":"2026-02-13T07:12:33.727Z","message":{"role":"user","content":[{"type":"text","text":"Array content msg"}]}}
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	slug, prompt := peekSession(path)
	assert.Equal(t, "arr-slug", slug)
	assert.Equal(t, "Array content msg", prompt)
}
