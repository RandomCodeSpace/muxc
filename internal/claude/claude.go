package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/RandomCodeSpace/muxc/internal/session"
)

// Session represents a Claude Code session derived from native CLI data.
type Session struct {
	Name      string    // customTitle from JSONL (may be empty)
	SessionID string    // UUID from JSONL filename
	Project   string    // project directory hash (e.g., -home-dev-git-muxc)
	Cwd       string    // working directory (from sessions/{pid}.json or decoded project hash)
	PID       int       // live PID if active, 0 if detached
	Status    string    // "active" or "detached" (computed, never stored)
	StartedAt time.Time // from sessions/{pid}.json
	ModTime   time.Time // JSONL file mtime (proxy for last-accessed)
}

// claudeDir returns ~/.claude
func claudeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude"), nil
}

// pidInfo holds data from ~/.claude/sessions/{pid}.json
type pidInfo struct {
	PID       int
	SessionID string
	Cwd       string
	StartedAt time.Time
}

// scanPIDFiles reads all ~/.claude/sessions/*.json and returns a map of sessionId → pidInfo.
// For live PIDs, PID is set. For dead PIDs, PID is 0 but Cwd/StartedAt are preserved.
func scanPIDFiles(claudeBase string) (map[string]pidInfo, error) {
	dir := filepath.Join(claudeBase, "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	result := make(map[string]pidInfo)
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var raw struct {
			PID       int    `json:"pid"`
			SessionID string `json:"sessionId"`
			Cwd       string `json:"cwd"`
			StartedAt int64  `json:"startedAt"`
		}
		if err := json.Unmarshal(data, &raw); err != nil || raw.SessionID == "" {
			continue
		}
		info := pidInfo{
			PID:       raw.PID,
			SessionID: raw.SessionID,
			Cwd:       raw.Cwd,
			StartedAt: time.UnixMilli(raw.StartedAt),
		}
		// Always store with PID=0 since we use tmux for active detection now.
		// Keep Cwd/StartedAt from the most recent PID file (highest PID number).
		info.PID = 0
		if _, exists := result[raw.SessionID]; !exists {
			result[raw.SessionID] = info
		}
	}
	return result, nil
}

// customTitle holds the name and sessionId from a JSONL first line.
type customTitle struct {
	Type      string `json:"type"`
	Title     string `json:"customTitle"`
	SessionID string `json:"sessionId"`
}

// readTitle reads the first line of a JSONL file and extracts the custom-title if present.
func readTitle(path string) (customTitle, error) {
	f, err := os.Open(path)
	if err != nil {
		return customTitle{}, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return customTitle{}, fmt.Errorf("empty file")
	}

	var ct customTitle
	if err := json.Unmarshal(scanner.Bytes(), &ct); err != nil {
		return customTitle{}, err
	}
	if ct.Type != "custom-title" {
		return customTitle{}, fmt.Errorf("first line is not custom-title")
	}
	return ct, nil
}

// ListSessions returns all Claude Code sessions with custom titles,
// sorted by ModTime descending (most recent first).
func ListSessions() ([]Session, error) {
	base, err := claudeDir()
	if err != nil {
		return nil, err
	}

	pidMap, err := scanPIDFiles(base)
	if err != nil {
		return nil, fmt.Errorf("scanning PID files: %w", err)
	}

	projectsDir := filepath.Join(base, "projects")
	projectEntries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []Session
	for _, projEntry := range projectEntries {
		if !projEntry.IsDir() {
			continue
		}
		projDir := filepath.Join(projectsDir, projEntry.Name())
		jsonlFiles, err := filepath.Glob(filepath.Join(projDir, "*.jsonl"))
		if err != nil {
			continue
		}
		for _, jf := range jsonlFiles {
			ct, err := readTitle(jf)
			if err != nil || ct.Title == "" {
				continue // skip unnamed sessions
			}

			fi, err := os.Stat(jf)
			if err != nil {
				continue
			}

			sess := Session{
				Name:      ct.Title,
				SessionID: ct.SessionID,
				Project:   projEntry.Name(),
				ModTime:   fi.ModTime(),
				Status:    "detached",
			}

			if pi, ok := pidMap[ct.SessionID]; ok {
				sess.Cwd = pi.Cwd
				sess.StartedAt = pi.StartedAt
			}

			if sess.Cwd == "" {
				sess.Cwd = DecodeProjectHash(projEntry.Name())
			}

			sessions = append(sessions, sess)
		}
	}

	// Check tmux for active sessions
	tmuxBin, _ := exec.LookPath("tmux")
	tmuxSessions := session.ListTmuxSessions(tmuxBin)
	for i := range sessions {
		tmuxName := session.TmuxSessionName(sessions[i].Name)
		if tmuxSessions[tmuxName] {
			sessions[i].Status = "active"
		}
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ModTime.After(sessions[j].ModTime)
	})
	return sessions, nil
}

