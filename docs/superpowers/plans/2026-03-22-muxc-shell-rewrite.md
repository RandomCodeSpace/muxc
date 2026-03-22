# muxc Shell Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the Go muxc binary with a single bash script that provides the same UX — list, info, create, resume Claude Code sessions — reading all data from `~/.claude/`.

**Architecture:** Single bash script with jq auto-bootstrap. Hybrid exec: subprocess on create (captures session ID), `exec` on resume (perfect TTY). Properties file for per-session args. Cross-platform (Linux + macOS).

**Tech Stack:** bash 3.2+, jq (auto-installed), coreutils

**Spec:** `docs/superpowers/specs/2026-03-22-muxc-shell-rewrite-design.md`

---

## File Map

### Create
- `muxc` — the entire script (replaces all Go code)

### Modify
- `README.md` — update for shell script
- `install.sh` — simplify (no binary download, just copy script)
- `.github/workflows/ci.yml` — shellcheck instead of Go build/test
- `.github/workflows/release.yml` — copy script to release instead of cross-compile

### Delete
- `main.go`
- `go.mod`, `go.sum`
- `cmd/root.go`, `cmd/ls.go`, `cmd/info.go`, `cmd/version.go`, `cmd/completion.go`
- `internal/claude/claude.go`, `internal/claude/claude_test.go`
- `internal/config/config.go`
- `internal/session/exec.go`
- `internal/ui/output.go`, `internal/ui/table.go`
- `Makefile`
- Compiled `muxc` binary (in repo root, gitignored or not)

---

### Task 1: Write the muxc bash script — scaffolding and helpers

**Files:**
- Create: `muxc`

This task creates the script with the shebang, globals, OS detection, jq bootstrap, and utility functions. No commands yet.

- [ ] **Step 1: Create the script with header, globals, OS detection**

```bash
#!/usr/bin/env bash
set -euo pipefail

VERSION="0.2.0"
CLAUDE_BIN="${MUXC_CLAUDE_BIN:-}"
CONFIG_FILE="${HOME}/.config/muxc.conf"
CLAUDE_DIR="${HOME}/.claude"

# OS detection (once, at startup)
OS=$(uname -s)
ARCH=$(uname -m)

# ANSI colors
BOLD=$'\e[1m'
DIM=$'\e[2m'
GREEN=$'\e[32m'
YELLOW=$'\e[33m'
CYAN=$'\e[36m'
MAGENTA=$'\e[35m'
RESET=$'\e[0m'
```

- [ ] **Step 2: Add utility functions**

```bash
die()  { echo "❌ $*" >&2; exit 1; }
warn() { echo "⚠️  $*" >&2; }
info() { echo "ℹ️  $*"; }

# Cross-platform stat for mtime epoch
file_mtime() {
    case "$OS" in
        Linux*)  stat -c %Y "$1" 2>/dev/null || echo 0 ;;
        Darwin*) stat -f %m "$1" 2>/dev/null || echo 0 ;;
        *)       echo 0 ;;
    esac
}

# Relative time from epoch seconds
relative_time() {
    local diff=$(( $(date +%s) - $1 ))
    if   (( diff < 60 ));    then echo "just now"
    elif (( diff < 3600 ));  then echo "$(( diff / 60 ))m ago"
    elif (( diff < 86400 )); then echo "$(( diff / 3600 ))h ago"
    else echo "$(( diff / 86400 ))d ago"
    fi
}

# Shorten home prefix
shorten_path() { echo "${1/#$HOME/~}"; }

# Decode project hash (best-effort)
decode_project_hash() {
    local candidate="${1//-//}"
    if [[ -d "$candidate" ]]; then
        echo "$candidate"
    else
        echo "$1"
    fi
}

# Cross-platform PID liveness check
check_pid() {
    local pid=$1
    (( pid <= 0 )) && return 1
    kill -0 "$pid" 2>/dev/null || return 1
    case "$OS" in
        Linux*)  grep -q claude "/proc/$pid/cmdline" 2>/dev/null ;;
        Darwin*) ps -p "$pid" -o command= 2>/dev/null | grep -q claude ;;
        *)       return 1 ;;
    esac
}
```

- [ ] **Step 3: Add jq bootstrap**

