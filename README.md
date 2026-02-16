# cclog

[![CI](https://github.com/SGNL-ai/cclog/actions/workflows/ci.yml/badge.svg)](https://github.com/SGNL-ai/cclog/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/SGNL-ai/cclog/graph/badge.svg)](https://codecov.io/gh/SGNL-ai/cclog)
[![Go Report Card](https://goreportcard.com/badge/github.com/sgnl-ai/cclog)](https://goreportcard.com/report/github.com/sgnl-ai/cclog)
[![Go Reference](https://pkg.go.dev/badge/github.com/sgnl-ai/cclog.svg)](https://pkg.go.dev/github.com/sgnl-ai/cclog)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/SGNL-ai/cclog)](go.mod)

Export Claude Code session transcripts as readable HTML or Markdown documents, with automatic secret scrubbing and smart boundary detection.

## Features

- **HTML & Markdown export** — Dark terminal-themed HTML or clean GFM Markdown
- **Secret scrubbing** — Automatically detects and redacts secrets using gitleaks rules
- **Boundary detection** — Identifies logical conversation segments (session restarts, time gaps, history snapshots)
- **Text boundaries** — Filter exports to specific message ranges via `--from-text` / `--to-text`
- **Prefix ID matching** — Use short session ID prefixes (e.g., `deadbeef` instead of the full UUID)
- **Gist publishing** — Push exports directly to GitHub gist
- **MCP server** — Register as a Claude Code MCP tool for conversational exports

## Installation

```bash
go install github.com/sgnl-ai/cclog/cmd/cclog@latest
```

Or download a binary from [Releases](https://github.com/SGNL-ai/cclog/releases).

## Usage

### List sessions

```bash
cclog list                            # list all recent sessions
cclog list --project /path/to/project # filter by project
```

### Export a session

```bash
cclog export                          # interactive session picker
cclog export --session <id>           # specific session by ID (or prefix)
cclog export --session dead           # prefix match works
cclog export --all                    # skip boundary detection
cclog export --format md              # Markdown output (default: html)
cclog export --from-text "fix the"    # start at message containing text
cclog export --to-text "looks good"   # end at message containing text
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
4. Applies text boundary filters if specified
5. Scans for secrets using gitleaks rule set and redacts them
6. Renders to self-contained HTML (dark theme) or clean Markdown

## Development

```bash
make all         # fmt + vet + lint + test + build
make test        # run tests with race detector
make coverage    # coverage report with 80% threshold
make lint        # golangci-lint
make build       # compile binary
make install     # go install
make security    # gosec + govulncheck
make clean       # remove artifacts
```

### Architecture

```
cmd/cclog/           CLI (cobra commands, interactive prompts)
internal/export/     Shared business logic (pipeline, formatting)
internal/parser/     JSONL parser, tag stripping, tool descriptions
internal/renderer/   HTML and Markdown renderers
internal/scanner/    Secret redaction via gitleaks
internal/session/    Session discovery, boundary detection
internal/gist/       GitHub gist publishing
internal/mcp/        MCP server (stdio transport)
```

## License

MIT