// ParseSessionRef splits "name:idprefix" into (name, idprefix).
// If there's no colon or the colon is trailing, idprefix is empty
// and the colon is stripped from the name.
func ParseSessionRef(ref string) (name, idPrefix string) {
	if idx := strings.LastIndex(ref, ":"); idx > 0 {
		if idx < len(ref)-1 {
			return ref[:idx], ref[idx+1:]
		}
		// Trailing colon — strip it, treat as name-only
		return ref[:idx], ""
	}
	return ref, ""
}

// GetSessionByRef finds a session by name, optionally narrowed by session ID prefix.
// Returns the session and the total count of sessions matching the name.
func GetSessionByRef(ref string) (sess *Session, nameMatchCount int, err error) {
	name, idPrefix := ParseSessionRef(ref)
	sessions, err := ListSessions()
	if err != nil {
		return nil, 0, err
	}

	var nameMatches []Session
	for _, s := range sessions {
		if s.Name == name {
			nameMatches = append(nameMatches, s)
		}
	}

	if len(nameMatches) == 0 {
		return nil, 0, fmt.Errorf("session %q not found", name)
	}

	if idPrefix != "" {
		for i := range nameMatches {
			if strings.HasPrefix(nameMatches[i].SessionID, idPrefix) {
				return &nameMatches[i], len(nameMatches), nil
			}
		}
		return nil, len(nameMatches), fmt.Errorf("no session %q with ID prefix %q", name, idPrefix)
	}

	return &nameMatches[0], len(nameMatches), nil
}

// ListSessionRefs returns session refs for tab completion.
// For duplicate names, includes "name:shortid" variants.
func ListSessionRefs(prefix string) ([]string, error) {
	sessions, err := ListSessions()
	if err != nil {
		return nil, err
	}
	nameCounts := make(map[string]int)
	for _, s := range sessions {
		nameCounts[s.Name]++
	}
	seen := make(map[string]bool)
	var refs []string
	for _, s := range sessions {
		shortID := s.SessionID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		ref := s.Name + ":" + shortID
		if !strings.HasPrefix(s.Name, prefix) && !strings.HasPrefix(ref, prefix) {
			continue
		}
		if nameCounts[s.Name] > 1 {
			refs = append(refs, ref)
		} else if !seen[s.Name] {
			seen[s.Name] = true
			refs = append(refs, s.Name)
		}
	}
	sort.Strings(refs)
	return refs, nil
}

// GetClaudeBin returns the claude binary from MUXC_CLAUDE_BIN env or PATH.
func GetClaudeBin() (string, error) {
	if bin := os.Getenv("MUXC_CLAUDE_BIN"); bin != "" {
		return bin, nil
	}
	path, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf("claude not found in PATH; set MUXC_CLAUDE_BIN env var")
	}
	return path, nil
}

// DecodeProjectHash converts a project hash back to an absolute path.
// e.g., "-home-dev-git-muxc" → "/home/dev/git/muxc"
//
// This is best-effort and display-only. The encoding replaces "/" with "-",
// which is ambiguous for paths containing dashes (e.g., "/home/dev/my-project"
// encodes the same as "/home/dev/my/project"). The stat check catches most
// false positives, but falls back to the raw hash when decoding fails.
//
// For accurate paths, prefer Session.Cwd (populated from PID files) over this.
// This function is only used as a fallback when no PID file data is available.
func DecodeProjectHash(hash string) string {
	candidate := strings.ReplaceAll(hash, "-", "/")
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate
	}
	return hash
}
