package renderer

import (
	"github.com/sgnl-ai/cclog/internal/parser"
)

// Options configures the rendering output.
type Options struct {
	Messages []parser.Message
	Title    string
}

// messageGroup is a run of consecutive messages from the same role.
type messageGroup struct {
	Role     string // "user" or "assistant"
	Messages []parser.Message
}

// groupMessages collapses consecutive same-role messages into groups.
// Boundaries and queue-operations are emitted as single-message groups
// with Role set to their type.
func groupMessages(messages []parser.Message) []messageGroup {
	var groups []messageGroup

	for _, msg := range messages {
		// Determine the grouping key.
		var role string
		switch {
		case msg.Type == "file-history-snapshot":
			role = "boundary"
		case msg.Type == "queue-operation":
			if msg.TextContent == "" {
				continue
			}
			role = "user"
		case msg.Role == "user":
			if msg.TextContent == "" {
				continue
			}
			role = "user"
		case msg.Role == "assistant":
			if msg.TextContent == "" && len(msg.ToolCalls) == 0 {
				continue
			}
			role = "assistant"
		default:
			continue
		}

		// Boundaries always break the group.
		if role == "boundary" {
			groups = append(groups, messageGroup{Role: role, Messages: []parser.Message{msg}})
			continue
		}

		// Append to current group if same role, otherwise start new group.
		if len(groups) > 0 && groups[len(groups)-1].Role == role {
			groups[len(groups)-1].Messages = append(groups[len(groups)-1].Messages, msg)
		} else {
			groups = append(groups, messageGroup{Role: role, Messages: []parser.Message{msg}})
		}
	}

	return groups
}
