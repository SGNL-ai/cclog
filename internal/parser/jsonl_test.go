package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseReader_UserMessageStringContent(t *testing.T) {
	line := `{"type":"user","sessionId":"abc-123","uuid":"u1","parentUuid":null,"timestamp":"2026-02-13T07:12:33.727Z","message":{"role":"user","content":"Hello, how are you?"}}`

	msgs, err := ParseReader(strings.NewReader(line), 0, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	msg := msgs[0]
	assert.Equal(t, "user", msg.Type)
	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "abc-123", msg.SessionID)
	assert.Equal(t, "u1", msg.UUID)
	assert.Nil(t, msg.ParentUUID)
	assert.Equal(t, "Hello, how are you?", msg.TextContent)
	assert.False(t, msg.Timestamp.IsZero())
	assert.NotEmpty(t, msg.Raw, "Raw should be populated")
}

func TestParseReader_UserMessageArrayContent(t *testing.T) {
	line := `{"type":"user","sessionId":"abc","uuid":"u2","parentUuid":"u1","timestamp":"2026-02-13T07:12:33.727Z","message":{"role":"user","content":[{"type":"text","text":"Please fix the bug"}]}}`

	msgs, err := ParseReader(strings.NewReader(line), 0, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	msg := msgs[0]
	assert.Equal(t, "user", msg.Type)
	assert.Equal(t, "Please fix the bug", msg.TextContent)

	parentUUID := "u1"
	assert.Equal(t, &parentUUID, msg.ParentUUID)
}

func TestParseReader_AssistantMessageWithText(t *testing.T) {
	line := `{"type":"assistant","sessionId":"abc","uuid":"a1","parentUuid":"u1","timestamp":"2026-02-13T07:12:36.915Z","message":{"role":"assistant","content":[{"type":"text","text":"I'll help you with that."}]}}`

	msgs, err := ParseReader(strings.NewReader(line), 0, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	msg := msgs[0]
	assert.Equal(t, "assistant", msg.Type)
	assert.Equal(t, "assistant", msg.Role)
	assert.Equal(t, "I'll help you with that.", msg.TextContent)
	assert.Empty(t, msg.ToolCalls)
}

func TestParseReader_AssistantMessageWithToolUse(t *testing.T) {
	line := `{"type":"assistant","sessionId":"abc","uuid":"a2","parentUuid":"u1","timestamp":"2026-02-13T07:12:39.033Z","message":{"role":"assistant","content":[{"type":"text","text":"Let me check."},{"type":"tool_use","id":"toolu_123","name":"Bash","input":{"command":"date","description":"Check current date"}}]}}`

	msgs, err := ParseReader(strings.NewReader(line), 0, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	msg := msgs[0]
	assert.Equal(t, "Let me check.", msg.TextContent)
	require.Len(t, msg.ToolCalls, 1)
	assert.Equal(t, "Bash", msg.ToolCalls[0].Name)
	assert.Equal(t, "Check current date", msg.ToolCalls[0].Description)
}

func TestParseReader_AssistantOnlyToolCalls_NoText(t *testing.T) {
	line := `{"type":"assistant","sessionId":"abc","uuid":"a1","timestamp":"2026-02-13T07:12:39.033Z","message":{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"Read","input":{"file_path":"/foo.go"}}]}}`

	msgs, err := ParseReader(strings.NewReader(line), 0, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	msg := msgs[0]
	assert.Equal(t, "", msg.TextContent, "TextContent should be empty when there's only tool_use")
	require.Len(t, msg.ToolCalls, 1)
	assert.Equal(t, "Read", msg.ToolCalls[0].Name)
}

func TestParseReader_ToolResultSkipped(t *testing.T) {
	line := `{"type":"user","sessionId":"abc","uuid":"u3","parentUuid":"a2","timestamp":"2026-02-13T07:12:39.455Z","message":{"role":"user","content":[{"tool_use_id":"toolu_123","type":"tool_result","content":"Thu Feb 12 23:12:39 PST 2026","is_error":false}]}}`

	msgs, err := ParseReader(strings.NewReader(line), 0, 0)
	require.NoError(t, err)
	assert.Empty(t, msgs, "pure tool_result messages should be skipped")
}

func TestParseReader_ToolResultWithTextNotSkipped(t *testing.T) {
	// A message with both tool_result AND text should NOT be skipped
	line := `{"type":"user","sessionId":"abc","uuid":"u3","parentUuid":"a2","timestamp":"2026-02-13T07:12:39.455Z","message":{"role":"user","content":[{"tool_use_id":"toolu_123","type":"tool_result","content":"output"},{"type":"text","text":"user follow-up"}]}}`

	msgs, err := ParseReader(strings.NewReader(line), 0, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1, "messages with tool_result AND text should be kept")
	assert.Equal(t, "user follow-up", msgs[0].TextContent)
}

func TestParseReader_SkipsProgressAndSystem(t *testing.T) {
	lines := strings.Join([]string{
		`{"type":"progress","sessionId":"abc","uuid":"p1","timestamp":"2026-02-13T07:12:40.000Z","data":{}}`,
		`{"type":"system","sessionId":"abc","uuid":"s1","timestamp":"2026-02-13T07:12:41.000Z","subtype":"turn_duration","durationMs":5000}`,
		`{"type":"pr-link","sessionId":"abc","timestamp":"2026-02-13T07:12:42.000Z","prUrl":"https://github.com/test/pr/1"}`,
		`{"type":"user","sessionId":"abc","uuid":"u1","timestamp":"2026-02-13T07:12:43.000Z","message":{"role":"user","content":"hello"}}`,
	}, "\n")

	msgs, err := ParseReader(strings.NewReader(lines), 0, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "user", msgs[0].Type)
}

func TestParseReader_FileHistorySnapshot(t *testing.T) {
	line := `{"type":"file-history-snapshot","messageId":"snap1","snapshot":{"messageId":"snap1","trackedFileBackups":{},"timestamp":"2026-02-13T07:12:33.883Z"},"isSnapshotUpdate":false}`

	msgs, err := ParseReader(strings.NewReader(line), 0, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "file-history-snapshot", msgs[0].Type)
}

func TestParseReader_QueueOperation(t *testing.T) {
	line := `{"type":"queue-operation","operation":"enqueue","timestamp":"2026-02-13T08:00:00.000Z","sessionId":"abc","content":"also fix the tests"}`

	msgs, err := ParseReader(strings.NewReader(line), 0, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "queue-operation", msgs[0].Type)
	assert.Equal(t, "also fix the tests", msgs[0].TextContent)
}

func TestParseReader_UnknownTypePassedThrough(t *testing.T) {
	line := `{"type":"some-future-type","sessionId":"abc","uuid":"x1","timestamp":"2026-02-13T07:00:00.000Z"}`

	msgs, err := ParseReader(strings.NewReader(line), 0, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "some-future-type", msgs[0].Type)
}

func TestParseReader_MalformedJSON(t *testing.T) {
	line := `{this is not valid json}`

	_, err := ParseReader(strings.NewReader(line), 0, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "line 1")
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestParseReader_LineRange(t *testing.T) {
	lines := strings.Join([]string{
		`{"type":"user","sessionId":"abc","uuid":"u1","timestamp":"2026-02-13T07:00:00.000Z","message":{"role":"user","content":"line one"}}`,
		`{"type":"user","sessionId":"abc","uuid":"u2","timestamp":"2026-02-13T07:01:00.000Z","message":{"role":"user","content":"line two"}}`,
		`{"type":"user","sessionId":"abc","uuid":"u3","timestamp":"2026-02-13T07:02:00.000Z","message":{"role":"user","content":"line three"}}`,
	}, "\n")

	msgs, err := ParseReader(strings.NewReader(lines), 2, 2)
	require.NoError(t, err)
	require.Len(t, msgs, 1, "should only include line 2")
	assert.Equal(t, "line two", msgs[0].TextContent)

	// Verify startLine excludes earlier lines
	msgs2, err := ParseReader(strings.NewReader(lines), 3, 0)
	require.NoError(t, err)
	require.Len(t, msgs2, 1, "should only include line 3 onward")
	assert.Equal(t, "line three", msgs2[0].TextContent)
}

func TestParseFile_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")
	content := `{"type":"user","sessionId":"abc","uuid":"u1","timestamp":"2026-02-13T07:00:00.000Z","message":{"role":"user","content":"from file"}}` + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	msgs, err := ParseFile(path, 0, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "from file", msgs[0].TextContent)
}

func TestParseFile_NonexistentFile(t *testing.T) {
	_, err := ParseFile("/nonexistent/path/file.jsonl", 0, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "open session file")
}

func TestStripSystemReminders(t *testing.T) {
	input := `Hello world <system-reminder>this should be removed</system-reminder> and more text`
	got := StripSystemReminders(input)
	assert.Equal(t, "Hello world  and more text", got)
}

func TestStripSystemReminders_MultiLine(t *testing.T) {
	input := "Hello\n<system-reminder>\nfoo\nbar\n</system-reminder>\nworld"
	got := StripSystemReminders(input)
	assert.Equal(t, "Hello\n\nworld", got)
}

func TestStripSystemReminders_Multiple(t *testing.T) {
	input := "<system-reminder>a</system-reminder>text<system-reminder>b</system-reminder>"
	got := StripSystemReminders(input)
	assert.Equal(t, "text", got)
}

func TestStripSystemReminders_InternalTags(t *testing.T) {
	input := `<local-command-caveat>Caveat: messages below</local-command-caveat> hello`
	got := StripSystemReminders(input)
	assert.Equal(t, "hello", got)
}

func TestStripSystemReminders_BashTags(t *testing.T) {
	input := `<bash-input>ls -la</bash-input><bash-stdout>output here</bash-stdout>`
	got := StripSystemReminders(input)
	assert.Equal(t, "", got)
}

func TestStripSystemReminders_MismatchedTagsPreserved(t *testing.T) {
	// Mismatched opening/closing tags should NOT be stripped — this was a bug
	input := `<system-reminder>content</bash-input> remaining`
	got := StripSystemReminders(input)
	assert.Equal(t, "<system-reminder>content</bash-input> remaining", got,
		"mismatched tags should not be stripped")
}

func TestStripSystemReminders_NoTags(t *testing.T) {
	input := "plain text with no tags at all"
	got := StripSystemReminders(input)
	assert.Equal(t, input, got)
}

func TestParseReader_ThinkingBlocksSkipped(t *testing.T) {
	line := `{"type":"assistant","sessionId":"abc","uuid":"a1","timestamp":"2026-02-13T07:12:37.582Z","message":{"role":"assistant","content":[{"type":"thinking","thinking":"Let me think about this..."},{"type":"text","text":"Here is my answer."}]}}`

	msgs, err := ParseReader(strings.NewReader(line), 0, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "Here is my answer.", msgs[0].TextContent)
	assert.Empty(t, msgs[0].ToolCalls)
}

func TestParseReader_SystemReminderInContent(t *testing.T) {
	line := `{"type":"user","sessionId":"abc","uuid":"u1","timestamp":"2026-02-13T07:12:33.727Z","message":{"role":"user","content":"do the thing <system-reminder>secret stuff</system-reminder>"}}`

	msgs, err := ParseReader(strings.NewReader(line), 0, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "do the thing", msgs[0].TextContent)
}

func TestExtractToolDescription_ReadFile(t *testing.T) {
	input := `{"file_path":"/Users/test/foo.go"}`
	desc := extractToolDescription("Read", []byte(input))
	assert.Equal(t, "/Users/test/foo.go", desc)
}

func TestExtractToolDescription_GrepPattern(t *testing.T) {
	input := `{"pattern":"func main"}`
	desc := extractToolDescription("Grep", []byte(input))
	assert.Equal(t, "pattern: func main", desc)
}

func TestExtractToolDescription_GlobPattern(t *testing.T) {
	input := `{"pattern":"**/*.go"}`
	desc := extractToolDescription("Glob", []byte(input))
	assert.Equal(t, "pattern: **/*.go", desc)
}

func TestExtractToolDescription_BashCommand(t *testing.T) {
	input := `{"command":"go test ./..."}`
	desc := extractToolDescription("Bash", []byte(input))
	assert.Equal(t, "go test ./...", desc)
}

func TestExtractToolDescription_BashWithDescription(t *testing.T) {
	input := `{"command":"go test ./...","description":"Run all tests"}`
	desc := extractToolDescription("Bash", []byte(input))
	assert.Equal(t, "Run all tests", desc)
}

func TestExtractToolDescription_TaskDescription(t *testing.T) {
	input := `{"description":"Explore auth module","prompt":"find auth code"}`
	desc := extractToolDescription("Task", []byte(input))
	assert.Equal(t, "Explore auth module", desc)
}

func TestExtractToolDescription_EmptyInput(t *testing.T) {
	assert.Equal(t, "", extractToolDescription("Read", nil))
	assert.Equal(t, "", extractToolDescription("Read", []byte{}))
	assert.Equal(t, "", extractToolDescription("Read", []byte(`{}`)))
}

func TestExtractToolDescription_InvalidJSON(t *testing.T) {
	desc := extractToolDescription("Read", []byte(`not json`))
	assert.Equal(t, "", desc)
}

func TestExtractToolDescription_UnknownToolWithDescription(t *testing.T) {
	input := `{"description":"some operation","other":"field"}`
	desc := extractToolDescription("UnknownTool", []byte(input))
	assert.Equal(t, "some operation", desc, "should fall through to generic description extraction")
}

func TestExtractToolDescription_UnknownToolWithoutDescription(t *testing.T) {
	input := `{"other":"field"}`
	desc := extractToolDescription("UnknownTool", []byte(input))
	assert.Equal(t, "", desc)
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "hello", truncate("hello", 10))
	assert.Equal(t, "hel...", truncate("hello world", 3))
	assert.Equal(t, "", truncate("", 5))
	assert.Equal(t, "exact", truncate("exact", 5))
}

func TestParseReader_EmptyLines(t *testing.T) {
	lines := "\n\n" + `{"type":"user","sessionId":"abc","uuid":"u1","timestamp":"2026-02-13T07:00:00.000Z","message":{"role":"user","content":"hello"}}` + "\n\n"

	msgs, err := ParseReader(strings.NewReader(lines), 0, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
}

func TestParseReader_MultipleToolCalls(t *testing.T) {
	line := `{"type":"assistant","sessionId":"abc","uuid":"a1","timestamp":"2026-02-13T07:12:39.033Z","message":{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"Read","input":{"file_path":"/foo.go"}},{"type":"tool_use","id":"t2","name":"Read","input":{"file_path":"/bar.go"}}]}}`

	msgs, err := ParseReader(strings.NewReader(line), 0, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Len(t, msgs[0].ToolCalls, 2)
	assert.Equal(t, "Read", msgs[0].ToolCalls[0].Name)
	assert.Equal(t, "/foo.go", msgs[0].ToolCalls[0].Description)
	assert.Equal(t, "Read", msgs[0].ToolCalls[1].Name)
	assert.Equal(t, "/bar.go", msgs[0].ToolCalls[1].Description)
}

func TestParseReader_WriteToolDescription(t *testing.T) {
	line := `{"type":"assistant","sessionId":"abc","uuid":"a1","timestamp":"2026-02-13T07:12:39.033Z","message":{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"Write","input":{"file_path":"/tmp/output.txt","content":"hello"}}]}}`

	msgs, err := ParseReader(strings.NewReader(line), 0, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Len(t, msgs[0].ToolCalls, 1)
	assert.Equal(t, "Write", msgs[0].ToolCalls[0].Name)
	assert.Equal(t, "/tmp/output.txt", msgs[0].ToolCalls[0].Description)
}

func TestParseReader_EditToolDescription(t *testing.T) {
	line := `{"type":"assistant","sessionId":"abc","uuid":"a1","timestamp":"2026-02-13T07:12:39.033Z","message":{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"Edit","input":{"file_path":"/tmp/output.txt","old_string":"foo","new_string":"bar"}}]}}`

	msgs, err := ParseReader(strings.NewReader(line), 0, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	require.Len(t, msgs[0].ToolCalls, 1)
	assert.Equal(t, "Edit", msgs[0].ToolCalls[0].Name)
	assert.Equal(t, "/tmp/output.txt", msgs[0].ToolCalls[0].Description)
}

func TestParseReader_MultipleTextParts(t *testing.T) {
	line := `{"type":"assistant","sessionId":"abc","uuid":"a1","timestamp":"2026-02-13T07:12:39.033Z","message":{"role":"assistant","content":[{"type":"text","text":"First part."},{"type":"text","text":"Second part."}]}}`

	msgs, err := ParseReader(strings.NewReader(line), 0, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "First part.\n\nSecond part.", msgs[0].TextContent,
		"multiple text blocks should be joined with double newline")
}

func TestParseReader_UserMessageWithEmptyMessage(t *testing.T) {
	line := `{"type":"user","sessionId":"abc","uuid":"u1","timestamp":"2026-02-13T07:00:00.000Z"}`

	msgs, err := ParseReader(strings.NewReader(line), 0, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "", msgs[0].TextContent)
	assert.Equal(t, "", msgs[0].Role)
}

func TestParseReader_FullConversation(t *testing.T) {
	lines := strings.Join([]string{
		`{"type":"file-history-snapshot","messageId":"snap1","snapshot":{}}`,
		`{"type":"user","sessionId":"s1","uuid":"u1","parentUuid":null,"timestamp":"2026-02-13T07:00:00.000Z","message":{"role":"user","content":"Fix the login bug"}}`,
		`{"type":"assistant","sessionId":"s1","uuid":"a1","parentUuid":"u1","timestamp":"2026-02-13T07:00:05.000Z","message":{"role":"assistant","content":[{"type":"text","text":"Looking at the code."},{"type":"tool_use","id":"t1","name":"Read","input":{"file_path":"/src/login.go"}}]}}`,
		`{"type":"user","sessionId":"s1","uuid":"u2","parentUuid":"a1","timestamp":"2026-02-13T07:00:06.000Z","message":{"role":"user","content":[{"tool_use_id":"t1","type":"tool_result","content":"package login\n..."}]}}`,
		`{"type":"assistant","sessionId":"s1","uuid":"a2","parentUuid":"u2","timestamp":"2026-02-13T07:00:10.000Z","message":{"role":"assistant","content":[{"type":"text","text":"Found the bug, fixing now."}]}}`,
		`{"type":"progress","sessionId":"s1","uuid":"p1","timestamp":"2026-02-13T07:00:11.000Z","data":{}}`,
		`{"type":"system","sessionId":"s1","uuid":"sys1","timestamp":"2026-02-13T07:00:15.000Z","subtype":"turn_duration","durationMs":15000}`,
	}, "\n")

	msgs, err := ParseReader(strings.NewReader(lines), 0, 0)
	require.NoError(t, err)
	// snapshot + user + assistant(text+tool) + assistant(text) = 4 (tool_result, progress, system skipped)
	require.Len(t, msgs, 4)
	assert.Equal(t, "file-history-snapshot", msgs[0].Type)
	assert.Equal(t, "Fix the login bug", msgs[1].TextContent)
	assert.Equal(t, "Looking at the code.", msgs[2].TextContent)
	require.Len(t, msgs[2].ToolCalls, 1)
	assert.Equal(t, "Found the bug, fixing now.", msgs[3].TextContent)
}
