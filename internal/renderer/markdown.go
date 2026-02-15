package renderer

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/sgnl-ai/cclog/internal/parser"
)

// RenderMarkdown produces clean GitHub Flavored Markdown.
func RenderMarkdown(opts Options) ([]byte, error) {
	var buf bytes.Buffer

	title := opts.Title
	if title == "" {
		title = "Claude Code Session"
	}
	buf.WriteString("# " + title + "\n\n")

	for _, msg := range opts.Messages {
		switch {
		case msg.Type == "file-history-snapshot":
			buf.WriteString("---\n\n")

		case msg.Type == "queue-operation":
			if msg.TextContent != "" {
				writeMarkdownUser(&buf, msg, " _(queued)_")
			}

		case msg.Role == "user":
			if msg.TextContent == "" {
				continue
			}
			writeMarkdownUser(&buf, msg, "")

		case msg.Role == "assistant":
			if msg.TextContent == "" && len(msg.ToolCalls) == 0 {
				continue
			}
			writeMarkdownAssistant(&buf, msg)
		}
	}

	return buf.Bytes(), nil
}

func writeMarkdownUser(buf *bytes.Buffer, msg parser.Message, suffix string) {
	buf.WriteString("### User" + suffix + "\n")
	writeMarkdownTimestamp(buf, msg)
	buf.WriteString("\n")

	// Blockquote user messages
	lines := strings.Split(msg.TextContent, "\n")
	for _, line := range lines {
		buf.WriteString("> " + line + "\n")
	}
	buf.WriteString("\n")
}

func writeMarkdownAssistant(buf *bytes.Buffer, msg parser.Message) {
	buf.WriteString("### Assistant\n")
	writeMarkdownTimestamp(buf, msg)
	buf.WriteString("\n")

	if msg.TextContent != "" {
		buf.WriteString(msg.TextContent + "\n\n")
	}

	for _, tc := range msg.ToolCalls {
		desc := tc.Description
		if desc != "" {
			desc = " — " + desc
		}
		buf.WriteString(fmt.Sprintf("- `%s`%s\n", tc.Name, desc))
	}

	if len(msg.ToolCalls) > 0 {
		buf.WriteString("\n")
	}
}

func writeMarkdownTimestamp(buf *bytes.Buffer, msg parser.Message) {
	if !msg.Timestamp.IsZero() {
		buf.WriteString("_" + msg.Timestamp.Format("15:04:05") + "_\n")
	}
}
