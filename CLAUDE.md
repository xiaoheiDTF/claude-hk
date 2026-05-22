# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Claude Code configuration and documentation project. The repo provides a hooks system, custom skills, Chinese-translated Claude Code documentation, and a Go-based Claude Code API traffic proxy (`claude_tap_plus/`).

## Architecture

### Hooks Pipeline

**Two-layer: `base.sh` (dispatcher) + sibling scripts (business logic)**

`settings.json` registers all 29 Claude Code lifecycle events (`SessionStart` through `SessionEnd`), each pointing to `hooks/XX-event-name/base.sh`. Each `base.sh` sources the shared `hooks/base.sh` (stdin JSON, `json_get()` parsing, structured logging, `hook_output()`), then calls sibling `.sh` scripts.

**Hooks with dispatch logic:**
- `01-session-start` — first run triggers `init.sh`; every session checks UTF-8/Python/dirs/skill health
- `03-user-prompt-submit` — calls `skill-inject.sh` to detect `/skill-name` and inject context
- `05-pre-tool-use` — two-layer interception: `enforce_boundary.sh` (tool whitelist) + `dispatch_to_skill` (skill-level)
- `16-stop` — runs active skill's `16Stop.sh` for cleanup; `skill-register.sh` auto-registers new skills
- `02`, `04`, `06`-`15`, `17`-`29` — logging only (forward to active skill via `dispatch_to_skill`)

**Shared infrastructure** (`hooks/base.sh`):
- `json_get()` — jq first, fallback `json_get.py` via Python, final sed fallback
- `dispatch_to_skill(event_num)` — reads `.active` for current session's skill, calls `skills/<skill>/scripts/<EventName>.sh`
- `hook_output(exit_code, json)` — wraps exit code and result JSON

**Platform** (`hooks/platform.sh`): `uname -s` → `OS_TYPE`; Python prefers embedded `.claude/localLanguage/python/python.exe`.

### Skill Lifecycle

Skills are functional units triggered by `/skill-name`. Each has a full lifecycle:

```
User types /skill-name
  → 03 skill-inject.sh matches registry.conf → runs 03UserPromptSubmit.sh (context injection)
    → writes .active (session_id|skill_name)
      → events 05-29 forwarded to skill scripts via dispatch_to_skill()
        → event 16 runs 16Stop.sh → removes .active entry
```

**Skill directory structure and script convention:**

```
XXX-skill-name/
├── SKILL.md                           # Definition (frontmatter: name/description/allowed-tools)
└── scripts/
    ├── 03UserPromptSubmit.sh          # Context injection (runs when /skill-name triggered)
    ├── 16Stop.sh                      # Cleanup after response (removes .active entry)
    ├── init.sh                        # Project first-time init (optional)
    └── init_check.sh                  # Per-session environment check (optional)
```

Script names follow hook event numbers: `02Setup.sh`, `05PreToolUse.sh`, `08PostToolUse.sh`, etc. `dispatch_to_skill()` in `hooks/base.sh` maps numbers to script names.

**Adding a new skill:**
1. Create `skills/XXX-name/` with `SKILL.md`, `scripts/03UserPromptSubmit.sh`, `scripts/16Stop.sh`
2. `skill-register.sh` (runs on `16-stop` event) auto-scans and registers into `registry.conf`; manual add also works
3. Every `.sh` script must set `SKILL_TAG="xxx-name"` before sourcing `log.sh`

**Key shared modules:**
- `skills/active.sh` — `.active` file CRUD (`active_add/get/remove`), uses `lock.sh` for file locking
- `skills/log.sh` — dual-write logging: unified `hooks/logs/<date>.log` + module `skills/log/<tag>/<date>.log`
- `skills/lock.sh` — cross-platform file lock via `mkdir` atomic operation (`lock_acquire/release`)
- `skills/enforce_boundary.sh` — parses `SKILL.md` frontmatter `allowed-tools`, blocks unauthorized tool calls in `05-pre-tool-use`

### 003 Issue Workflow

The 003 series is a complete issue-driven development pipeline:

```
/003-1-issue-init → /003-2-issue → /003-3-issue-discuss
                                       |
/003-9-issue-review <- /003-8-issue-test <- /003-7-issue-pr <- /003-6-issue-done <- /003-5-issue-fix <- /003-4-issue-claim
```

