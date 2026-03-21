#!/usr/bin/env bash
# muxc-common.sh — shared functions for muxc

MUXC_VERSION="0.1.0"
MUXC_HOME="${MUXC_HOME:-$HOME/.muxc}"
MUXC_SESSIONS_DIR="$MUXC_HOME/sessions"

# ── Emoji + Color Logging ─────────────────────────────────────────────

die()     { echo "❌ $*" >&2; exit 1; }
warn()    { echo "⚠️  $*" >&2; }
info()    { echo "ℹ️  $*"; }
success() { echo "✨ $*"; }
action()  { echo "🔗 $*"; }
nav()     { echo "📂 $*"; }
launch()  { echo "🚀 $*"; }

# ── Path Helpers ──────────────────────────────────────────────────────

ensure_muxc_home() {
    mkdir -p "$MUXC_SESSIONS_DIR"
}

session_dir() {
    local name="$1"
    echo "$MUXC_SESSIONS_DIR/$name"
}

session_exists() {
    local name="$1"
    [[ -d "$(session_dir "$name")" ]]
}

# ── Name Validation ──────────────────────────────────────────────────

validate_name() {
    local name="$1"
    if [[ -z "$name" ]]; then
        die "Session name is required"
    fi
    if [[ ${#name} -gt 64 ]]; then
        die "Session name must be 64 characters or fewer"
    fi
    if [[ ! "$name" =~ ^[a-zA-Z0-9_-]+$ ]]; then
        die "Session name must contain only letters, numbers, hyphens, and underscores"
    fi
}

# ── Meta Read/Write ─────────────────────────────────────────────────

# Read meta file into shell variables (session_id, claude_pid, cwd, status, etc.)
read_meta() {
    local name="$1"
    local meta_file
    meta_file="$(session_dir "$name")/meta"
    if [[ ! -f "$meta_file" ]]; then
        die "Session \"$name\" not found"
    fi
    # Validate meta file lines before sourcing (guard against code injection)
    while IFS= read -r line; do
        if [[ -n "$line" && ! "$line" =~ ^[a-z_]+= ]]; then
            die "Corrupt meta file for session \"$name\": invalid line: $line"
        fi
    done < "$meta_file"
    source "$meta_file"
}

# Write meta file atomically (write to .tmp, then mv)
# All values are quoted to handle spaces in paths
write_meta() {
    local name="$1"
    local dir
    dir="$(session_dir "$name")"
    local tmp_file="$dir/meta.tmp"

    cat > "$tmp_file" <<EOF
session_id="${session_id}"
claude_pid="${claude_pid}"
cwd="${cwd}"
status="${status}"
created_at="${created_at}"
accessed_at="${accessed_at}"
claude_args="${claude_args}"
EOF
    mv -f "$tmp_file" "$dir/meta"
}

# ── History ──────────────────────────────────────────────────────────

append_history() {
    local name="$1"
    local event="$2"
    local details="${3:-}"
    local dir
    dir="$(session_dir "$name")"
    echo -e "$(iso_now)\t$event\t$details" >> "$dir/history"
}

# ── PID Management ───────────────────────────────────────────────────

# Check if a PID is alive and belongs to a claude process
check_pid() {
    local pid="$1"
    if [[ -z "$pid" ]]; then
        return 1
    fi
    # Check if process exists
    if ! kill -0 "$pid" 2>/dev/null; then
        return 1
    fi
    # Verify it's actually a claude process (guard against PID reuse)
    if [[ -f "/proc/$pid/cmdline" ]]; then
        if grep -q "claude" "/proc/$pid/cmdline" 2>/dev/null; then
            return 0
        fi
    fi
    return 1
}

# Scan all active sessions and transition dead ones to detached
reap_dead_sessions() {
    ensure_muxc_home
    local dir
    for dir in "$MUXC_SESSIONS_DIR"/*/; do
        [[ -d "$dir" ]] || continue
        local meta_file="$dir/meta"
        [[ -f "$meta_file" ]] || continue

        # Read meta in a subshell to avoid polluting variables
        local _status _pid _name
        _status=$(grep '^status=' "$meta_file" 2>/dev/null | cut -d= -f2- | tr -d '"')
        _pid=$(grep '^claude_pid=' "$meta_file" 2>/dev/null | cut -d= -f2- | tr -d '"')
        _name=$(basename "$dir")

        if [[ "$_status" == "active" && -n "$_pid" ]]; then
            if ! check_pid "$_pid"; then
                # PID is dead — transition to detached atomically
                (
                    read_meta "$_name"
                    status="detached"
                    claude_pid=""
                    accessed_at="$(iso_now)"
                    write_meta "$_name"
                )
                append_history "$_name" "detached" "pid=$_pid (process died)"
            fi
        fi
    done
}

# ── Claude Binary ────────────────────────────────────────────────────

get_claude_bin() {
    # Check config first
    if [[ -f "$MUXC_HOME/config" ]]; then
        local configured
        configured=$(grep '^claude_bin=' "$MUXC_HOME/config" 2>/dev/null | cut -d= -f2-)
        if [[ -n "$configured" ]]; then
            echo "$configured"
            return
        fi
    fi
    # Default: find claude in PATH
    local bin
    bin=$(command -v claude 2>/dev/null) || die "claude not found in PATH. Set claude_bin in ~/.muxc/config"
    echo "$bin"
}

# ── Claude Args Encoding ────────────────────────────────────────────

encode_claude_args() {
    local args="$*"
    if [[ -z "$args" ]]; then
        echo ""
        return
    fi
    echo "$args" | base64 -w0
}

decode_claude_args() {
    local encoded="$1"
    if [[ -z "$encoded" ]]; then
        echo ""
        return
    fi
    echo "$encoded" | base64 -d
}

# ── Time Helpers ─────────────────────────────────────────────────────

iso_now() {
    date -u +%Y-%m-%dT%H:%M:%SZ
}

# Format a relative time string from an ISO timestamp
relative_time() {
    local timestamp="$1"
    if [[ -z "$timestamp" ]]; then
        echo "never"
        return
    fi
    local then_epoch now_epoch diff
    then_epoch=$(date -d "$timestamp" +%s 2>/dev/null) || { echo "$timestamp"; return; }
    now_epoch=$(date +%s)
    diff=$((now_epoch - then_epoch))

    if [[ $diff -lt 60 ]]; then
        echo "just now"
    elif [[ $diff -lt 3600 ]]; then
        echo "$((diff / 60))m ago"
    elif [[ $diff -lt 86400 ]]; then
        echo "$((diff / 3600))h ago"
    else
        echo "$((diff / 86400))d ago"
    fi
}

# ── Table Formatting ─────────────────────────────────────────────────

# Print a formatted table row
# Usage: fmt_row "col1" "col2" "col3" ...
fmt_row() {
    printf "   %-2s %-20s %-30s %-12s %s\n" "$@"
}
