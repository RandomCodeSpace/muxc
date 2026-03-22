# muxc Shell Rewrite — Design Spec

## Problem

muxc is a 1,040-line Go binary with 17 dependencies (cobra, lipgloss, charmbracelet ecosystem) that fundamentally just scans files and `exec`s claude. The Go toolchain is required to build, binaries must be compiled per platform, and users cannot easily modify the tool. Since the zero-storage refactor, muxc reads all data from Claude Code's native `~/.claude/` files — there's no complex state management left.

## Solution

Rewrite muxc as a single bash script (~250-300 lines). Auto-install jq for JSON parsing. Hybrid exec strategy: subprocess on create (to capture session ID + save args), `exec` on resume (for perfect TTY). Store per-session args in a simple properties file.

## Architecture

### Single File Distribution

- **File:** `muxc` — a bash script, `chmod +x`, drop in PATH
- **Install:** `curl -fsSL <url> | install -m 755 /dev/stdin ~/.local/bin/muxc`
- **Dependencies:** bash (3.2+ compatible), curl (for jq bootstrap), standard coreutils
- **jq:** Auto-installed to `~/.local/bin/jq` on first run if not present

### jq Bootstrap

On script start, check `command -v jq`. If missing:
1. Detect OS/arch via `uname`
2. Download pinned version from `github.com/jqlang/jq/releases/download/jq-1.7.1/`
3. Install to `~/.local/bin/jq`, add to PATH
4. Exit with clear error if download fails: "jq is required. Install via your package manager or ensure curl works."

### Config File

**Path:** `~/.config/muxc.conf`
**Format:** Simple properties — one line per session:
```
8c145fa0-0e57-45ba-9a8e-f440c9a75b9e=--dangerously-skip-permissions
27843d46-120c-4089-b23a-937cdd15cee4=--dangerously-skip-permissions --model opus
```
**Read:** `grep "^${session_id}=" ~/.config/muxc.conf | tail -1 | cut -d= -f2-` (tail -1 ensures latest entry wins)
**Write:** Deduplicate on write — `sed -i "/^${session_id}=/d"` then `echo >>` to avoid stale entries.
**Args splitting:** Use `read -ra args <<< "$line"` for safe word-splitting (no eval).

**Migration:** If `~/.config/muxc.json` exists (from Go version), ignore it. The old JSON config is not migrated — users re-pass args on next create.

### Environment Variables

| Variable | Description |
|----------|-------------|
| `MUXC_CLAUDE_BIN` | Path to `claude` binary (default: auto-detected from PATH) |

### Commands

| Command | Description |
|---------|-------------|
| `muxc` | List sessions (same as `muxc ls`) |
| `muxc <name>` | Resume most recent session with that name, or create |
| `muxc <name>:<idprefix>` | Resume specific session by ID prefix |
| `muxc ls [-s status]` | List sessions with ID, directory, modified time |
| `muxc list` / `muxc l` | Aliases for `muxc ls` |
| `muxc info <name>` | Show session details |
| `muxc version` | Print version |

**Shell completion:** Deferred. Not implemented in the initial shell rewrite. Users use `muxc ls` to discover session names.

### Create Flow (subprocess — captures session ID)

1. Parse name, reject reserved commands (`ls`, `list`, `l`, `info`, `version`, `help`)
2. Resolve `--cwd <dir>` flag or default to `pwd`
3. Prompt "Session not found. Create? [Y/n]"
4. Run claude as subprocess: `claude --dangerously-skip-permissions --name "$name" "$@"`
5. In background, poll `~/.claude/sessions/$!.json` for session ID (up to 10s)
6. After claude exits, save `-- args` to `~/.config/muxc.conf` keyed by captured session ID
7. Forward signals (trap SIGINT/SIGTERM → kill child)

### Resume Flow (exec — perfect TTY)

