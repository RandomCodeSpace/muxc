#!/usr/bin/env bash
# muxc help — display usage information

cmd_help() {
    local cmd="${1:-}"

    case "$cmd" in
        new|n)
            cat <<'EOF'
✨ muxc new — Create a new named session

Usage: muxc new <name> [--cwd <dir>] [--tag <t>...] [-- <claude-args>...]

Options:
  --cwd <dir>     Working directory (default: current directory)
  --tag <tag>     Add a tag (repeatable)
  --              Everything after this is passed to claude

Examples:
  muxc new my-feature
  muxc new backend --cwd ~/git/api --tag urgent
  muxc new dev -- --dangerously-skip-permissions --model opus
EOF
            ;;
        attach|a)
            cat <<'EOF'
🔗 muxc attach — Re-attach to a detached session

Usage: muxc attach <name>

Automatically navigates to the session's stored directory and resumes
the Claude Code session with the original flags.

Aliases: a
EOF
            ;;
        detach|d)
            cat <<'EOF'
🔌 muxc detach — Detach from an active session

Usage: muxc detach <name>

Sends SIGTERM to the claude process and marks the session as detached.
Note: closing the terminal or Ctrl-C also detaches (detected lazily).

Aliases: d
EOF
            ;;
        ls|list|l)
            cat <<'EOF'
📋 muxc ls — List sessions

Usage: muxc ls [--tag <t>] [--status <s>] [--all]

Options:
  --tag <tag>       Filter by tag
  --status <s>      Filter by status (active|detached|archived)
  --all, -a         Show all sessions including archived

Aliases: list, l
EOF
            ;;
        info|i)
            cat <<'EOF'
ℹ️  muxc info — Show full session details

Usage: muxc info <name>

Displays metadata, tags, notes, and recent history.

Aliases: i
EOF
            ;;
        tag|t)
            cat <<'EOF'
🏷️  muxc tag — Manage session tags

Usage: muxc tag <name> add <tag>
       muxc tag <name> rm <tag>

Aliases: t
EOF
            ;;
        note)
            cat <<'EOF'
📝 muxc note — Add or edit session notes

Usage: muxc note <name> [<text>]

With text: appends to notes file.
Without text: opens $EDITOR on the notes file.
EOF
            ;;
        rename|mv)
            cat <<'EOF'
✏️  muxc rename — Rename a session

Usage: muxc rename <old-name> <new-name>

Cannot rename active sessions. Kill first.

Aliases: mv
EOF
            ;;
        archive)
            cat <<'EOF'
📦 muxc archive — Archive a session

Usage: muxc archive <name>

Archived sessions are hidden from 'muxc ls' unless --all is used.
Can be re-activated via 'muxc attach'.
EOF
            ;;
        kill|k)
            cat <<'EOF'
💀 muxc kill — Stop a running claude session

Usage: muxc kill <name> [--force]

Options:
  --force, -f    Use SIGKILL instead of SIGTERM

Aliases: k
EOF
            ;;
        rm)
            cat <<'EOF'
🗑️  muxc rm — Delete a session

Usage: muxc rm <name> [--force]

Options:
  --force, -f    Kill active session and remove

Deletes all session metadata (meta, tags, notes, history).
Does NOT affect the Claude Code session itself.
EOF
            ;;
        import)
            cat <<'EOF'
📥 muxc import — Adopt orphaned Claude Code sessions

Usage: muxc import [--scan]

Options:
  --scan    Only show orphaned sessions, don't adopt them

Scans ~/.claude/sessions/ for sessions not tracked by muxc
and offers to adopt them with a name you choose.
EOF
            ;;
        "")
            cat <<'EOF'
🔧 muxc — tmux-like session manager for Claude Code

Usage: muxc <command> [options]

Session Commands:
  new, n        Create a new named session and attach
  attach, a     Re-attach to a detached session
  detach, d     Detach from an active session
  ls, l         List sessions (default command)
  kill, k       Stop a running session
  rm            Delete a session

Organization:
  info, i       Show full session details
  tag, t        Manage session tags
  note          Add or edit session notes
  rename, mv    Rename a session
  archive       Archive a session

Utilities:
  import        Adopt orphaned Claude Code sessions
  help, h       Show this help
  version       Show version

Run 'muxc help <command>' for details on a specific command.
EOF
            ;;
        *)
            die "Unknown command: $cmd. Run 'muxc help' for usage."
            ;;
    esac
}