```bash
ensure_jq() {
    command -v jq &>/dev/null && return
    local os arch url
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$ARCH" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) die "Unsupported architecture: $ARCH" ;;
    esac
    [[ "$os" == "darwin" ]] && os="macos"
    url="https://github.com/jqlang/jq/releases/download/jq-1.7.1/jq-${os}-${arch}"
    info "📦 Installing jq to ~/.local/bin..."
    mkdir -p "$HOME/.local/bin"
    curl -fsSL "$url" -o "$HOME/.local/bin/jq" && chmod +x "$HOME/.local/bin/jq" \
        || die "Failed to install jq. Install manually: https://jqlang.github.io/jq/"
    export PATH="$HOME/.local/bin:$PATH"
}

# Resolve claude binary
find_claude() {
    if [[ -n "$CLAUDE_BIN" ]]; then
        [[ -x "$CLAUDE_BIN" ]] || die "MUXC_CLAUDE_BIN=$CLAUDE_BIN is not executable"
        return
    fi
    CLAUDE_BIN=$(command -v claude 2>/dev/null) \
        || die "claude not found in PATH. Set MUXC_CLAUDE_BIN or install Claude Code."
}
```

- [ ] **Step 4: Add config read/write functions**

```bash
# Read saved args for a session ID from properties file
config_get_args() {
    local sid="$1"
    [[ -f "$CONFIG_FILE" ]] || return 0
    grep "^${sid}=" "$CONFIG_FILE" 2>/dev/null | tail -1 | cut -d= -f2-
}

# Save args for a session ID (dedup on write)
config_set_args() {
    local sid="$1" args="$2"
    mkdir -p "$(dirname "$CONFIG_FILE")"
    if [[ -f "$CONFIG_FILE" ]]; then
        # Remove old entry (portable sed -i)
        local tmp="${CONFIG_FILE}.tmp"
        grep -v "^${sid}=" "$CONFIG_FILE" > "$tmp" 2>/dev/null || true
        mv "$tmp" "$CONFIG_FILE"
    fi
    echo "${sid}=${args}" >> "$CONFIG_FILE"
}
```

- [ ] **Step 5: Update .gitignore — remove `muxc` entry**

The current `.gitignore` has `muxc` (the compiled Go binary). Since the script IS named `muxc`, we must remove that entry so git tracks it.

```bash
sed -i '/^muxc$/d' .gitignore
```

- [ ] **Step 6: Make executable and verify syntax**

```bash
chmod +x muxc
bash -n muxc  # syntax check
shellcheck muxc
```

- [ ] **Step 7: Commit**

```bash
git add muxc .gitignore
git commit -m "feat(shell): add muxc script scaffolding with helpers and jq bootstrap"
```

---

### Task 2: Session scanning functions

**Files:**
- Modify: `muxc`

Add the core scanning logic: build PID map, scan JSONL titles, resolve sessions.

- [ ] **Step 1: Add scan_sessions function**

This builds an associative-array-like structure using parallel indexed arrays (bash 3.2 compatible — no associative arrays).