- `/003-1-issue-init` — initialize labels (one-time), reads `labels.conf`
- `/003-4-issue-claim` — atomic claim (`in-progress` label), file lock prevents multi-agent conflicts
- `/003-5-issue-fix` — generates branch name from issue labels (bug→fix, enhancement→feat), creates branch
- `/003-7-issue-pr` — creates PR; **must include `## Test plan` section** (each item as `- [ ]` checkbox)
- `/003-8-issue-test` — executes Test Plan items, checks off `- [x]`
- `/003-9-issue-review merge/reject` — **blocks merge if any Test Plan item is unchecked `[ ]`**

### Initialization Flow

First session: `01-session-start/base.sh` detects missing `.initialized` → runs `init.sh`:
1. Reads `dirs.conf` to create directories
2. Detects platform + installs Python (downloads embeddable version on Windows)
3. Configures UTF-8 in `~/.bashrc` (with idempotent marker guards)
4. Runs each skill's `init.sh`
5. Writes `.initialized` marker

Every session: re-checks UTF-8, Python, directory integrity, and each skill's `init_check.sh`.

### Skills Summary

| Skill | Output Directory | Purpose |
|-------|-----------------|---------|
| `001-testcode-python` | `doc/testcode/python/{api,other}/` | Python test scripts and utilities |
| `002-otherdoc` | `doc/otherDoc/YYYY-MM-DD/` | General documentation by date |
| `003-1-issue-init` | — | Initialize issue label system (one-time) |
| `003-2-issue` | `doc/issues/{drafts,templates}/` | Create GitHub Issue |
| `003-3-issue-discuss` | — | Pull issue content for discussion |
| `003-4-issue-claim` | — | Atomically claim issue |
| `003-5-issue-fix` | — | Create branch from issue and start development |
| `003-6-issue-done` | — | Mark development complete, ready for PR |
| `003-7-issue-pr` | — | Create PR linked to issue |
| `003-8-issue-test` | — | Execute PR's Test Plan |
| `003-9-issue-review` | — | Review PR: merge or reject |
| `004-git-push` | — | Commit (grouped, Chinese messages) + push |
| `005-git-commit` | — | Commit only (grouped, Chinese messages), no push |

## claude_tap_plus (Go Project)

A Go rewrite of claude-tap — a reverse proxy that intercepts Claude Code API traffic, records JSONL traces, and prints token usage summaries.

**Build & Run:**
```bash
cd claude_tap_plus
go build -o claude-tap-plus ./cmd/claude-tap
go run ./cmd/claude-tap claude          # Run directly
```

**Architecture:**
- `cmd/claude-tap/main.go` — CLI entry: flag parsing, proxy lifecycle, child process management, session sync subcommands
- `internal/config/` — Client config, upstream URL detection, `~/.claude.json` reader
- `internal/proxy/` — Reverse proxy: request forwarding, header redaction, path whitelist, SSE/non-SSE handling
- `internal/sse/` — SSE byte stream reassembler → complete API response reconstruction
- `internal/trace/` — JSONL trace writer with token statistics (Anthropic-specific fields)
- `internal/usage/` — Multi-provider token field normalization (Anthropic/OpenAI/Gemini)
- `internal/session/` — Session push/pull/status: collect `~/.claude/` sessions to local storage and restore

**Data flow:** CLI parses flags → detects upstream URL → starts local proxy → intercepts requests → SSE reassembler for streaming → JSONL trace per API call → exit summary (API calls, token counts).

**Testing:**
```bash
cd claude_tap_plus
go test ./...                           # All tests
go test ./internal/sse/...              # Specific package
```

## Git Commit Convention (Skills 004/005)

All commits use Chinese: `<type>: <主描述>` with `- 具体修改描述` sub-items.

Types: fix/feat/update/style/refactor/perf/test/docs/revert/build/chore

Grouping priority: type → directory/module → functional association → impact scope. Never `git add .` or `git add -A` — always add specific files by group.


# CLAUDE.md

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

---

**These guidelines are working if:** fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and clarifying questions come before implementation rather than after mistakes.
