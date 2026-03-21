# muxc

[![Release](https://img.shields.io/github/v/release/RandomCodeSpace/muxc?style=flat-square)](https://github.com/RandomCodeSpace/muxc/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/RandomCodeSpace/muxc?style=flat-square)](https://goreportcard.com/report/github.com/RandomCodeSpace/muxc)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/RandomCodeSpace/muxc?style=flat-square)](go.mod)

**Claude Multiplexer — session manager for [Claude Code](https://docs.anthropic.com/en/docs/claude-code).**

Run multiple Claude Code sessions side-by-side, detach them to the background, and reattach later. Sessions persist in a local SQLite database so you never lose context even after reboots.

## Why muxc?

Claude Code runs in a single foreground terminal. If you close it, reconnecting to the same conversation is manual and fragile. **muxc** fixes this:

- **Multiple sessions** — work on different projects or tasks in parallel, each in its own named session
- **Detach / reattach** — send a session to the background and pick it up later, from any terminal
- **Session metadata** — tag, annotate, and filter sessions so you can find what you need
- **Import orphans** — discover and adopt Claude Code sessions that were started outside muxc
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

### Manual download

Download the binary for your platform from the [releases page](https://github.com/RandomCodeSpace/muxc/releases/latest), then:

```sh
chmod +x muxc-*
mv muxc-* ~/.local/bin/muxc
```

## Quick start

```sh
muxc new myproject           # Create and launch a new session
# ... work with Claude, then press Ctrl-C or close the terminal ...
muxc attach myproject        # Reattach to the same conversation
muxc myproject               # Shortcut — same as attach
```

## Commands

| Command | Description |
|---------|-------------|
| `muxc new <name> [-- <claude-args>]` | Create and launch a new Claude session |
| `muxc attach [<name>]` / `muxc <name>` | Attach to an existing session (interactive picker if no name) |
| `muxc detach <name>` | Detach an active session to the background |
| `muxc kill <name>` | Kill a session's Claude process (`-f` for SIGKILL) |
| `muxc ls` | List sessions (`-s status`, `-t tag`, `-a` for archived) |
| `muxc info <name>` | Show detailed session info and history |
| `muxc tag <name> add\|rm <tag>` | Add or remove tags on a session |
| `muxc note <name> [text]` | Set or edit session notes |
| `muxc rename <old> <new>` | Rename a session |
| `muxc archive <name>` | Archive a session |
| `muxc rm <name>` | Remove a session (`-f` to force-kill first) |
| `muxc import` | Adopt orphaned Claude Code sessions (`--scan` to preview) |
| `muxc completion bash\|zsh\|fish` | Generate shell completions |
| `muxc version` | Print version |

### Session lifecycle

```
  new ──▶ active ──▶ detach ──▶ detached ──▶ attach ──▶ active
                       │                        │
                       ▼                        ▼
                     kill                    archive / rm
```

### Status icons

| Icon | Status |
|------|--------|
| `▶` | Active — Claude process is running |
| `⏸` | Detached — session saved, process stopped |
| `◼` | Archived — session preserved but hidden from default list |

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
