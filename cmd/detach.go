package cmd

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/RandomCodeSpace/muxc/internal/session"
	"github.com/RandomCodeSpace/muxc/internal/ui"
)

var detachCmd = &cobra.Command{
	Use:               "detach <name>",
	Short:             "Detach an active Claude session",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: sessionNameCompletion,
	RunE:              detachRun,
}

func init() {
	rootCmd.AddCommand(detachCmd)
}

func detachRun(cmd *cobra.Command, args []string) error {
	name := args[0]

	sess, err := db.GetSession(name)
	if err != nil {
		return fmt.Errorf("session %q not found: %w", name, err)
	}

	if sess.Status != "active" {
		return fmt.Errorf("session %q is not active (status: %s)", name, sess.Status)
	}

	// If PID is alive, send SIGTERM
	if sess.ClaudePID > 0 && session.CheckPID(sess.ClaudePID) {
		proc, err := os.FindProcess(sess.ClaudePID)
		if err == nil {
			if err := proc.Signal(syscall.SIGTERM); err != nil {
				ui.Warn("failed to send SIGTERM to PID %d: %v", sess.ClaudePID, err)
			}
		}
	}

	// Update session
	sess.Status = "detached"
	sess.ClaudePID = 0
	sess.AccessedAt = time.Now()
	if err := db.UpdateSession(sess); err != nil {
		return fmt.Errorf("updating session: %w", err)
	}

	// Append "detached" history
	if err := db.AppendHistory(name, "detached", ""); err != nil {
		return fmt.Errorf("appending history: %w", err)
	}

	ui.Success("Session %q detached", name)
	return nil
}
