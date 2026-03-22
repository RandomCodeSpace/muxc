# muxc

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)

**Claude Multiplexer — session viewer and launcher for [Claude Code](https://docs.anthropic.com/en/docs/claude-code).**

List, inspect, and resume your Claude Code sessions from any terminal. muxc reads directly from Claude Code's native data — no extra config, no extra storage. Single bash script, zero compilation.

## Why muxc?

- **Zero storage** — reads session data directly from `~/.claude/`, never writes its own state
- **Named sessions** — create and resume sessions by name across terminals
- **Session listing** — see all your Claude Code sessions at a glance with status, IDs, and metadata
- **Resume by name** — `muxc myproject` picks up where you left off
- **Single file** — one bash script, no compilation, no runtime dependencies
- **jq auto-install** — jq is bootstrapped automatically if not already present

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/RandomCodeSpace/muxc/main/install.sh | sh
```

Or directly:

```sh
curl -fsSL https://raw.githubusercontent.com/RandomCodeSpace/muxc/main/muxc | install -m 755 /dev/stdin ~/.local/bin/muxc
```

## Quick start

```sh
muxc myproject               # Create a new session (or resume if it exists)
# ... work with Claude, then press Ctrl-C or close the terminal ...
muxc myproject               # Resume the same conversation
muxc myproject -- --model opus  # Create with extra Claude flags
muxc ls                      # List all sessions (with session IDs)
muxc info myproject          # Show session details
muxc myproject:8c14          # Resume a specific session by ID prefix
```

## Commands

| Command | Description |
|---------|-------------|
| `muxc <name> [-- <claude-args>]` | Resume session by name, or create if it doesn't exist |
| `muxc <name>:<id>` | Resume a specific session by name and ID prefix |
| `muxc` | List sessions (same as `muxc ls`) |
| `muxc ls` | List sessions with IDs (`-s active` or `-s detached` to filter) |
| `muxc list` / `muxc l` | Aliases for `muxc ls` |
| `muxc info <name>` | Show detailed session info |
| `muxc version` | Print version |

### Flags

| Flag | Description |
|------|-------------|
| `--cwd <dir>` | Working directory for new session (default: current dir) |

### Status icons

| Icon | Status |
|------|--------|
| `▶` | Active — Claude process is running |
| `⏸` | Detached — session saved, process stopped |

## How it works

muxc is a **read-only** bash script that wraps Claude Code's native data:

- **Session names** come from the `--name` flag, which Claude Code stores as a `custom-title` record in `~/.claude/projects/`
- **Session IDs** are read from `~/.claude/sessions/{pid}.json` (written by Claude Code at startup)
- **Active/detached status** is computed by checking if the session's PID is still alive
- **Create** runs claude as a foreground subprocess (captures session ID for future resume)
- **Resume** uses `exec claude --resume <sessionId>` for perfect TTY passthrough

### Per-session args

When you create a session with extra args (`muxc myproject -- --model opus`), those args are saved to `~/.config/muxc.conf` and automatically replayed on resume.

### Environment variables

| Variable | Description |
|----------|-------------|
| `MUXC_CLAUDE_BIN` | Path to the `claude` binary (default: auto-detected from `PATH`) |

## License

[MIT](LICENSE)
