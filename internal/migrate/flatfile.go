package migrate

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/RandomCodeSpace/muxc/internal/store"
)

// Result holds migration statistics.
type Result struct {
	Migrated int
	Skipped  int
	Errors   int
}

// FlatFiles reads legacy ~/.muxc/sessions/<name>/ directories and returns Session objects.
func FlatFiles(sessionsDir string) ([]store.Session, []error) {
	var sessions []store.Session
	var errs []error

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil, []error{fmt.Errorf("reading sessions dir: %w", err)}
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		dir := filepath.Join(sessionsDir, name)

		sess, err := parseSession(name, dir)
		if err != nil {
			errs = append(errs, fmt.Errorf("session %q: %w", name, err))
			continue
		}
		sessions = append(sessions, *sess)
	}

	return sessions, errs
}

func parseSession(name, dir string) (*store.Session, error) {
	meta, err := parseMeta(filepath.Join(dir, "meta"))
	if err != nil {
		return nil, fmt.Errorf("parsing meta: %w", err)
	}

	sess := &store.Session{
		Name:      name,
		SessionID: meta["session_id"],
		Cwd:       meta["cwd"],
		Status:    meta["status"],
		Notes:     readFileContents(filepath.Join(dir, "notes")),
	}

	// Parse timestamps
	if t, err := time.Parse(time.RFC3339, meta["created_at"]); err == nil {
		sess.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, meta["accessed_at"]); err == nil {
		sess.AccessedAt = t
	}

	// Decode claude_args from base64 to JSON array
	if encoded := meta["claude_args"]; encoded != "" {
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err == nil && len(decoded) > 0 {
			// The decoded string is space-separated args; convert to JSON array
			args := strings.Fields(string(decoded))
			jsonBytes, _ := json.Marshal(args)
			sess.ClaudeArgs = string(jsonBytes)
		}
	}

	// Parse tags
	tagsFile := filepath.Join(dir, "tags")
	if tags, err := readLines(tagsFile); err == nil {
		for _, tag := range tags {
			if tag != "" {
				sess.Tags = append(sess.Tags, store.Tag{Value: tag})
			}
		}
	}

	// Parse history
	historyFile := filepath.Join(dir, "history")
	if lines, err := readLines(historyFile); err == nil {
		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "\t", 3)
			entry := store.HistoryEntry{
				Event: "unknown",
			}
			if len(parts) >= 1 {
				if t, err := time.Parse(time.RFC3339, parts[0]); err == nil {
					entry.Timestamp = t
				}
			}
			if len(parts) >= 2 {
				entry.Event = parts[1]
			}
			if len(parts) >= 3 {
				entry.Details = parts[2]
			}
			sess.History = append(sess.History, entry)
		}
	}

	return sess, nil
}

func parseMeta(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	meta := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := line[:idx]
		value := line[idx+1:]
		// Strip surrounding quotes
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}
		meta[key] = value
	}
	return meta, scanner.Err()
}

func readFileContents(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}
