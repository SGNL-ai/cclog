package renderer

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/sgnl-ai/cclog/internal/parser"
)

// RenderMarkdown produces clean GitHub Flavored Markdown with grouped messages.
func RenderMarkdown(opts Options) ([]byte, error) {
	var buf bytes.Buffer

	title := opts.Title
	if title == "" {
		title = "Claude Code Session"
	}
	buf.WriteString("# " + title + "\n\n")

	for _, g := range groupMessages(opts.Messages) {
		switch g.Role {
		case "boundary":
			buf.WriteString("---\n\n")
		case "user":
			buf.WriteString("### User\n\n")
			for _, msg := range g.Messages {
				writeMarkdownEntry(&buf, msg)
			}
			buf.WriteString("\n")
		case "assistant":
			buf.WriteString("### Assistant\n\n")
			for _, msg := range g.Messages {
				writeMarkdownEntry(&buf, msg)
			}
			buf.WriteString("\n")
		}
	}

	return buf.Bytes(), nil
}

// writeMarkdownEntry writes a single timestamped message within a group.
func writeMarkdownEntry(buf *bytes.Buffer, msg parser.Message) {
	ts := "         "
	if !msg.Timestamp.IsZero() {
		ts = msg.Timestamp.Format("15:04:05") + " "
	}

	if msg.TextContent != "" {
		lines := strings.Split(msg.TextContent, "\n")
		for i, line := range lines {
			prefix := ts
			if i > 0 {
				prefix = "         "
			}
			buf.WriteString(prefix + line + "\n")
		}
	}

	for _, tc := range msg.ToolCalls {
		desc := tc.Description
		if desc != "" {
			desc = " — " + desc
		}
		fmt.Fprintf(buf, "%s- `%s`%s\n", ts, tc.Name, desc)
		ts = "         " // only first line gets timestamp
	}
}
