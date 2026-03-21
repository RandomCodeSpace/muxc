#!/usr/bin/env bash
# muxc info — display full session details

cmd_info() {
    local name="${1:-}"
    [[ -n "$name" ]] || die "Usage: muxc info <name>"

    session_exists "$name" || die "Session \"$name\" not found"

    read_meta "$name"

    # Check PID liveness for accurate status
    if [[ "$status" == "active" && -n "$claude_pid" ]]; then
        if ! check_pid "$claude_pid"; then
            status="detached"
            claude_pid=""
            accessed_at="$(iso_now)"
            write_meta "$name"
            append_history "$name" "detached" "pid=$claude_pid (process died)"
        fi
    fi

    local status_icon
    case "$status" in
        active)   status_icon="🟢" ;;
        detached) status_icon="🟡" ;;
        archived) status_icon="📦" ;;
        *)        status_icon="❓" ;;
    esac

    local short_cwd="${cwd/#$HOME/~}"

    echo ""
    echo "ℹ️  Session: $name"
    echo "   ─────────────────────────────────────"
    echo "   Status:     $status_icon $status${claude_pid:+ (pid $claude_pid)}"
    echo "   Session ID: $session_id"
    echo "   Directory:  $short_cwd"
    echo "   Created:    $created_at ($(relative_time "$created_at"))"
    echo "   Accessed:   $accessed_at ($(relative_time "$accessed_at"))"

    # Show decoded claude args
    if [[ -n "$claude_args" ]]; then
        local decoded
        decoded=$(decode_claude_args "$claude_args")
        echo "   Claude args: $decoded"
    fi

    # Tags
    local dir
    dir="$(session_dir "$name")"
    local tags_file="$dir/tags"
    echo ""
    if [[ -f "$tags_file" && -s "$tags_file" ]]; then
        echo "🏷️  Tags: $(paste -sd', ' "$tags_file")"
    else
        echo "🏷️  Tags: (none)"
    fi

    # Notes
    local notes_file="$dir/notes"
    echo ""
    if [[ -f "$notes_file" && -s "$notes_file" ]]; then
        echo "📝 Notes:"
        sed 's/^/   /' "$notes_file"
    else
        echo "📝 Notes: (none)"
    fi

    # Recent history
    local history_file="$dir/history"
    echo ""
    if [[ -f "$history_file" && -s "$history_file" ]]; then
        echo "📜 Recent history:"
        tail -10 "$history_file" | while IFS=$'\t' read -r ts event details; do
            printf "   %-24s %-12s %s\n" "$ts" "$event" "$details"
        done
    fi
    echo ""
}
