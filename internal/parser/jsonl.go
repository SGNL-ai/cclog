package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

// Message represents a parsed JSONL line from a Claude Code session.
type Message struct {
	Type       string
	Role       string
	Timestamp  time.Time
	SessionID  string
	UUID       string
	ParentUUID *string

	TextContent string
	ToolCalls   []ToolCall

	Raw json.RawMessage
}

// ToolCall represents an assistant tool invocation.
type ToolCall struct {
	Name        string
	Description string
}

// ParseFile reads a JSONL session file and returns parsed messages.
// If startLine/endLine are 0, all lines are included.
func ParseFile(path string, startLine, endLine int) ([]Message, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open session file: %w", err)
	}
	defer f.Close() //nolint:errcheck

	return ParseReader(f, startLine, endLine)
}

// ParseReader reads JSONL from a reader and returns parsed messages.
func ParseReader(r io.Reader, startLine, endLine int) ([]Message, error) {
	var messages []Message
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 10MB max line

	lineNum := 0
	for scanner.Scan() {
		lineNum++

		if startLine > 0 && lineNum < startLine {
			continue
		}
		if endLine > 0 && lineNum > endLine {
			break
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		msg, skip, err := parseLine(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		if skip {
			continue
		}

		messages = append(messages, msg)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}

	return messages, nil
}

// rawLine is the intermediate structure for JSON unmarshalling.
type rawLine struct {
	Type       string          `json:"type"`
	SessionID  string          `json:"sessionId"`
	UUID       string          `json:"uuid"`
	ParentUUID *string         `json:"parentUuid"`
	Timestamp  string          `json:"timestamp"`
	Message    json.RawMessage `json:"message"`
	Slug       string          `json:"slug"`

	// queue-operation fields
	Content string `json:"content"`
}

type rawMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type contentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	Name  string          `json:"name"`  // tool_use
	ID    string          `json:"id"`    // tool_use
	Input json.RawMessage `json:"input"` // tool_use

	// tool_result fields
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"` // can be string or array
}

var internalTagNames = []string{
	"system-reminder",
	"local-command-caveat",
	"bash-input",
	"bash-stdout",
	"bash-stderr",
	"antml:thinking",
	"antml:function_calls",
	"user-prompt-submit-hook",
}

var internalTagRes []*regexp.Regexp

func init() {
	for _, tag := range internalTagNames {
		// Each regex matches its own opening and closing tag, ensuring they pair correctly.
		re := regexp.MustCompile(`(?s)<` + regexp.QuoteMeta(tag) + `>.*?</` + regexp.QuoteMeta(tag) + `>`)
		internalTagRes = append(internalTagRes, re)
	}
}

// StripSystemReminders removes internal Claude Code tags from text.
func StripSystemReminders(text string) string {
	for _, re := range internalTagRes {
		text = re.ReplaceAllString(text, "")
	}
	return strings.TrimSpace(text)
}

func parseLine(line []byte) (Message, bool, error) {
	var raw rawLine
	if err := json.Unmarshal(line, &raw); err != nil {
		return Message{}, false, fmt.Errorf("unmarshal: %w", err)
	}

	// Skip types we don't render
	switch raw.Type {
	case "progress", "system", "pr-link":
		return Message{}, true, nil
	}

	ts, _ := time.Parse(time.RFC3339Nano, raw.Timestamp)

	msg := Message{
		Type:       raw.Type,
		Timestamp:  ts,
		SessionID:  raw.SessionID,
		UUID:       raw.UUID,
		ParentUUID: raw.ParentUUID,
		Raw:        json.RawMessage(line),
	}

	switch raw.Type {
	case "file-history-snapshot":
		return msg, false, nil

	case "queue-operation":
		msg.TextContent = StripSystemReminders(raw.Content)
		return msg, false, nil

	case "user", "assistant":
		if len(raw.Message) == 0 {
			return msg, false, nil
		}
		return parseMessage(msg, raw.Message)

	default:
		// Unknown type — include as-is
		return msg, false, nil
	}
}

func parseMessage(msg Message, rawMsg json.RawMessage) (Message, bool, error) {
	var rm rawMessage
	if err := json.Unmarshal(rawMsg, &rm); err != nil {
		return msg, false, fmt.Errorf("unmarshal message: %w", err)
	}

	msg.Role = rm.Role

	// Content can be a string or an array of content blocks
	if len(rm.Content) == 0 {
		return msg, false, nil
	}

	// Try string first
	var textContent string
	if err := json.Unmarshal(rm.Content, &textContent); err == nil {
		msg.TextContent = StripSystemReminders(textContent)
		return msg, false, nil
	}

	// Parse as array of content blocks
	var blocks []contentBlock
	if err := json.Unmarshal(rm.Content, &blocks); err != nil {
		return msg, false, fmt.Errorf("unmarshal content blocks: %w", err)
	}

	var textParts []string
	isToolResult := false

	for _, b := range blocks {
		switch b.Type {
		case "text":
			cleaned := StripSystemReminders(b.Text)
			if cleaned != "" {
				textParts = append(textParts, cleaned)
			}

		case "thinking":
			// Skip thinking blocks — internal to Claude

		case "tool_use":
			tc := ToolCall{Name: b.Name}
			tc.Description = extractToolDescription(b.Name, b.Input)
			msg.ToolCalls = append(msg.ToolCalls, tc)

		case "tool_result":
			isToolResult = true
		}
	}

	// Skip pure tool_result messages (they're internal plumbing)
	if isToolResult && len(textParts) == 0 {
		return msg, true, nil
	}

	msg.TextContent = strings.Join(textParts, "\n\n")
	return msg, false, nil
}

// toolDescFields maps tool names to the JSON field and optional prefix used
// for generating a human-readable description.
var toolDescFields = map[string]struct {
	field  string
	prefix string
}{
	"Read":  {field: "file_path"},
	"Write": {field: "file_path"},
	"Edit":  {field: "file_path"},
	"Grep":  {field: "pattern", prefix: "pattern: "},
	"Glob":  {field: "pattern", prefix: "pattern: "},
	"Task":  {field: "description"},
}

// extractToolDescription pulls a human-readable summary from tool input.
func extractToolDescription(toolName string, input json.RawMessage) string {
	if len(input) == 0 {
		return ""
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(input, &fields); err != nil {
		return ""
	}

	// Bash is special: prefer description, fall back to command.
	if toolName == "Bash" {
		if desc, ok := fields["description"]; ok {
			var s string
			if json.Unmarshal(desc, &s) == nil && s != "" {
				return s
			}
		}
		if cmd, ok := fields["command"]; ok {
			var s string
			if json.Unmarshal(cmd, &s) == nil {
				return truncate(s, 120)
			}
		}
	}

	// Table-driven lookup for all other tools.
	if spec, ok := toolDescFields[toolName]; ok {
		if raw, ok := fields[spec.field]; ok {
			var s string
			if json.Unmarshal(raw, &s) == nil && s != "" {
				return spec.prefix + s
			}
		}
	}

	// Generic fallback: try "description" field.
	if desc, ok := fields["description"]; ok {
		var s string
		if json.Unmarshal(desc, &s) == nil && s != "" {
			return truncate(s, 120)
		}
	}

	return ""
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
