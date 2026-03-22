# muxc

[![Release](https://img.shields.io/github/v/release/RandomCodeSpace/muxc?style=flat-square)](https://github.com/RandomCodeSpace/muxc/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/RandomCodeSpace/muxc?style=flat-square)](https://goreportcard.com/report/github.com/RandomCodeSpace/muxc)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/RandomCodeSpace/muxc?style=flat-square)](go.mod)

**Claude Multiplexer — persistent session manager for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) using tmux.**

Run Claude Code sessions in tmux. Detach, close your terminal, SSH from another device, and reattach to the same running session.

## Why muxc?

Claude Code runs in a single foreground terminal. Close it, and you have to manually resume. **muxc** fixes this:

- **Persistent sessions** — Claude runs inside tmux, survives terminal close and SSH disconnects
- **Detach / reattach** — `Ctrl-B d` to detach, `muxc myproject` from any terminal to reattach
- **Cross-device access** — SSH from another machine and reattach to a running session
- **Named sessions** — create and resume sessions by name
- **Session listing** — see all sessions at a glance with IDs and status
- **Zero storage** — reads session data from `~/.claude/`, never writes its own files

## Requirements

- **tmux** (any recent version) — `sudo apt install tmux` or `brew install tmux`
- **Claude Code** — installed and in PATH

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
muxc myproject               # Create a new session in tmux
# ... work with Claude ...
# Press Ctrl-B d to detach (Claude keeps running!)

muxc myproject               # Reattach from any terminal or device
muxc myproject -- --model opus  # Create with extra Claude flags
muxc ls                      # List all sessions (with session IDs)
muxc info myproject          # Show session details
muxc myproject:8c14          # Resume a specific session by ID prefix
muxc kill myproject          # Stop a running session
```

## Commands

| Command | Description |
|---------|-------------|
| `muxc <name> [-- <claude-args>]` | Create/attach to a tmux session |
| `muxc <name>:<id>` | Resume a specific session by name and ID prefix |
| `muxc` | List sessions (same as `muxc ls`) |
| `muxc ls` | List sessions with IDs (`-s active` or `-s detached` to filter) |
| `muxc info <name>` | Show detailed session info |
| `muxc kill <name>` | Kill a running tmux session |
| `muxc completion bash\|zsh\|fish` | Generate shell completions |
| `muxc version` | Print version |

### Flags

| Flag | Description |
|------|-------------|
| `--cwd <dir>` | Working directory for new session (default: current dir) |

### Status icons

| Icon | Status |
|------|--------|
| `▶` | Active — Claude is running in a tmux session |
| `⏸` | Detached — session data saved, can be resumed |

## How it works

muxc manages Claude Code sessions through **tmux**:

1. `muxc myproject` creates a tmux session named `muxc-myproject` running `claude --name myproject`
2. The Claude process runs inside tmux, independent of your terminal
3. Detach with `Ctrl-B d` — Claude keeps running in the background
4. Reattach from any terminal with `muxc myproject`
5. If the tmux session died but Claude Code saved the conversation, muxc resumes it with `claude --resume <sessionId>`

**Session data** is read from Claude Code's native `~/.claude/` directory:
- Session names from `custom-title` records in JSONL files
- Session IDs from `~/.claude/sessions/{pid}.json`
- Active status from tmux session state

muxc never writes its own files. All side effects go through `tmux` and `claude` CLI.

### Environment variables

| Variable | Description |
|----------|-------------|
| `MUXC_CLAUDE_BIN` | Path to the `claude` binary (default: auto-detected from `PATH`) |
| `MUXC_TMUX_BIN` | Path to the `tmux` binary (default: auto-detected from `PATH`) |

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
