#!/usr/bin/env bash
# muxc import — adopt orphaned Claude Code sessions

cmd_import() {
    local scan_only=false

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --scan) scan_only=true; shift ;;
            *) die "Unknown option: $1" ;;
        esac
    done

    local claude_sessions_dir="$HOME/.claude/sessions"
    if [[ ! -d "$claude_sessions_dir" ]]; then
        die "Claude sessions directory not found: $claude_sessions_dir"
    fi

    echo "🔍 Scanning for Claude Code sessions..."
    echo ""

    # Collect known session IDs from muxc
    local known_ids=()
    for dir in "$MUXC_SESSIONS_DIR"/*/; do
        [[ -d "$dir" ]] || continue
        local meta_file="$dir/meta"
        [[ -f "$meta_file" ]] || continue
        local sid
        sid=$(grep '^session_id=' "$meta_file" 2>/dev/null | cut -d= -f2-)
        [[ -n "$sid" ]] && known_ids+=("$sid")
    done

    # Scan Claude sessions
    local orphan_count=0
    for session_file in "$claude_sessions_dir"/*.json; do
        [[ -f "$session_file" ]] || continue

        # Parse JSON manually (no jq dependency)
        local sid cwd started_at pid_str
        sid=$(grep -o '"sessionId"[[:space:]]*:[[:space:]]*"[^"]*"' "$session_file" 2>/dev/null | head -1 | sed 's/.*"sessionId"[[:space:]]*:[[:space:]]*"//;s/"//')
        cwd=$(grep -o '"cwd"[[:space:]]*:[[:space:]]*"[^"]*"' "$session_file" 2>/dev/null | head -1 | sed 's/.*"cwd"[[:space:]]*:[[:space:]]*"//;s/"//')
        started_at=$(grep -o '"startedAt"[[:space:]]*:[[:space:]]*[0-9]*' "$session_file" 2>/dev/null | head -1 | sed 's/.*"startedAt"[[:space:]]*:[[:space:]]*//')
        pid_str=$(grep -o '"pid"[[:space:]]*:[[:space:]]*[0-9]*' "$session_file" 2>/dev/null | head -1 | sed 's/.*"pid"[[:space:]]*:[[:space:]]*//')

        [[ -n "$sid" ]] || continue

        # Check if already known
        local is_known=false
        for known in "${known_ids[@]}"; do
            if [[ "$known" == "$sid" ]]; then
                is_known=true
                break
            fi
        done
        $is_known && continue

        orphan_count=$((orphan_count + 1))
        local short_cwd="${cwd/#$HOME/~}"

        # Convert epoch ms to readable
        local time_str="unknown"
        if [[ -n "$started_at" ]]; then
            local epoch_sec=$((started_at / 1000))
            time_str=$(date -d "@$epoch_sec" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo "unknown")
        fi

        # Check if process alive
        local alive="dead"
        if [[ -n "$pid_str" ]] && check_pid "$pid_str"; then
            alive="alive"
        fi

        echo "   📥 Session: ${sid:0:8}..."
        echo "      Directory: $short_cwd"
        echo "      Started:   $time_str"
        echo "      Process:   ${pid_str:-unknown} ($alive)"
        echo ""

        if ! $scan_only; then
            # Prompt for name
            local suggested_name
            suggested_name=$(basename "$cwd" 2>/dev/null | tr '[:upper:]' '[:lower:]' | tr -c 'a-z0-9_-' '-' | head -c 64)

            echo -n "   Name for this session [$suggested_name]: "
            read -r user_name < /dev/tty
            local import_name="${user_name:-$suggested_name}"

            validate_name "$import_name" 2>/dev/null || {
                warn "Invalid name, skipping"
                continue
            }

            if session_exists "$import_name"; then
                warn "Session \"$import_name\" already exists, skipping"
                continue
            fi

            # Create muxc session
            local import_dir
            import_dir="$(session_dir "$import_name")"
            mkdir -p "$import_dir"

            local import_status="detached"
            if [[ "$alive" == "alive" ]]; then
                import_status="active"
            fi

            # Write meta
            session_id="$sid"
            claude_pid="${pid_str:-}"
            cwd_val="${cwd:-$PWD}"
            status="$import_status"
            created_at="$time_str"
            accessed_at="$(iso_now)"
            claude_args=""

            cat > "$import_dir/meta" <<EOF
session_id=$session_id
claude_pid=$claude_pid
cwd=$cwd_val
status=$status
created_at=$created_at
accessed_at=$accessed_at
claude_args=
EOF
            touch "$import_dir/tags" "$import_dir/notes"
            echo -e "$(iso_now)\timported\tsource=claude-sessions" >> "$import_dir/history"

            success "Imported as \"$import_name\""
            echo ""
        fi
    done

    if [[ $orphan_count -eq 0 ]]; then
        info "No orphaned sessions found. All Claude sessions are tracked by muxc."
    elif $scan_only; then
        info "Found $orphan_count orphaned session(s). Run 'muxc import' (without --scan) to adopt them."
    fi
}
