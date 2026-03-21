package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/RandomCodeSpace/muxc/internal/migrate"
	"github.com/RandomCodeSpace/muxc/internal/ui"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "📦 Migrate legacy flat-file sessions to SQLite",
	Long:  "Reads ~/.muxc/sessions/<name>/ flat files from the Bash version and imports them into the SQLite database.",
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionsDir := filepath.Join(cfg.DataDir, "sessions")
		if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
			ui.Info("No legacy sessions directory found at %s", sessionsDir)
			return nil
		}

		ui.Action("Scanning %s for legacy sessions...", sessionsDir)

		sessions, errs := migrate.FlatFiles(sessionsDir)
		for _, err := range errs {
			ui.Warn("Parse error: %v", err)
		}

		if len(sessions) == 0 {
			ui.Info("No sessions found to migrate.")
			return nil
		}

		migrated := 0
		skipped := 0

		for i := range sessions {
			sess := &sessions[i]
			// Check if session already exists by UUID
			existing, _ := db.GetSessionByID(sess.SessionID)
			if existing != nil {
				skipped++
				continue
			}

			if err := db.CreateSession(sess); err != nil {
				ui.Warn("Failed to migrate %q: %v", sess.Name, err)
				continue
			}

			// Add tags (they're created with the session via GORM associations)
			// Append migration history
			_ = db.AppendHistory(sess.ID, "migrated", "source=flat-files")
			migrated++
		}

		ui.Success("Migration complete: %d migrated, %d skipped (already exist), %d errors",
			migrated, skipped, len(errs))

		// Offer to rename old directory
		backupDir := sessionsDir + ".bak"
		if _, err := os.Stat(backupDir); os.IsNotExist(err) {
			fmt.Printf("\n📂 Rename %s → %s? [y/N] ", sessionsDir, backupDir)
			var answer string
			fmt.Scanln(&answer)
			if answer == "y" || answer == "Y" {
				if err := os.Rename(sessionsDir, backupDir); err != nil {
					ui.Warn("Failed to rename: %v", err)
				} else {
					ui.Success("Renamed to %s", backupDir)
				}
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
