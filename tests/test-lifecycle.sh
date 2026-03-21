#!/usr/bin/env bash
set -euo pipefail

# test-lifecycle.sh — test muxc session lifecycle

SCRIPT_DIR="$(dirname "$(readlink -f "$0")")"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
MUXC_BIN="$PROJECT_DIR/bin/muxc"

source "$SCRIPT_DIR/test-helpers.sh"

# We need to source commands directly since exec replaces the process.
# For testing, we'll override exec to prevent process replacement.
MUXC_LIB="$PROJECT_DIR/lib"
source "$MUXC_LIB/muxc-common.sh"

# ── Setup ──────────────────────────────────────────────────────────

setup_test_env

# Re-derive MUXC_SESSIONS_DIR after MUXC_HOME changed
MUXC_SESSIONS_DIR="$MUXC_HOME/sessions"

# Override exec for testing (prevent process replacement)
exec() {
    echo "EXEC_CALLED: $*" >> "${MUXC_TEST_DIR}/exec-calls.log"
}

# Create a test working directory
TEST_CWD="$MUXC_TEST_DIR/test-project"
mkdir -p "$TEST_CWD"

echo "🧪 muxc lifecycle tests"
echo "════════════════════════════════════"

# ── Test: Name Validation ────────────────────────────────────────

echo ""
echo "📌 Name validation"

validate_name "valid-name" 2>/dev/null
assert_exit_code "0" "$?" "valid name with hyphens"

validate_name "valid_name_123" 2>/dev/null
assert_exit_code "0" "$?" "valid name with underscores and numbers"

output=$(validate_name "" 2>&1) || true
assert_contains "$output" "required" "empty name rejected"

output=$(validate_name "has spaces" 2>&1) || true
assert_contains "$output" "letters, numbers" "name with spaces rejected"

output=$(validate_name "../traversal" 2>&1) || true
assert_contains "$output" "letters, numbers" "path traversal rejected"

# ── Test: Session Creation (muxc new) ───────────────────────────

echo ""
echo "📌 Session creation"

source "$MUXC_LIB/muxc-new.sh"

# Test basic creation
cmd_new "test-session" --cwd "$TEST_CWD" 2>/dev/null || true

assert_file_exists "$MUXC_HOME/sessions/test-session/meta" "meta file created"
assert_file_exists "$MUXC_HOME/sessions/test-session/tags" "tags file created"
assert_file_exists "$MUXC_HOME/sessions/test-session/notes" "notes file created"
assert_file_exists "$MUXC_HOME/sessions/test-session/history" "history file created"

# Verify meta contents
meta_content=$(cat "$MUXC_HOME/sessions/test-session/meta")
assert_contains "$meta_content" "session_id=" "meta has session_id"
assert_contains "$meta_content" "cwd=$TEST_CWD" "meta has correct cwd"
assert_contains "$meta_content" "status=active" "meta has active status"

# Verify history
history_content=$(cat "$MUXC_HOME/sessions/test-session/history")
assert_contains "$history_content" "created" "history has created event"

# Verify exec was called with claude
if [[ -f "${MUXC_TEST_DIR}/exec-calls.log" ]]; then
    exec_log=$(cat "${MUXC_TEST_DIR}/exec-calls.log")
    assert_contains "$exec_log" "--session-id" "exec called with --session-id"
    assert_contains "$exec_log" "--name" "exec called with --name"
fi

# ── Test: Duplicate session name ─────────────────────────────────

echo ""
echo "📌 Duplicate detection"

output=$(cmd_new "test-session" 2>&1) || true
assert_contains "$output" "already exists" "duplicate name rejected"

# ── Test: Session with claude args ───────────────────────────────

echo ""
echo "📌 Claude args passthrough"

> "${MUXC_TEST_DIR}/exec-calls.log"  # Clear log
cmd_new "args-test" --cwd "$TEST_CWD" -- --dangerously-skip-permissions --model opus 2>/dev/null || true

meta_content=$(cat "$MUXC_HOME/sessions/args-test/meta")
assert_contains "$meta_content" "claude_args=" "meta has claude_args"

# Verify args are base64 encoded
claude_args_line=$(grep 'claude_args=' "$MUXC_HOME/sessions/args-test/meta" | cut -d= -f2-)
if [[ -n "$claude_args_line" ]]; then
    decoded=$(echo "$claude_args_line" | base64 -d 2>/dev/null)
    assert_contains "$decoded" "--dangerously-skip-permissions" "claude args decoded correctly"
    assert_contains "$decoded" "--model opus" "model arg preserved"
fi

# ── Test: Session with tags ──────────────────────────────────────

echo ""
echo "📌 Tags on creation"

cmd_new "tagged-session" --cwd "$TEST_CWD" --tag backend --tag urgent 2>/dev/null || true

tags_content=$(cat "$MUXC_HOME/sessions/tagged-session/tags")
assert_contains "$tags_content" "backend" "tag 'backend' saved"
assert_contains "$tags_content" "urgent" "tag 'urgent' saved"

# ── Test: Session Listing (muxc ls) ─────────────────────────────

echo ""
echo "📌 Session listing"

source "$MUXC_LIB/muxc-ls.sh"

# Mark sessions as detached for listing (since our mock doesn't create real PIDs)
for session_dir in "$MUXC_HOME/sessions"/*/; do
    [[ -d "$session_dir" ]] || continue
    sed -i 's/status=active/status=detached/' "$session_dir/meta"
    sed -i 's/claude_pid=.*/claude_pid=/' "$session_dir/meta"
