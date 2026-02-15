package session

import (
	"testing"
	"time"

	"github.com/sgnl-ai/cclog/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectBoundaries_Empty(t *testing.T) {
	boundaries := DetectBoundaries(nil)
	assert.Nil(t, boundaries)
}

func TestDetectBoundaries_SessionStart(t *testing.T) {
	messages := []parser.Message{
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:00:00Z"), TextContent: "Hello"},
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:00:05Z"), TextContent: "Hi there"},
	}

	boundaries := DetectBoundaries(messages)
	require.Len(t, boundaries, 1)
	assert.Equal(t, 0, boundaries[0].LineIndex)
	assert.Equal(t, "session start", boundaries[0].Reason)
	assert.Equal(t, "Hello", boundaries[0].PreviewText)
}

func TestDetectBoundaries_FileHistorySnapshot(t *testing.T) {
	messages := []parser.Message{
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:00:00Z"), TextContent: "First"},
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:00:05Z"), TextContent: "Response"},
		{Type: "file-history-snapshot", Timestamp: ts("2026-02-13T07:01:00Z")},
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:01:01Z"), ParentUUID: strPtr("prev"), TextContent: "Second"},
	}

	boundaries := DetectBoundaries(messages)
	require.Len(t, boundaries, 2)
	assert.Equal(t, "session start", boundaries[0].Reason)
	assert.Equal(t, "history snapshot", boundaries[1].Reason)
	assert.Equal(t, 2, boundaries[1].LineIndex)
	assert.Equal(t, "Second", boundaries[1].PreviewText)
}

func TestDetectBoundaries_ParentUUIDNil(t *testing.T) {
	messages := []parser.Message{
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:00:00Z"), ParentUUID: strPtr("prev"), TextContent: "First"},
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:00:05Z")},
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:00:10Z"), ParentUUID: nil, TextContent: "Restart"},
	}

	boundaries := DetectBoundaries(messages)
	require.Len(t, boundaries, 2)
	assert.Equal(t, "conversation restart", boundaries[1].Reason)
	assert.Equal(t, "Restart", boundaries[1].PreviewText)
}

func TestDetectBoundaries_TimeGap(t *testing.T) {
	messages := []parser.Message{
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:00:00Z"), TextContent: "Morning"},
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:00:05Z")},
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T10:00:00Z"), ParentUUID: strPtr("prev"), TextContent: "Afternoon"},
	}

	boundaries := DetectBoundaries(messages)
	require.Len(t, boundaries, 2)
	assert.Equal(t, "time gap (2h59m)", boundaries[1].Reason)
	assert.Equal(t, "Afternoon", boundaries[1].PreviewText)
}

func TestDetectBoundaries_MultipleBoundaries(t *testing.T) {
	messages := []parser.Message{
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:00:00Z"), TextContent: "Start"},
		{Type: "file-history-snapshot", Timestamp: ts("2026-02-13T07:01:00Z")},
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:01:01Z"), ParentUUID: nil, TextContent: "After snapshot"},
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:01:10Z")},
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T09:00:00Z"), ParentUUID: strPtr("p"), TextContent: "After gap"},
	}

	boundaries := DetectBoundaries(messages)
	require.Len(t, boundaries, 4)
	assert.Equal(t, "session start", boundaries[0].Reason)
	assert.Equal(t, 0, boundaries[0].LineIndex)
	assert.Equal(t, "history snapshot", boundaries[1].Reason)
	assert.Equal(t, 1, boundaries[1].LineIndex)
	assert.Equal(t, "conversation restart", boundaries[2].Reason)
	assert.Equal(t, 2, boundaries[2].LineIndex)
	assert.Contains(t, boundaries[3].Reason, "time gap")
	assert.Equal(t, 4, boundaries[3].LineIndex)
}

func TestDetectBoundaries_NoPreviewAvailable(t *testing.T) {
	// Boundary at a snapshot with no user messages within 5-message lookahead
	messages := []parser.Message{
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:00:00Z"), TextContent: "Start"},
		{Type: "file-history-snapshot", Timestamp: ts("2026-02-13T07:01:00Z")},
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:01:01Z"), ParentUUID: strPtr("p")},
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:01:02Z"), ParentUUID: strPtr("p")},
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:01:03Z"), ParentUUID: strPtr("p")},
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:01:04Z"), ParentUUID: strPtr("p")},
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:01:05Z"), ParentUUID: strPtr("p")},
	}

	boundaries := DetectBoundaries(messages)
	require.Len(t, boundaries, 2)
	assert.Equal(t, "", boundaries[1].PreviewText, "should be empty when no user message within 5")
}

func TestDetectBoundaries_ZeroTimestampNoGap(t *testing.T) {
	messages := []parser.Message{
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:00:00Z"), TextContent: "First"},
		{Type: "user", Role: "user", ParentUUID: strPtr("p"), TextContent: "No timestamp"},
	}

	boundaries := DetectBoundaries(messages)
	require.Len(t, boundaries, 1, "zero timestamp should not trigger time gap")
}

func TestDetectBoundaries_PreviewTruncation(t *testing.T) {
	longText := "This is a very long message that exceeds eighty characters in length and should be truncated in the preview text output"
	messages := []parser.Message{
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:00:00Z"), TextContent: longText},
	}

	boundaries := DetectBoundaries(messages)
	require.Len(t, boundaries, 1)
	assert.Len(t, boundaries[0].PreviewText, 83) // 80 chars + "..."
	assert.True(t, len(boundaries[0].PreviewText) <= 83)
}

func TestDetectBoundaries_PreviewFromNextUserMessage(t *testing.T) {
	messages := []parser.Message{
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:00:00Z"), TextContent: "First"},
		{Type: "file-history-snapshot", Timestamp: ts("2026-02-13T07:01:00Z")},
		{Type: "assistant", Role: "assistant", Timestamp: ts("2026-02-13T07:01:01Z"), ParentUUID: strPtr("prev"), TextContent: "assistant text"},
		{Type: "user", Role: "user", Timestamp: ts("2026-02-13T07:01:02Z"), ParentUUID: strPtr("prev2"), TextContent: "After snapshot user msg"},
	}

	boundaries := DetectBoundaries(messages)
	require.Len(t, boundaries, 2)
	assert.Equal(t, "After snapshot user msg", boundaries[1].PreviewText)
}

func TestFormatDuration(t *testing.T) {
	assert.Equal(t, "1h30m", formatDuration(90*time.Minute))
	assert.Equal(t, "2h0m", formatDuration(120*time.Minute))
	assert.Equal(t, "45m", formatDuration(45*time.Minute))
}

// helpers

func ts(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func strPtr(s string) *string {
	return &s
}