```bash
# Global session data (parallel arrays, populated by scan_sessions)
S_NAMES=()    # customTitle
S_IDS=()      # sessionId
S_PROJS=()    # project hash
S_CWDS=()     # working directory
S_PIDS=()     # live PID or 0
S_STATUSES=() # "active" or "detached"
S_MTIMES=()   # mtime epoch

scan_sessions() {
    S_NAMES=(); S_IDS=(); S_PROJS=(); S_CWDS=(); S_PIDS=(); S_STATUSES=(); S_MTIMES=()

    local projects_dir="$CLAUDE_DIR/projects"
    local sessions_dir="$CLAUDE_DIR/sessions"
    [[ -d "$projects_dir" ]] || return 0

    # Build PID map as temp file: "sessionId\tpid\tcwd" per line
    # (bash 3.2 compatible — no associative arrays)
    local pid_map_file
    pid_map_file=$(mktemp)
    trap "rm -f '$pid_map_file'" RETURN
    if [[ -d "$sessions_dir" ]]; then
        for pf in "$sessions_dir"/*.json; do
            [[ -f "$pf" ]] || continue
            local pdata
            pdata=$(jq -r '[.pid // 0, .sessionId // "", .cwd // ""] | @tsv' "$pf" 2>/dev/null) || continue
            local ppid psid pcwd
            IFS=$'\t' read -r ppid psid pcwd <<< "$pdata"
            [[ -n "$psid" ]] || continue
            if check_pid "$ppid"; then
                # Live PID — overwrite any existing entry
                sed -i "/^${psid}	/d" "$pid_map_file" 2>/dev/null || true
                printf '%s\t%s\t%s\n' "$psid" "$ppid" "$pcwd" >> "$pid_map_file"
            elif ! grep -q "^${psid}	" "$pid_map_file" 2>/dev/null; then
                printf '%s\t%s\t%s\n' "$psid" "0" "$pcwd" >> "$pid_map_file"
            fi
        done
    fi

    # Scan JSONL files for titled sessions
    local entries=()
    for projdir in "$projects_dir"/*/; do
        [[ -d "$projdir" ]] || continue
        local proj_hash
        proj_hash=$(basename "$projdir")
        for jf in "$projdir"*.jsonl; do
            [[ -f "$jf" ]] || continue
            local title sid
            local firstline
            firstline=$(head -1 "$jf" 2>/dev/null) || continue
            title=$(echo "$firstline" | jq -r 'select(.type=="custom-title") | .customTitle // ""' 2>/dev/null)
            [[ -n "$title" ]] || continue
            sid=$(echo "$firstline" | jq -r '.sessionId // ""' 2>/dev/null)
            [[ -n "$sid" ]] || continue

            local mtime
            mtime=$(file_mtime "$jf")
            local pid=0 cwd="" status="detached"
            local pid_entry
            pid_entry=$(grep "^${sid}	" "$pid_map_file" 2>/dev/null) || true
            if [[ -n "$pid_entry" ]]; then
                IFS=$'\t' read -r _ pid cwd <<< "$pid_entry"
                (( pid > 0 )) && status="active"
            fi
            [[ -z "$cwd" ]] && cwd=$(decode_project_hash "$proj_hash")

            entries+=("${mtime}|${#S_NAMES[@]}")
            S_NAMES+=("$title")
            S_IDS+=("$sid")
            S_PROJS+=("$proj_hash")
            S_CWDS+=("$cwd")
            S_PIDS+=("$pid")
            S_STATUSES+=("$status")
            S_MTIMES+=("$mtime")
        done
    done

    # Sort by mtime descending — rebuild arrays in sorted order
    if (( ${#entries[@]} > 0 )); then
        local sorted
        sorted=$(printf '%s\n' "${entries[@]}" | sort -t'|' -k1,1rn)
        local new_names=() new_ids=() new_projs=() new_cwds=() new_pids=() new_statuses=() new_mtimes=()
        while IFS='|' read -r _ idx; do
            new_names+=("${S_NAMES[$idx]}")
            new_ids+=("${S_IDS[$idx]}")
            new_projs+=("${S_PROJS[$idx]}")
            new_cwds+=("${S_CWDS[$idx]}")
            new_pids+=("${S_PIDS[$idx]}")
            new_statuses+=("${S_STATUSES[$idx]}")
            new_mtimes+=("${S_MTIMES[$idx]}")
        done <<< "$sorted"
        S_NAMES=("${new_names[@]}"); S_IDS=("${new_ids[@]}"); S_PROJS=("${new_projs[@]}")
        S_CWDS=("${new_cwds[@]}"); S_PIDS=("${new_pids[@]}"); S_STATUSES=("${new_statuses[@]}")
        S_MTIMES=("${new_mtimes[@]}")
    fi
}
```

- [ ] **Step 2: Add session lookup functions**

```bash
# Parse "name:idprefix" → sets REF_NAME and REF_PREFIX
parse_ref() {
    local ref="$1"
    # Strip trailing colon
    [[ "$ref" == *: ]] && ref="${ref%:}"
    if [[ "$ref" == *:* ]]; then
        REF_NAME="${ref%:*}"
        REF_PREFIX="${ref##*:}"
    else
        REF_NAME="$ref"
        REF_PREFIX=""
    fi
}

# Find session by ref. Sets FOUND_IDX and FOUND_COUNT.
find_session() {
    local ref="$1"
    parse_ref "$ref"
    FOUND_IDX=-1
    FOUND_COUNT=0

    local first_match=-1
    for i in "${!S_NAMES[@]}"; do
        if [[ "${S_NAMES[$i]}" == "$REF_NAME" ]]; then
            (( FOUND_COUNT++ ))
            if [[ -n "$REF_PREFIX" ]]; then
                if [[ "${S_IDS[$i]}" == "$REF_PREFIX"* ]]; then
                    FOUND_IDX=$i
                    return 0
                fi
            elif (( first_match < 0 )); then
                first_match=$i
            fi
        fi
    done

    if [[ -n "$REF_PREFIX" ]]; then
        return 1  # prefix didn't match
    fi
    if (( first_match >= 0 )); then
        FOUND_IDX=$first_match
        return 0
    fi
    return 1  # not found
}
```

