#!/usr/bin/env bash
# muxc rm — delete a session's metadata

cmd_rm() {
    local name=""
    local force=false

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force|-f) force=true; shift ;;
            -*) die "Unknown option: $1" ;;
            *)
                [[ -z "$name" ]] || die "Unexpected argument: $1"
                name="$1"; shift
                ;;
        esac
    done

    [[ -n "$name" ]] || die "Usage: muxc rm <name> [--force]"
    session_exists "$name" || die "Session \"$name\" not found"

    read_meta "$name"

    # If active, refuse unless --force
    if [[ "$status" == "active" && -n "$claude_pid" ]]; then
        if check_pid "$claude_pid"; then
            if $force; then
                warn "Killing active session first..."
                kill -TERM "$claude_pid" 2>/dev/null
                sleep 1
                kill -KILL "$claude_pid" 2>/dev/null
            else
                die "Session \"$name\" is active (pid $claude_pid). Use --force to kill and remove, or 'muxc kill $name' first."
            fi
        fi
    fi

    local dir
    dir="$(session_dir "$name")"
    rm -rf "$dir"

    success "Removed session \"$name\""
}
