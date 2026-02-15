# cclog

Export Claude Code session transcripts as readable HTML or Markdown documents, with automatic secret scrubbing and smart boundary detection.

## Features

- **HTML & Markdown export** — Dark terminal-themed HTML or clean GFM Markdown
- **Secret scrubbing** — Automatically detects and redacts secrets using gitleaks rules
- **Boundary detection** — Identifies logical conversation segments (session restarts, time gaps, history snapshots)
- **Gist publishing** — Push exports directly to GitHub gist
- **MCP server** — Register as a Claude Code MCP tool for conversational exports

## Installation

```bash
go install github.com/sgnl-ai/cclog/cmd/cclog@latest
```

## Usage

### List sessions

```bash
cclog list                            # list all recent sessions
cclog list --project /path/to/project # filter by project
```

### Export a session

```bash
cclog export                          # interactive session picker
cclog export --session <id>           # specific session by ID
cclog export --all                    # skip boundary detection
cclog export --format md              # Markdown output (default: html)
cclog export --gist                   # publish to gist
cclog export --gist --public          # public gist
```

Output is written to `~/cclog/{slug}-{date}.html` (or `.md`).

### MCP server setup

Register cclog as an MCP tool in Claude Code:

```bash
cclog setup
```

This runs `claude mcp add --transport stdio --scope user cclog -- cclog serve`.

Once registered, you can ask Claude:
- "Export this session as markdown"
- "List my recent sessions"
- "Export session abc123 as HTML and push to a gist"

### MCP tools

| Tool | Description |
|------|-------------|
| `export_transcript` | Export a session transcript. Params: `session_id`, `format`, `from_text`, `to_text`, `all`, `gist` |
| `list_sessions` | List available sessions. Params: `project`, `limit` |

## How it works

Claude Code stores session transcripts as JSONL files in `~/.claude/projects/`. Each line is a JSON object representing a message, tool call, or system event. cclog:

1. Discovers sessions from `sessions-index.json` files and JSONL filenames
2. Parses JSONL into structured messages (user text, assistant text, tool calls)
3. Strips internal tags (`<system-reminder>`, `<bash-input>`, etc.)
4. Detects logical boundaries (session restarts, time gaps, history snapshots)
5. Scans for secrets using gitleaks rule set and redacts them
6. Renders to self-contained HTML (dark theme) or clean Markdown

## Development

```bash
go test ./...                          # run all tests
go test ./... -coverprofile=cover.out  # coverage report
go build ./cmd/cclog/                  # build binary
```

## License

MIT
