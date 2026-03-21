#!/usr/bin/env bash
# muxc rename — rename a session

cmd_rename() {
    local old_name="${1:-}"
    local new_name="${2:-}"

    [[ -n "$old_name" && -n "$new_name" ]] || die "Usage: muxc rename <old-name> <new-name>"

    session_exists "$old_name" || die "Session \"$old_name\" not found"
    validate_name "$new_name"

    if session_exists "$new_name"; then
        die "Session \"$new_name\" already exists"
    fi

    read_meta "$old_name"

    # Refuse to rename active sessions
    if [[ "$status" == "active" && -n "$claude_pid" ]]; then
        if check_pid "$claude_pid"; then
            die "Cannot rename active session \"$old_name\" (pid $claude_pid). Kill it first."
        fi
    fi

    local old_dir new_dir
    old_dir="$(session_dir "$old_name")"
    new_dir="$(session_dir "$new_name")"

    mv "$old_dir" "$new_dir"
    append_history "$new_name" "renamed" "from=$old_name"

    success "Renamed \"$old_name\" → \"$new_name\""
}