- [ ] **Step 3: Verify syntax**

```bash
bash -n muxc && shellcheck muxc
```

- [ ] **Step 4: Commit**

```bash
git add muxc
git commit -m "feat(shell): add session scanning and lookup functions"
```

---

### Task 3: Command implementations

**Files:**
- Modify: `muxc`

Add `cmd_ls`, `cmd_info`, `cmd_create`, `cmd_resume`, and the main routing logic.

- [ ] **Step 1: Add cmd_ls**

```bash
cmd_ls() {
    local filter_status=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -s) [[ $# -ge 2 ]] || die "Usage: muxc ls [-s status]"; filter_status="$2"; shift 2 ;;
            *)  shift ;;
        esac
    done

    scan_sessions

    if (( ${#S_NAMES[@]} == 0 )); then
        info "No sessions found."
        return 0
    fi

    # Header
    printf "  ${DIM}%-4s  %-24s  %-10s  %-32s  %-12s${RESET}\n" "" "NAME" "ID" "DIRECTORY" "MODIFIED"

    local count=0
    for i in "${!S_NAMES[@]}"; do
        [[ -n "$filter_status" && "${S_STATUSES[$i]}" != "$filter_status" ]] && continue
        local icon
        if [[ "${S_STATUSES[$i]}" == "active" ]]; then
            icon="${GREEN}▶${RESET} "
        else
            icon="${YELLOW}⏸${RESET} "
        fi
        local short_id="${S_IDS[$i]:0:8}"
        local rel
        rel=$(relative_time "${S_MTIMES[$i]}")
        printf "  %-4s  ${BOLD}%-24s${RESET}  ${DIM}%-10s  %-32s  %-12s${RESET}\n" \
            "$icon" "${S_NAMES[$i]}" "$short_id" "$(shorten_path "${S_CWDS[$i]}")" "$rel"
        (( count++ ))
    done

    echo
    printf "  ${DIM}%d session(s)${RESET}\n" "$count"
}
```

- [ ] **Step 2: Add cmd_info**

```bash
cmd_info() {
    [[ $# -lt 1 ]] && die "Usage: muxc info <name>"
    scan_sessions
    find_session "$1" || die "session \"$REF_NAME\" not found"

    local i=$FOUND_IDX
    echo "ℹ️  Session: ${S_NAMES[$i]}"
    local status_icon
    if [[ "${S_STATUSES[$i]}" == "active" ]]; then
        status_icon="${GREEN}▶${RESET}"
        echo "   Status: $status_icon ${S_STATUSES[$i]} (PID ${S_PIDS[$i]})"
    else
        status_icon="${YELLOW}⏸${RESET}"
        echo "   Status: $status_icon ${S_STATUSES[$i]}"
    fi
    echo "   Session ID: ${S_IDS[$i]}"
    echo "   Project: $(decode_project_hash "${S_PROJS[$i]}")"
    echo "   Directory: $(shorten_path "${S_CWDS[$i]}")"
    if (( S_MTIMES[$i] > 0 )); then
        echo "   Last modified: $(date -d "@${S_MTIMES[$i]}" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date -r "${S_MTIMES[$i]}" '+%Y-%m-%d %H:%M:%S' 2>/dev/null) ($(relative_time "${S_MTIMES[$i]}"))"
    fi
}
```

- [ ] **Step 3: Add create and resume flows**

