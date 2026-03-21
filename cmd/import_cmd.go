package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/RandomCodeSpace/muxc/internal/session"
	"github.com/RandomCodeSpace/muxc/internal/store"
	"github.com/RandomCodeSpace/muxc/internal/ui"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "📥 Adopt orphaned Claude Code sessions",
	Long:  "Scans ~/.claude/sessions/ for sessions not tracked by muxc and offers to adopt them.",
	RunE:  importRun,
}

var importScanOnly bool

func init() {
	importCmd.Flags().BoolVar(&importScanOnly, "scan", false, "only show orphaned sessions, don't adopt them")
	rootCmd.AddCommand(importCmd)
}

type claudeSession struct {
	PID       int    `json:"pid"`
	SessionID string `json:"sessionId"`
	Cwd       string `json:"cwd"`
	StartedAt int64  `json:"startedAt"`
}

func importRun(cmd *cobra.Command, args []string) error {
	homeDir, _ := os.UserHomeDir()
	claudeDir := filepath.Join(homeDir, ".claude", "sessions")

	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		return fmt.Errorf("claude sessions directory not found: %s", claudeDir)
	}

	fmt.Println("🔍 Scanning for Claude Code sessions...")
	fmt.Println()

	// Collect known session IDs from muxc
	knownIDs := make(map[string]bool)
	allSessions, _ := db.ListSessions("", "", true)
	for _, s := range allSessions {
		knownIDs[s.SessionID] = true
	}

	// Scan Claude session files
	entries, err := os.ReadDir(claudeDir)
	if err != nil {
		return fmt.Errorf("reading claude sessions: %w", err)
	}

	orphanCount := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(claudeDir, entry.Name()))
		if err != nil {
			continue
		}

		var cs claudeSession
		if err := json.Unmarshal(data, &cs); err != nil {
			continue
		}

		if cs.SessionID == "" || knownIDs[cs.SessionID] {
			continue
		}

		orphanCount++
		shortCwd := ui.ShortenPath(cs.Cwd)

		// Convert epoch ms to time
		timeStr := "unknown"
		if cs.StartedAt > 0 {
			t := time.UnixMilli(cs.StartedAt)
			timeStr = t.UTC().Format(time.RFC3339)
		}

		alive := "dead"
		if cs.PID > 0 && session.CheckPID(cs.PID) {
			alive = "alive"
		}

		fmt.Printf("   📥 Session: %s...\n", cs.SessionID[:8])
		fmt.Printf("      Directory: %s\n", shortCwd)
		fmt.Printf("      Started:   %s\n", timeStr)
		fmt.Printf("      Process:   %d (%s)\n", cs.PID, alive)
		fmt.Println()

		if !importScanOnly {
			// Suggest name from directory
			suggested := filepath.Base(cs.Cwd)

			fmt.Printf("   Name for this session [%s]: ", suggested)
			var name string
			fmt.Scanln(&name)
			if name == "" {
				name = suggested
			}

			// Check if name exists
			if existing, _ := db.GetSession(name); existing != nil {
				ui.Warn("Session %q already exists, skipping", name)
				continue
			}

			status := "detached"
			pid := 0
			if alive == "alive" {
				status = "active"
				pid = cs.PID
			}

			startTime := time.Now().UTC()
			if cs.StartedAt > 0 {
				startTime = time.UnixMilli(cs.StartedAt).UTC()
			}

			sess := &store.Session{
				Name:       name,
				SessionID:  cs.SessionID,
				ClaudePID:  pid,
				Cwd:        cs.Cwd,
				Status:     status,
				CreatedAt:  startTime,
				AccessedAt: time.Now().UTC(),
			}

			if err := db.CreateSession(sess); err != nil {
				ui.Warn("Failed to import: %v", err)
				continue
			}
			_ = db.AppendHistory(sess.ID, "imported", "source=claude-sessions")
			ui.Success("Imported as %q", name)
			fmt.Println()
		}
	}

	if orphanCount == 0 {
		ui.Info("No orphaned sessions found. All Claude sessions are tracked by muxc.")
	} else if importScanOnly {
		ui.Info("Found %d orphaned session(s). Run 'muxc import' (without --scan) to adopt them.", orphanCount)
	}

	return nil
}
