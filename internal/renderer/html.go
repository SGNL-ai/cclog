package renderer

import (
	"bytes"
	"fmt"
	"html"

	"github.com/sgnl-ai/cclog/internal/parser"
	"github.com/yuin/goldmark"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

// RenderHTML produces a self-contained HTML file with a dark terminal theme.
// Consecutive same-role messages are grouped under a single role header.
func RenderHTML(opts Options) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString(htmlHead(opts.Title))

	for _, g := range groupMessages(opts.Messages) {
		switch g.Role {
		case "boundary":
			msg := g.Messages[0]
			buf.WriteString(`<div class="boundary">` + "\n")
			buf.WriteString(`  <span class="boundary-marker">` + html.EscapeString("──── session boundary ────") + `</span>` + "\n")
			if !msg.Timestamp.IsZero() {
				buf.WriteString(`  <span class="timestamp">` + msg.Timestamp.Format("2006-01-02 15:04:05") + `</span>` + "\n")
			}
			buf.WriteString("</div>\n")

		case "user":
			buf.WriteString(`<div class="message user">` + "\n")
			buf.WriteString(`  <div class="role">user</div>` + "\n")
			for _, msg := range g.Messages {
				writeHTMLEntry(&buf, msg)
			}
			buf.WriteString("</div>\n")

		case "assistant":
			buf.WriteString(`<div class="message assistant">` + "\n")
			buf.WriteString(`  <div class="role">assistant</div>` + "\n")
			for _, msg := range g.Messages {
				writeHTMLEntry(&buf, msg)
				for _, tc := range msg.ToolCalls {
					writeToolCall(&buf, tc, msg)
				}
			}
			buf.WriteString("</div>\n")
		}
	}

	buf.WriteString(htmlFoot())

	return buf.Bytes(), nil
}

// writeHTMLEntry writes a single timestamped entry within a message group.
// Only renders if there is text content; tool calls are handled by writeToolCall.
func writeHTMLEntry(buf *bytes.Buffer, msg parser.Message) {
	if msg.TextContent == "" {
		return
	}
	buf.WriteString(`  <div class="entry">` + "\n")
	if !msg.Timestamp.IsZero() {
		buf.WriteString(`    <span class="timestamp">` + msg.Timestamp.Format("15:04:05") + `</span>` + "\n")
	}
	buf.WriteString(`    <div class="content">` + renderMarkdownToHTML(msg.TextContent) + `</div>` + "\n")
	buf.WriteString(`  </div>` + "\n")
}

func writeToolCall(buf *bytes.Buffer, tc parser.ToolCall, msg parser.Message) {
	desc := tc.Description
	if desc != "" {
		desc = " — " + html.EscapeString(desc)
	}
	buf.WriteString(`  <div class="entry">` + "\n")
	if !msg.Timestamp.IsZero() {
		buf.WriteString(`    <span class="timestamp">` + msg.Timestamp.Format("15:04:05") + `</span>` + "\n")
	}
	fmt.Fprintf(buf, `    <div class="tool-call">● %s%s</div>`+"\n", html.EscapeString(tc.Name), desc)
	buf.WriteString(`  </div>` + "\n")
}

// mdRenderer is a goldmark instance configured for safe HTML output.
var mdRenderer = goldmark.New(
	goldmark.WithRendererOptions(
		gmhtml.WithHardWraps(),
	),
)

// renderMarkdownToHTML converts Markdown text to HTML using goldmark.
func renderMarkdownToHTML(text string) string {
	var buf bytes.Buffer
	if err := mdRenderer.Convert([]byte(text), &buf); err != nil {
		// Fallback: escape and wrap in <p>.
		return "<p>" + html.EscapeString(text) + "</p>"
	}
	return buf.String()
}

func htmlHead(title string) string {
	if title == "" {
		title = "Claude Code Session"
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<style>
  :root {
    --bg: #1a1b26;
    --fg: #c0caf5;
    --user-border: #9ece6a;
    --assistant-border: #7aa2f7;
    --tool-color: #bb9af7;
    --boundary-color: #565f89;
    --timestamp-color: #565f89;
    --badge-bg: #3d59a1;
  }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    background: var(--bg);
    color: var(--fg);
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    font-size: 15px;
    line-height: 1.6;
    padding: 2rem;
    max-width: 900px;
    margin: 0 auto;
  }
  h1 {
    color: var(--fg);
    margin-bottom: 2rem;
    font-size: 1.5rem;
    border-bottom: 1px solid var(--boundary-color);
    padding-bottom: 0.5rem;
  }
  .message {
    margin: 1.5rem 0;
    padding: 1rem 1.25rem;
    border-radius: 8px;
    background: rgba(255,255,255,0.03);
  }
  .message.user { border-left: 3px solid var(--user-border); }
  .message.assistant { border-left: 3px solid var(--assistant-border); }
  .role {
    font-weight: 600;
    font-size: 0.8rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-bottom: 0.75rem;
  }
  .message.user .role { color: var(--user-border); }
  .message.assistant .role { color: var(--assistant-border); }
  .entry {
    display: grid;
    grid-template-columns: 5.5em 1fr;
    gap: 0 0.75rem;
    margin: 0.4rem 0;
    align-items: start;
  }
  .timestamp {
    color: var(--timestamp-color);
    font-size: 0.75rem;
    font-family: monospace;
    padding-top: 0.15em;
  }
  .content { line-height: 1.45; min-width: 0; overflow-wrap: break-word; }
  .content p { margin: 0.3em 0; }
  .content code {
    background: rgba(255,255,255,0.08);
    padding: 0.15em 0.35em;
    border-radius: 3px;
    font-family: 'SF Mono', 'Fira Code', monospace;
    font-size: 0.9em;
  }
  .content pre {
    background: rgba(0,0,0,0.3);
    border-radius: 6px;
    padding: 0.75rem 1rem;
    margin: 0.5em 0;
    overflow-x: auto;
    max-width: 100%%;
  }
  .content pre code {
    background: none;
    padding: 0;
    font-size: 0.85em;
    line-height: 1.4;
  }
  .tool-call {
    color: var(--tool-color);
    font-family: 'SF Mono', 'Fira Code', monospace;
    font-size: 0.85rem;
  }
  .boundary {
    text-align: center;
    margin: 2rem 0;
    color: var(--boundary-color);
    font-size: 0.85rem;
  }
  .boundary .timestamp {
    display: block;
    margin-top: 0.25rem;
  }
  @media (max-width: 600px) {
    body { padding: 1rem; font-size: 14px; }
    .message { padding: 0.75rem 1rem; }
    .entry { grid-template-columns: 1fr; }
    .entry .timestamp { margin-bottom: 0.2rem; }
  }
</style>
</head>
<body>
<h1>%s</h1>
`, html.EscapeString(title), html.EscapeString(title))
}

func htmlFoot() string {
	return `</body>
</html>
`
}