```bash
cmd_create() {
    local name="$1"; shift
    local cwd="${FLAG_CWD:-$(pwd)}"
    local claude_args=("--dangerously-skip-permissions" "--name" "$name" "$@")
    local extra_args="$*"

    echo "🚀 Creating session \"$name\""

    # We need claude in the FOREGROUND for TTY/stdin access.
    # Poll for session ID in a BACKGROUND subshell.
    # After claude exits, the background poller's result is in a temp file.
    local sid_file
    sid_file=$(mktemp)

    # Start background poller — it will discover claude's PID via the
    # temporary marker and poll ~/.claude/sessions/<pid>.json
    (
        # Wait briefly for claude to start and get a PID
        sleep 0.5
        # Find the claude PID by looking for the most recent sessions/*.json
        local pid_file=""
        for f in "$CLAUDE_DIR/sessions"/*.json; do
            [[ -f "$f" ]] || continue
            local fpid
            fpid=$(jq -r '.pid // 0' "$f" 2>/dev/null) || continue
            if check_pid "$fpid"; then
                # Verify this is our claude process (has --name in cmdline)
                local cmdline
                cmdline=$(cat "/proc/$fpid/cmdline" 2>/dev/null || ps -p "$fpid" -o command= 2>/dev/null || true)
                if echo "$cmdline" | grep -q "$name"; then
                    pid_file="$f"
                fi
            fi
        done
        if [[ -n "$pid_file" ]]; then
            local sid
            sid=$(jq -r '.sessionId // ""' "$pid_file" 2>/dev/null)
            [[ -n "$sid" ]] && echo "$sid" > "$sid_file"
        fi
    ) &
    local poller_pid=$!

    # Run claude in foreground — gets full TTY access
    "$CLAUDE_BIN" "${claude_args[@]}" || true

    # Wait for poller to finish
    wait "$poller_pid" 2>/dev/null || true

    # Read captured session ID
    local captured_sid=""
    [[ -f "$sid_file" ]] && captured_sid=$(cat "$sid_file")
    rm -f "$sid_file"

    # Save args if we captured a session ID and user passed extra args
    if [[ -n "$captured_sid" && -n "$extra_args" ]]; then
        config_set_args "$captured_sid" "$extra_args"
    fi

    if [[ -z "$captured_sid" ]]; then
        warn "Could not capture session ID — resume may not work for session \"$name\""
    fi
}

cmd_resume() {
    local i=$1; shift
    local sid="${S_IDS[$i]}"
    local cwd="${S_CWDS[$i]}"

    # Block if already active
    if [[ "${S_STATUSES[$i]}" == "active" ]] && check_pid "${S_PIDS[$i]}"; then
        die "session \"${S_NAMES[$i]}\" is already active (PID ${S_PIDS[$i]})"
    fi

    # Restore saved args
    local saved_args
    saved_args=$(config_get_args "$sid")
    local -a exec_args=("--dangerously-skip-permissions")
    if [[ -n "$saved_args" ]]; then
        local -a parsed
        read -ra parsed <<< "$saved_args"
        exec_args+=("${parsed[@]}")
    fi
    exec_args+=("--resume" "$sid" "$@")

    echo "🔗 Resuming session \"${S_NAMES[$i]}\""
    cd "$cwd" 2>/dev/null || true
    exec "$CLAUDE_BIN" "${exec_args[@]}"
}
```

- [ ] **Step 4: Add main routing**

```bash
# Reserved command names
is_reserved() {
    case "$1" in
        ls|list|l|info|version|help|completion) return 0 ;;
        *) return 1 ;;
    esac
}

main() {
    ensure_jq
    find_claude

    local FLAG_CWD=""

    # Parse global flags before command
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --cwd)  FLAG_CWD="$2"; shift 2 ;;
            --)     shift; break ;;
            *)      break ;;
        esac
    done

    # No args → list sessions
    if [[ $# -eq 0 ]]; then
        cmd_ls
        return
    fi

    local cmd="$1"; shift

    case "$cmd" in
        ls|list|l)  cmd_ls "$@" ;;
        info)       cmd_info "$@" ;;
        version)    echo "muxc $VERSION" ;;
        help|--help|-h) echo "Usage: muxc [<session>] [flags] [-- <claude-args>...]"; echo; echo "Commands: ls, info, version" ;;
        *)
            # Session name — resume or create
            local ref="$cmd"
            parse_ref "$ref"

            if is_reserved "$REF_NAME"; then
                die "\"$REF_NAME\" is a reserved command name — choose a different session name"
            fi

            scan_sessions

            if find_session "$ref"; then
                if (( FOUND_COUNT > 1 )) && [[ -z "$REF_PREFIX" ]]; then
                    info "📋 $FOUND_COUNT sessions named \"$REF_NAME\" — using most recent. Run muxc ls to see all IDs, use muxc $REF_NAME:<id> to select."
                fi
                cmd_resume "$FOUND_IDX" "$@"
            elif (( FOUND_COUNT > 0 )); then
                # Name exists but prefix didn't match
                die "no session \"$REF_NAME\" with ID prefix \"$REF_PREFIX\""
            else
                # Not found — offer to create
                printf 'Session "%s" not found. Create it? [Y/n]: ' "$REF_NAME"
                read -r answer
                answer=$(echo "$answer" | tr '[:upper:]' '[:lower:]')
                if [[ -z "$answer" || "$answer" == "y" || "$answer" == "yes" ]]; then
                    cmd_create "$REF_NAME" "$@"
                fi
            fi
            ;;
    esac
}

main "$@"
```

