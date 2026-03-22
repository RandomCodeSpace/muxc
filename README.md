# muxc

[![Release](https://img.shields.io/github/v/release/RandomCodeSpace/muxc?style=flat-square)](https://github.com/RandomCodeSpace/muxc/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/RandomCodeSpace/muxc?style=flat-square)](https://goreportcard.com/report/github.com/RandomCodeSpace/muxc)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/RandomCodeSpace/muxc?style=flat-square)](go.mod)

**Claude Multiplexer — session viewer and launcher for [Claude Code](https://docs.anthropic.com/en/docs/claude-code).**

List, inspect, and resume your Claude Code sessions from any terminal. muxc reads directly from Claude Code's native data — no extra config, no extra storage.

## Why muxc?

Claude Code stores your sessions but doesn't make them easy to manage across terminals. **muxc** fixes this:

- **Zero storage** — reads session data directly from `~/.claude/`, never writes its own files
- **Named sessions** — create and resume sessions by name across terminals
- **Session listing** — see all your Claude Code sessions at a glance with status and metadata
- **Resume by name** — `muxc myproject` picks up where you left off
- **Zero dependencies** — single static binary, no CGO, no runtime requirements

## Install

### Quick install (Linux / macOS)

```sh
curl -fsSL https://raw.githubusercontent.com/RandomCodeSpace/muxc/main/install.sh | sh
```

Or with `wget`:

```sh
wget -qO- https://raw.githubusercontent.com/RandomCodeSpace/muxc/main/install.sh | sh
```

### Go install

```sh
go install github.com/RandomCodeSpace/muxc@latest
```

### Download from releases

Download the binary for your platform from the [releases page](https://github.com/RandomCodeSpace/muxc/releases/latest), then:

```sh
chmod +x muxc-*
mv muxc-* ~/.local/bin/muxc
```

## Quick start

```sh
muxc myproject               # Create a new session (or resume if it exists)
# ... work with Claude, then press Ctrl-C or close the terminal ...
muxc myproject               # Resume the same conversation
muxc myproject -- --model opus  # Create with extra Claude flags
muxc ls                      # List all sessions
muxc info myproject          # Show session details
```

## Commands

| Command | Description |
|---------|-------------|
| `muxc <name> [-- <claude-args>]` | Resume session by name, or create if it doesn't exist |
| `muxc` | List sessions (same as `muxc ls`) |
| `muxc ls` | List sessions (`-s active` or `-s detached` to filter) |
| `muxc info <name>` | Show detailed session info |
| `muxc completion bash\|zsh\|fish` | Generate shell completions |
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

muxc is a **read-only** wrapper around Claude Code's native data:

- **Session names** come from the `--name` flag, which Claude Code stores as a `custom-title` record in `~/.claude/projects/`
- **Session IDs** are read from `~/.claude/sessions/{pid}.json` (written by Claude Code at startup)
- **Active/detached status** is computed by checking if the session's PID is still alive in the process table
- **Resume** uses `claude --resume <sessionId>` to reconnect to an existing conversation

muxc never writes files, sends signals, or modifies Claude Code's data. All side effects go through the `claude` CLI.

### Environment variables

| Variable | Description |
|----------|-------------|
| `MUXC_CLAUDE_BIN` | Path to the `claude` binary (default: auto-detected from `PATH`) |

## Shell completions

```sh
# bash
muxc completion bash > /etc/bash_completion.d/muxc

# zsh
muxc completion zsh > "${fpath[1]}/_muxc"

# fish
muxc completion fish > ~/.config/fish/completions/muxc.fish
```

## Building from source

```sh
git clone https://github.com/RandomCodeSpace/muxc.git
cd muxc
make build       # → ./muxc
make test        # run tests
make install     # install to ~/.local/bin/muxc
```

## License

[MIT](LICENSE)
