#!/usr/bin/env bash
# muxc detach — manually mark a session as detached

cmd_detach() {
    local name="${1:-}"
    [[ -n "$name" ]] || die "Usage: muxc detach <name>"

    session_exists "$name" || die "Session \"$name\" not found"

    read_meta "$name"

    if [[ "$status" != "active" ]]; then
        warn "Session \"$name\" is not active (status: $status)"
        return 0
    fi

    # Kill the process if alive
    if [[ -n "$claude_pid" ]] && check_pid "$claude_pid"; then
        action "Sending SIGTERM to claude (pid $claude_pid)..."
        kill -TERM "$claude_pid" 2>/dev/null
        sleep 1
    fi

    local old_pid="$claude_pid"
    status="detached"
    claude_pid=""
    accessed_at="$(iso_now)"
    write_meta "$name"
    append_history "$name" "detached" "pid=$old_pid (manual)"

    success "Detached session \"$name\""
}
