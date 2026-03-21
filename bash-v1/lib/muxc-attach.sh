#!/usr/bin/env bash
# muxc attach — re-attach to a detached session

cmd_attach() {
    local name="${1:-}"
    [[ -n "$name" ]] || die "Usage: muxc attach <name>"

    session_exists "$name" || die "Session \"$name\" not found"

    read_meta "$name"

    # Check if session is already active
    if [[ "$status" == "active" && -n "$claude_pid" ]]; then
        if check_pid "$claude_pid"; then
            die "Session \"$name\" is already active (pid $claude_pid). Use 'muxc kill $name' first."
        fi
        # PID is dead — transition to detached
        local dead_pid="$claude_pid"
        status="detached"
        claude_pid=""
        write_meta "$name"
        append_history "$name" "detached" "pid=$dead_pid (process died)"
    fi

    if [[ "$status" == "archived" ]]; then
        info "Unarchiving session \"$name\""
        append_history "$name" "unarchived" ""
    fi

    # Lock to prevent double-attach
    local dir
    dir="$(session_dir "$name")"
    local lock_file="$dir/.lock"

    (
        if ! flock -n 9; then
            die "Session \"$name\" is locked by another process"
        fi

        # Update meta
        status="active"
        claude_pid="$$"
        accessed_at="$(iso_now)"
        write_meta "$name"
    ) 9>"$lock_file" || exit 1

    append_history "$name" "attached" "pid=$$"

    action "Attaching to \"$name\"..."

    # Navigate to session's working directory
    if [[ -d "$cwd" ]]; then
        nav "Navigating to $cwd"
        cd "$cwd" || die "Failed to cd to $cwd"
    else
        warn "Session directory $cwd no longer exists. Staying in $PWD"
    fi

    # Build claude command with stored args
    local claude_bin
    claude_bin=$(get_claude_bin)

    local claude_cmd=("$claude_bin" "--resume" "$session_id")

    # Decode and append stored claude args
    if [[ -n "$claude_args" ]]; then
        local decoded
        decoded=$(decode_claude_args "$claude_args")
        # Split decoded args back into array
        read -ra extra_args <<< "$decoded"
        claude_cmd+=("${extra_args[@]}")
    fi

    launch "Launching Claude Code..."
    exec "${claude_cmd[@]}"
}