done

output=$(cmd_ls 2>/dev/null)
assert_contains "$output" "test-session" "ls shows test-session"
assert_contains "$output" "args-test" "ls shows args-test"
assert_contains "$output" "tagged-session" "ls shows tagged-session"
assert_contains "$output" "🟡" "ls shows detached indicator"

# Test tag filter
output=$(cmd_ls --tag backend 2>/dev/null)
assert_contains "$output" "tagged-session" "tag filter finds tagged session"

# ── Test: Session Info ───────────────────────────────────────────

echo ""
echo "📌 Session info"

source "$MUXC_LIB/muxc-info.sh"

output=$(cmd_info "test-session" 2>/dev/null)
assert_contains "$output" "test-session" "info shows session name"
assert_contains "$output" "Session ID:" "info shows session ID"
assert_contains "$output" "$TEST_CWD" "info shows cwd"

# ── Test: Tags Management ───────────────────────────────────────

echo ""
echo "📌 Tag management"

source "$MUXC_LIB/muxc-tag.sh"

cmd_tag "test-session" add "feature" 2>/dev/null
tags_content=$(cat "$MUXC_HOME/sessions/test-session/tags")
assert_contains "$tags_content" "feature" "tag added"

cmd_tag "test-session" rm "feature" 2>/dev/null
tags_content=$(cat "$MUXC_HOME/sessions/test-session/tags")
[[ "$tags_content" != *"feature"* ]]
assert_exit_code "0" "$?" "tag removed"

# ── Test: Notes ──────────────────────────────────────────────────

echo ""
echo "📌 Notes"

source "$MUXC_LIB/muxc-note.sh"

cmd_note "test-session" "Working on the API refactor" 2>/dev/null
notes_content=$(cat "$MUXC_HOME/sessions/test-session/notes")
assert_contains "$notes_content" "API refactor" "note appended"

cmd_note "test-session" "Left off at migration step" 2>/dev/null
notes_content=$(cat "$MUXC_HOME/sessions/test-session/notes")
assert_contains "$notes_content" "migration step" "second note appended"

# ── Test: Rename ─────────────────────────────────────────────────

echo ""
echo "📌 Rename"

source "$MUXC_LIB/muxc-rename.sh"

cmd_rename "test-session" "renamed-session" 2>/dev/null
assert_file_not_exists "$MUXC_HOME/sessions/test-session" "old dir removed"
assert_file_exists "$MUXC_HOME/sessions/renamed-session/meta" "new dir created"

history_content=$(cat "$MUXC_HOME/sessions/renamed-session/history")
assert_contains "$history_content" "renamed" "history records rename"

# ── Test: Archive ────────────────────────────────────────────────

echo ""
echo "📌 Archive"

source "$MUXC_LIB/muxc-archive.sh"

cmd_archive "renamed-session" 2>/dev/null
meta_content=$(cat "$MUXC_HOME/sessions/renamed-session/meta")
assert_contains "$meta_content" "status=archived" "status set to archived"

# Verify ls hides archived by default
output=$(cmd_ls 2>/dev/null)
[[ "$output" != *"renamed-session"* ]]
assert_exit_code "0" "$?" "archived hidden from default ls"

# Verify ls --all shows archived
output=$(cmd_ls --all 2>/dev/null)
assert_contains "$output" "renamed-session" "archived shown with --all"

# ── Test: Remove ─────────────────────────────────────────────────

echo ""
echo "📌 Remove"

source "$MUXC_LIB/muxc-rm.sh"

cmd_rm "renamed-session" 2>/dev/null
assert_file_not_exists "$MUXC_HOME/sessions/renamed-session" "session dir removed"

# ── Test: Args Encoding/Decoding ─────────────────────────────────

echo ""
echo "📌 Args encoding/decoding"

encoded=$(encode_claude_args "--dangerously-skip-permissions --model opus")
decoded=$(decode_claude_args "$encoded")
assert_eq "--dangerously-skip-permissions --model opus" "$decoded" "round-trip encode/decode"

empty_encoded=$(encode_claude_args "")
assert_eq "" "$empty_encoded" "empty args encode to empty"

empty_decoded=$(decode_claude_args "")
assert_eq "" "$empty_decoded" "empty args decode to empty"

# ── Test: Relative Time ─────────────────────────────────────────

echo ""
echo "📌 Relative time"

now_iso=$(iso_now)
rel=$(relative_time "$now_iso")
assert_eq "just now" "$rel" "current time shows 'just now'"

empty_rel=$(relative_time "")
assert_eq "never" "$empty_rel" "empty timestamp shows 'never'"

# ── Test: Help ───────────────────────────────────────────────────

echo ""
echo "📌 Help output"

source "$MUXC_LIB/muxc-help.sh"

output=$(cmd_help 2>/dev/null)
assert_contains "$output" "muxc" "help mentions muxc"
assert_contains "$output" "new" "help mentions new command"
assert_contains "$output" "attach" "help mentions attach command"

output=$(cmd_help new 2>/dev/null)
assert_contains "$output" "--cwd" "new help mentions --cwd"
assert_contains "$output" "--" "new help mentions -- separator"

# ── Cleanup & Summary ───────────────────────────────────────────

cleanup_test_env
echo ""
test_summary
