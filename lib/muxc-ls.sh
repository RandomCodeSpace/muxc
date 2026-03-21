#!/usr/bin/env bash
# muxc ls — list sessions

cmd_ls() {
    local filter_tag=""
    local filter_status=""
    local show_all=false

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --tag)
                [[ -n "${2:-}" ]] || die "--tag requires a value"
                filter_tag="$2"
                shift 2
                ;;
            --status)
                [[ -n "${2:-}" ]] || die "--status requires a value (active|detached|archived)"
                filter_status="$2"
                shift 2
                ;;
            --all|-a)
                show_all=true
                shift
                ;;
            *)
                die "Unknown option: $1"
                ;;
        esac
    done

    # Reap dead sessions first
    reap_dead_sessions

    # Collect sessions
    local count=0
    local rows=()

    for dir in "$MUXC_SESSIONS_DIR"/*/; do
        [[ -d "$dir" ]] || continue
        local meta_file="$dir/meta"
        [[ -f "$meta_file" ]] || continue

        local _name _status _cwd _accessed _tags_str _pid
        _name=$(basename "$dir")

        # Read meta fields
        _status=$(grep '^status=' "$meta_file" 2>/dev/null | cut -d= -f2-)
        _cwd=$(grep '^cwd=' "$meta_file" 2>/dev/null | cut -d= -f2-)
        _accessed=$(grep '^accessed_at=' "$meta_file" 2>/dev/null | cut -d= -f2-)
        _pid=$(grep '^claude_pid=' "$meta_file" 2>/dev/null | cut -d= -f2-)

        # Skip archived unless --all
        if [[ "$_status" == "archived" && "$show_all" == false && "$filter_status" != "archived" ]]; then
            continue
        fi

        # Filter by status
        if [[ -n "$filter_status" && "$_status" != "$filter_status" ]]; then
            continue
        fi

        # Read tags
        local tags_file="$dir/tags"
        _tags_str=""
        if [[ -f "$tags_file" && -s "$tags_file" ]]; then
            _tags_str=$(paste -sd, "$tags_file")
        fi

        # Filter by tag
        if [[ -n "$filter_tag" ]]; then
            if ! grep -qx "$filter_tag" "$tags_file" 2>/dev/null; then
                continue
            fi
        fi

        # Status emoji
        local status_icon
        case "$_status" in
            active)   status_icon="🟢" ;;
            detached) status_icon="🟡" ;;
            archived) status_icon="📦" ;;
            *)        status_icon="❓" ;;
        esac

        # Shorten cwd for display
        local short_cwd="${_cwd/#$HOME/~}"

        # Relative time
        local rel_time
        rel_time=$(relative_time "$_accessed")

        rows+=("$(printf "%-2s %-20s %-30s %-12s %s" "$status_icon" "$_name" "$short_cwd" "$rel_time" "${_tags_str:--}")")
        count=$((count + 1))
    done

    if [[ $count -eq 0 ]]; then
        info "No sessions found. Create one with: muxc new <name>"
        return 0
    fi

    echo "📋 Sessions:"
    echo ""
    printf "   %-2s %-20s %-30s %-12s %s\n" "" "NAME" "CWD" "ACCESSED" "TAGS"
    printf "   %-2s %-20s %-30s %-12s %s\n" "" "────" "───" "────────" "────"
    for row in "${rows[@]}"; do
        echo "   $row"
    done
    echo ""
    info "$count session(s)"
}
