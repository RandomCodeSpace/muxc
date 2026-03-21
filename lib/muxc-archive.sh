#!/usr/bin/env bash
# muxc archive — archive a session (hide from default ls)

cmd_archive() {
    local name="${1:-}"
    [[ -n "$name" ]] || die "Usage: muxc archive <name>"

    session_exists "$name" || die "Session \"$name\" not found"

    read_meta "$name"

    if [[ "$status" == "archived" ]]; then
        warn "Session \"$name\" is already archived"
        return 0
    fi

    # Kill if active
    if [[ "$status" == "active" && -n "$claude_pid" ]]; then
        if check_pid "$claude_pid"; then
            warn "Killing active session before archiving..."
            kill -TERM "$claude_pid" 2>/dev/null
            sleep 1
        fi
        claude_pid=""
    fi

    status="archived"
    accessed_at="$(iso_now)"
    write_meta "$name"
    append_history "$name" "archived" ""

    success "Archived session \"$name\""
}
