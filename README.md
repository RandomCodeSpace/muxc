# muxc

tmux-like session manager for [Claude Code](https://docs.anthropic.com/en/docs/claude-code).

Create, attach, detach, and manage multiple Claude Code sessions from your terminal.

## Install

### Quick install (Linux / macOS)

```sh
curl -fsSL https://raw.githubusercontent.com/randomcodespace/muxc/main/install.sh | sh
```

Or with `wget`:

```sh
wget -qO- https://raw.githubusercontent.com/randomcodespace/muxc/main/install.sh | sh
```

### Go install

```sh
go install github.com/RandomCodeSpace/muxc@latest
```

### Manual download

Download the binary for your platform from the [releases page](https://github.com/randomcodespace/muxc/releases/latest), then:

```sh
chmod +x muxc-*
sudo mv muxc-* /usr/local/bin/muxc
```

## Usage

```sh
muxc new myproject              # Create a new session
muxc attach myproject           # Attach to a session
muxc attach -f myproject        # Force reattach (detaches existing)
muxc detach myproject           # Detach a session
muxc ls                         # List all sessions
muxc kill myproject             # Kill a session
```

Run `muxc --help` for all commands.

## Shell completions

```sh
# bash
muxc completion bash > /etc/bash_completion.d/muxc

# zsh
muxc completion zsh > "${fpath[1]}/_muxc"

# fish
muxc completion fish > ~/.config/fish/completions/muxc.fish
```

## License

MIT