- [ ] **Step 5: Verify syntax and lint**

```bash
bash -n muxc && shellcheck muxc
```

- [ ] **Step 6: Smoke test**

```bash
chmod +x muxc
./muxc ls
./muxc info muxc
./muxc version
```

- [ ] **Step 7: Commit**

```bash
git add muxc
git commit -m "feat(shell): add all commands — ls, info, create, resume, version"
```

---

### Task 4: Delete Go codebase

**Files:**
- Delete: `main.go`, `go.mod`, `go.sum`, `Makefile`
- Delete: `cmd/` directory
- Delete: `internal/` directory

- [ ] **Step 1: Delete all Go files and build config**

```bash
rm -f main.go go.mod go.sum Makefile
rm -rf cmd/ internal/
```

- [ ] **Step 2: Remove compiled binary if present**

```bash
rm -f /home/dev/git/muxc/muxc.exe  # Windows artifact if any
# The script IS named muxc, which replaces the binary
```

- [ ] **Step 3: Update .gitignore if needed**

Remove any Go-specific entries (like `muxc` binary). The script is now tracked as source.

- [ ] **Step 4: Verify clean state**

```bash
./muxc ls  # still works
./muxc version  # prints version
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "chore: remove entire Go codebase — replaced by bash script"
```

---

### Task 5: Update README, install.sh, and CI

**Files:**
- Modify: `README.md`
- Modify: `install.sh`
- Modify: `.github/workflows/ci.yml`
- Modify: `.github/workflows/release.yml`

- [ ] **Step 1: Rewrite README.md**

Update to reflect shell script:
- Remove "Go install" and "Build from source" sections (replace with `chmod +x`)
- Remove shell completion section (deferred)
- Update install instructions (curl the script directly)
- Keep: Quick start, Commands table, How it works, Status icons, Environment variables
- Add: "jq is auto-installed if missing" note

- [ ] **Step 2: Simplify install.sh**

Replace the binary-download logic with:
```bash
#!/bin/sh
set -e
INSTALL_DIR="${HOME}/.local/bin"
mkdir -p "$INSTALL_DIR"
curl -fsSL "https://raw.githubusercontent.com/RandomCodeSpace/muxc/main/muxc" \
    -o "$INSTALL_DIR/muxc"
chmod +x "$INSTALL_DIR/muxc"
echo "✅ muxc installed to $INSTALL_DIR/muxc"
```

- [ ] **Step 3: Update ci.yml**

Replace Go build/test with:
```yaml
- name: Lint
  run: shellcheck muxc
- name: Syntax check
  run: bash -n muxc
```

- [ ] **Step 4: Update release.yml**

Replace cross-compile with simple release that attaches the script + checksums.

- [ ] **Step 5: Commit**

```bash
git add README.md install.sh .github/workflows/
git commit -m "docs: update README, install.sh, and CI for shell rewrite"
```

---

### Task 6: End-to-end verification

- [ ] **Step 1: Verify muxc ls**

```bash
./muxc ls
# Should show sessions with status, name, ID, directory, modified time
```

- [ ] **Step 2: Verify muxc info**

```bash
./muxc info muxc
# Should show session details
```

- [ ] **Step 3: Verify muxc info with ID prefix**

```bash
./muxc info muxc:27843d46
# Should select specific session
```

- [ ] **Step 4: Verify muxc version**

```bash
./muxc version
# Should print: muxc 0.2.0
```

- [ ] **Step 5: Verify status filter**

```bash
./muxc ls -s detached
# Should only show detached sessions
```

- [ ] **Step 6: Verify bad prefix error**

```bash
./muxc info muxc:zzzz
# Should print: no session "muxc" with ID prefix "zzzz"
```

- [ ] **Step 7: Shellcheck passes**

```bash
shellcheck muxc
```

- [ ] **Step 8: No Go files remain**

```bash
find . -name '*.go' -not -path './.git/*'
# Should return nothing
```

- [ ] **Step 9: Final commit with tag**

```bash
git add -A
git commit -m "feat: muxc v0.2.0 — rewritten as single bash script"
git tag v0.2.0
```
