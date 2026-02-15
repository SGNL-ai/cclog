package renderer

import (
	"bytes"
	"fmt"
	"html"
	"strings"

	"github.com/sgnl-ai/cclog/internal/parser"
)

// RenderHTML produces a self-contained HTML file with a dark terminal theme.
func RenderHTML(opts Options) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString(htmlHead(opts.Title))

	for _, msg := range opts.Messages {
		switch {
		case msg.Type == "file-history-snapshot":
			buf.WriteString(`<div class="boundary">` + "\n")
			buf.WriteString(`  <span class="boundary-marker">` + html.EscapeString("──── session boundary ────") + `</span>` + "\n")
			if !msg.Timestamp.IsZero() {
				buf.WriteString(`  <span class="timestamp">` + msg.Timestamp.Format("2006-01-02 15:04:05") + `</span>` + "\n")
			}
			buf.WriteString("</div>\n")

		case msg.Type == "queue-operation":
			if msg.TextContent != "" {
				buf.WriteString(`<div class="message user">` + "\n")
				buf.WriteString(`  <div class="role">user <span class="badge">queued</span></div>` + "\n")
				writeTimestamp(&buf, msg)
				buf.WriteString(`  <div class="content">` + escapeAndFormat(msg.TextContent) + `</div>` + "\n")
				buf.WriteString("</div>\n")
			}

		case msg.Role == "user":
			if msg.TextContent == "" {
				continue
			}
			buf.WriteString(`<div class="message user">` + "\n")
			buf.WriteString(`  <div class="role">user</div>` + "\n")
			writeTimestamp(&buf, msg)
			buf.WriteString(`  <div class="content">` + escapeAndFormat(msg.TextContent) + `</div>` + "\n")
			buf.WriteString("</div>\n")

		case msg.Role == "assistant":
			if msg.TextContent == "" && len(msg.ToolCalls) == 0 {
				continue
			}
			buf.WriteString(`<div class="message assistant">` + "\n")
			buf.WriteString(`  <div class="role">assistant</div>` + "\n")
			writeTimestamp(&buf, msg)
			if msg.TextContent != "" {
				buf.WriteString(`  <div class="content">` + escapeAndFormat(msg.TextContent) + `</div>` + "\n")
			}
			for _, tc := range msg.ToolCalls {
				writeToolCall(&buf, tc)
			}
			buf.WriteString("</div>\n")
		}
	}

	buf.WriteString(htmlFoot())

	return buf.Bytes(), nil
}

func writeTimestamp(buf *bytes.Buffer, msg parser.Message) {
	if !msg.Timestamp.IsZero() {
		buf.WriteString(`  <div class="timestamp">` + msg.Timestamp.Format("15:04:05") + `</div>` + "\n")
	}
}

func writeToolCall(buf *bytes.Buffer, tc parser.ToolCall) {
	desc := tc.Description
	if desc != "" {
		desc = " — " + html.EscapeString(desc)
	}
	buf.WriteString(fmt.Sprintf(`  <div class="tool-call">● %s%s</div>`+"\n", html.EscapeString(tc.Name), desc))
}

func escapeAndFormat(text string) string {
	escaped := html.EscapeString(text)
	// Convert double newlines to paragraph breaks
	paragraphs := strings.Split(escaped, "\n\n")
	var parts []string
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p != "" {
			// Convert single newlines to <br>
			p = strings.ReplaceAll(p, "\n", "<br>\n")
			parts = append(parts, "<p>"+p+"</p>")
		}
	}
	return strings.Join(parts, "\n")
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
  .message.user {
    border-left: 3px solid var(--user-border);
  }
  .message.assistant {
    border-left: 3px solid var(--assistant-border);
  }
  .role {
    font-weight: 600;
    font-size: 0.8rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-bottom: 0.5rem;
  }
  .message.user .role { color: var(--user-border); }
  .message.assistant .role { color: var(--assistant-border); }
  .badge {
    background: var(--badge-bg);
    color: var(--fg);
    padding: 0.1em 0.4em;
    border-radius: 3px;
    font-size: 0.7rem;
    margin-left: 0.5em;
  }
  .timestamp {
    color: var(--timestamp-color);
    font-size: 0.75rem;
    font-family: monospace;
  }
  .content p { margin: 0.5em 0; }
  .content code {
    background: rgba(255,255,255,0.08);
    padding: 0.15em 0.35em;
    border-radius: 3px;
    font-family: 'SF Mono', 'Fira Code', monospace;
    font-size: 0.9em;
  }
  .tool-call {
    color: var(--tool-color);
    font-family: 'SF Mono', 'Fira Code', monospace;
    font-size: 0.85rem;
    margin: 0.4rem 0;
    padding: 0.3rem 0;
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
