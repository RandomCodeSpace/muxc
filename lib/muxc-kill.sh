#!/usr/bin/env bash
# muxc kill — stop a running claude session

cmd_kill() {
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

    [[ -n "$name" ]] || die "Usage: muxc kill <name> [--force]"
    session_exists "$name" || die "Session \"$name\" not found"

    read_meta "$name"

    if [[ "$status" != "active" || -z "$claude_pid" ]]; then
        warn "Session \"$name\" is not active (status: $status)"
        return 0
    fi

    if ! check_pid "$claude_pid"; then
        info "Process already dead. Updating status."
        status="detached"
        claude_pid=""
        accessed_at="$(iso_now)"
        write_meta "$name"
        append_history "$name" "detached" "pid=$claude_pid (already dead)"
        return 0
    fi

    local signal="TERM"
    if $force; then
        signal="KILL"
    fi

    action "Sending SIG$signal to claude (pid $claude_pid)..."
    kill -"$signal" "$claude_pid" 2>/dev/null

    # Wait briefly for process to die
    local waited=0
    while check_pid "$claude_pid" && [[ $waited -lt 5 ]]; do
        sleep 1
        waited=$((waited + 1))
    done

    if check_pid "$claude_pid"; then
        if ! $force; then
            warn "Process still alive. Try: muxc kill $name --force"
            return 1
        fi
    fi

    local old_pid="$claude_pid"
    status="detached"
    claude_pid=""
    accessed_at="$(iso_now)"
    write_meta "$name"
    append_history "$name" "killed" "pid=$old_pid signal=$signal"

    success "Killed session \"$name\" (pid $old_pid)"
}
