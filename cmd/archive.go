package cmd

import (
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/RandomCodeSpace/muxc/internal/session"
	"github.com/RandomCodeSpace/muxc/internal/ui"
)

var archiveCmd = &cobra.Command{
	Use:               "archive <name>",
	Short:             "Archive a session",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: sessionNameCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		sess, err := db.GetSession(name)
		if err != nil {
			return err
		}

		// If active and PID alive, kill it first
		if sess.Status == "active" && session.CheckPID(sess.ClaudePID) {
			_ = syscall.Kill(sess.ClaudePID, syscall.SIGTERM)
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				if !session.CheckPID(sess.ClaudePID) {
					break
				}
				time.Sleep(200 * time.Millisecond)
			}
			if session.CheckPID(sess.ClaudePID) {
				_ = syscall.Kill(sess.ClaudePID, syscall.SIGKILL)
			}
			sess.ClaudePID = 0
		}

		sess.Status = "archived"
		if err := db.UpdateSession(sess); err != nil {
			return err
		}

		_ = db.AppendHistory(name, "archived", "")

		ui.Success("Archived session %q", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(archiveCmd)
}
