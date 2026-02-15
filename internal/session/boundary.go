package session

import (
	"fmt"
	"time"

	"github.com/sgnl-ai/cclog/internal/parser"
)

const timeGapThreshold = 1 * time.Hour

// Boundary represents a logical breakpoint in a session transcript.
type Boundary struct {
	LineIndex   int
	Timestamp   time.Time
	Reason      string
	PreviewText string
}

// DetectBoundaries identifies logical boundaries in a parsed message list.
// Boundaries are points where a new conversation segment begins.
func DetectBoundaries(messages []parser.Message) []Boundary {
	if len(messages) == 0 {
		return nil
	}

	var boundaries []Boundary

	// First message is always a boundary
	boundaries = append(boundaries, Boundary{
		LineIndex:   0,
		Timestamp:   messages[0].Timestamp,
		Reason:      "session start",
		PreviewText: previewFor(messages, 0),
	})

	for i := 1; i < len(messages); i++ {
		msg := messages[i]

		// file-history-snapshot marks a session boundary
		if msg.Type == "file-history-snapshot" {
			boundaries = append(boundaries, Boundary{
				LineIndex:   i,
				Timestamp:   msg.Timestamp,
				Reason:      "history snapshot",
				PreviewText: previewFor(messages, i),
			})
			continue
		}

		// parentUuid == nil on a user message means conversation restart
		if msg.Type == "user" && msg.ParentUUID == nil {
			boundaries = append(boundaries, Boundary{
				LineIndex:   i,
				Timestamp:   msg.Timestamp,
				Reason:      "conversation restart",
				PreviewText: previewFor(messages, i),
			})
			continue
		}

		// Time gap > threshold
		if !msg.Timestamp.IsZero() && !messages[i-1].Timestamp.IsZero() {
			gap := msg.Timestamp.Sub(messages[i-1].Timestamp)
			if gap >= timeGapThreshold {
				boundaries = append(boundaries, Boundary{
					LineIndex:   i,
					Timestamp:   msg.Timestamp,
					Reason:      fmt.Sprintf("time gap (%s)", formatDuration(gap)),
					PreviewText: previewFor(messages, i),
				})
				continue
			}
		}
	}

	return boundaries
}

// previewFor returns the text of the first user message at or after index i.
func previewFor(messages []parser.Message, startIdx int) string {
	for j := startIdx; j < len(messages) && j < startIdx+5; j++ {
		if messages[j].Role == "user" && messages[j].TextContent != "" {
			text := messages[j].TextContent
			if len(text) > 80 {
				return text[:80] + "..."
			}
			return text
		}
	}
	return ""
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}
