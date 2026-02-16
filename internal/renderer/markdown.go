package renderer

import (
	"bytes"
	"fmt"

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
		case "assistant":
			buf.WriteString("### Assistant\n\n")
			for _, msg := range g.Messages {
				writeMarkdownEntry(&buf, msg)
			}
		}
	}

	return buf.Bytes(), nil
}

// writeMarkdownEntry writes a single timestamped message within a group.
// The timestamp appears on its own line, and the content flows at column 0
// so that GFM features (code fences, lists, etc.) render correctly.
func writeMarkdownEntry(buf *bytes.Buffer, msg parser.Message) {
	if msg.TextContent == "" && len(msg.ToolCalls) == 0 {
		return
	}

	if !msg.Timestamp.IsZero() {
		fmt.Fprintf(buf, "%s ", msg.Timestamp.Format("15:04:05"))
	}

	if msg.TextContent != "" {
		buf.WriteString(msg.TextContent + "\n")
	}

	for _, tc := range msg.ToolCalls {
		desc := tc.Description
		if desc != "" {
			desc = " — " + desc
		}
		fmt.Fprintf(buf, "- `%s`%s\n", tc.Name, desc)
	}

	buf.WriteString("\n")
}
