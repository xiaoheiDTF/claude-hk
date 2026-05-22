# claude-tap-plus

Go rewrite of [claude-tap](https://github.com/liaohch3/claude-tap) — a lightweight reverse proxy that intercepts AI coding agent API traffic, records JSONL traces, and prints token usage summaries.

## Features

- **Zero-config proxy** — auto-detects upstream API URL from `ANTHROPIC_BASE_URL` or `~/.claude.json`
- **SSE stream reconstruction** — reassembles streaming responses into complete API objects
- **Multi-provider support** — Anthropic, OpenAI, Google Gemini token field normalization
- **JSONL trace output** — one JSON record per API call, organized by date
- **Sensitive header redaction** — API keys and tokens are masked in trace files

## Quick Start

### Prerequisites

- Go 1.23+
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) CLI installed (`claude` in PATH)

### Build

```bash
cd claude_tap_plus
go build -o claude-tap-plus ./cmd/claude-tap
```

### Run

```bash
# Basic usage — wraps Claude Code with tracing
claude-tap-plus claude

# With options
claude-tap-plus \
  --tap-target=https://api.anthropic.com \
  --tap-port=8080 \
  --tap-output-dir=.traces \
  --tap-verbose \
  claude
```

Everything after the `--tap-*` flags is passed through to `claude`.

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--tap-target` | auto-detect | Upstream API base URL |
| `--tap-port` | `0` (random) | Local proxy port (`0` = random available) |
| `--tap-output-dir` | `.traces` | Directory for JSONL trace files |
| `--tap-verbose` | `false` | Print debug logs |

### Output Example

```
🚀 Starting Claude Code: claude
   ANTHROPIC_BASE_URL=http://127.0.0.1:52341

proxy listening on http://127.0.0.1:52341

⏳ Shutting down Claude Code... (Ctrl+C again to force)

📋 Claude Code exited
   API calls:      12
   Input tokens:   45230
   Output tokens:  8192
   Cache read:     32000
   Cache create:   5000
   Trace: .traces/2026-05-19/trace_143052.jsonl
```

## Architecture

```
cmd/claude-tap/main.go          CLI entry — flag parsing, proxy lifecycle, child process management
internal/
├── config/
│   ├── client.go               Client config (binary name, env vars, upstream detection)
│   └── claudeconfig.go         ~/.claude.json reader
├── proxy/
│   ├── reverse.go              Reverse proxy — request forwarding, SSE/non-SSE handling
│   ├── headers.go              Header filtering and sensitive value redaction
│   ├── paths.go                Allowed API path whitelist
│   └── netutil.go              Cross-platform network utilities
├── sse/
│   └── reassembler.go          SSE byte stream parser → complete API response reconstruction
├── trace/
│   └── writer.go               JSONL trace writer with token statistics
└── usage/
    └── normalize.go            Multi-provider token field normalization
```

### Data Flow

1. CLI parses flags, detects upstream URL, starts local proxy
2. Proxy intercepts requests, forwards to upstream API
3. For streaming responses, `SSEReassembler` reassembles SSE chunks into a complete response object
4. Each request/response pair is recorded as one JSONL line via `TraceWriter`
5. On exit, a usage summary is printed (API calls, token counts by category)

## Supported API Paths

The proxy only forwards recognized paths and rejects everything else (scanners, crawlers) with 404:

- `/v1/messages` — Anthropic Messages API
- `/v1/responses` — OpenAI Responses API
- `/v1/chat/completions` — OpenAI Chat Completions
- `/v1/models`, `/v1/embeddings`, `/v1/files` — OpenAI utilities
- `/v1beta/`, `/v1/` — Google Gemini API
- `/coding/v1/` — Kimi API

## Development

```bash
# Tidy dependencies
go mod tidy

# Run directly without building
go run ./cmd/claude-tap claude

# Format
gofmt -w .
```