1. Scan JSONL files for matching `customTitle`
2. If multiple matches and no `:idprefix`, use most recent (by mtime), print hint
3. If `:idprefix` given, match against session IDs
4. **Check PID liveness** — if session is active, print error and exit (prevent double-attach)
5. Look up saved args from `~/.config/muxc.conf`
6. `exec claude $saved_args --resume "$session_id" "$@"` — replaces shell, perfect TTY

**Resume failure:** Since `exec` replaces the process, there is no fallback if Claude can't find the session. The user sees Claude's native error message and can re-run `muxc <name>` to create fresh. This is an accepted trade-off for clean TTY handling.

### Session Scanning

**Three data sources:**

1. **JSONL titles** — `head -1 "$file" | jq -r '[.customTitle, .sessionId] | @tsv'`
   - Scan `~/.claude/projects/*/*.jsonl` (top-level only, skip subdirs)
   - Skip files where type != "custom-title" or title is empty

2. **PID files** — `jq -r '[.pid, .sessionId, .cwd] | @tsv' "$file"`
   - Scan `~/.claude/sessions/*.json`
   - Cross-reference sessionId with JSONL data

3. **PID liveness** — cross-platform:
   - Linux: `kill -0 "$pid" 2>/dev/null && grep -q claude "/proc/$pid/cmdline" 2>/dev/null`
   - macOS: `kill -0 "$pid" 2>/dev/null && ps -p "$pid" -o command= 2>/dev/null | grep -q claude`
   - Detect OS once at script start, set check function accordingly

**Sort:** By JSONL file mtime, most recent first.
- Linux: `stat -c %Y "$file"`
- macOS: `stat -f %m "$file"`
- Detect OS once, use appropriate stat format.

**Project hash decode:** Replace `-` with `/`, stat check, fall back to raw hash.

**Name:idprefix parse:** Check for `:` in ref. `${ref%:*}` for name, `${ref##*:}` for prefix. Trailing colon stripped.

### Table Output

ANSI-styled printf, no dependencies:
```
       NAME                     ID        DIRECTORY                    MODIFIED
  ⏸  muxc                     8c145fa0  ~/git/muxc                   just now
  ⏸  muxc                     acb91e49  ~/git/muxc                   1h ago
  ▶  otelctx                  b1f83b6d  ~/git/otelcontext            5h ago

  3 session(s)
```

**Colors:**
- `▶` active: green
- `⏸` detached: yellow
- Name: bold
- ID, directory, time: dim/faint
- Header: faint
- Session count footer: dim/faint

**Relative time:** Compare mtime epoch vs `date +%s`. Output: "just now" (<60s), "Xm ago", "Xh ago", "Xd ago".

**Home shortening:** `${path/#$HOME/~}`

### Flags

| Flag | Scope | Description |
|------|-------|-------------|
| `--cwd <dir>` | create | Working directory for new session (default: current dir) |
| `-s <status>` | ls | Filter by status (active/detached) |

## What Gets Deleted

The entire Go codebase:
- `main.go`, `go.mod`, `go.sum`
- `cmd/` directory (root.go, ls.go, info.go, version.go, completion.go)
- `internal/` directory (claude/, config/, session/, ui/)
- `Makefile` (Go build targets — replaced with simple install target)
- `.goreleaser.yml` or any Go release config

## What Gets Kept

- `README.md` (updated for shell script)
- `LICENSE`
- `install.sh` (updated)
- `.github/` workflows (simplified — shellcheck + install test)

## Testing

- **shellcheck:** Lint the script for common bash pitfalls
- **Manual smoke tests:** `muxc ls`, `muxc info <name>`, `muxc <name>` create/resume
- **CI:** shellcheck + basic functional tests in GitHub Actions

## Success Criteria

- Single file, <350 lines
- Same UX as Go binary (ls, info, create, resume, name:id disambiguation)
- Perfect TTY on resume (exec), session ID capture on create (subprocess)
- No compilation, no Go toolchain
- Works on Linux and macOS with bash 3.2+
