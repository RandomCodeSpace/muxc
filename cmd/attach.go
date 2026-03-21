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

var attachCmd = &cobra.Command{
	Use:               "attach [<name>]",
	Short:             "Attach to an existing Claude session",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: sessionNameCompletion,
	RunE:              attachRun,
}

func init() {
	attachCmd.Flags().BoolP("force", "f", false, "Detach active session before reattaching")
	rootCmd.AddCommand(attachCmd)
}

func attachRun(cmd *cobra.Command, args []string) error {
	var name string

	if len(args) == 0 {
		// Interactive picker if stdin is a TTY
		sessions, err := db.ListSessions("", "", false)
		if err != nil {
			return err
		}
		if len(sessions) == 0 {
			ui.Info("No sessions to attach to.")
			return nil
		}

		rows := make([]ui.SessionRow, len(sessions))
		for i, s := range sessions {
			rows[i] = ui.SessionRow{
				Status:   s.Status,
				Name:     s.Name,
				Cwd:      ui.ShortenPath(s.Cwd),
				Accessed: ui.RelativeTime(s.AccessedAt),
			}
		}

		picked, err := ui.PickSession(rows)
		if err != nil {
			return fmt.Errorf("no session selected")
		}
		name = picked
	} else {
		name = args[0]
	}

	sess, err := db.GetSession(name)
	if err != nil {
		return fmt.Errorf("session %q not found: %w", name, err)
	}

	// If session is marked active, check if PID is actually alive
	if sess.Status == "active" {
		if session.CheckPID(sess.ClaudePID) {
			force, _ := cmd.Flags().GetBool("force")
			if !force {
				return fmt.Errorf("session %q is already active (PID %d); detach it first (or use --force)", name, sess.ClaudePID)
			}
			// Force-detach: send SIGTERM to the existing Claude process
			if proc, err := os.FindProcess(sess.ClaudePID); err == nil {
				if err := proc.Signal(syscall.SIGTERM); err != nil {
					ui.Warn("failed to send SIGTERM to PID %d: %v", sess.ClaudePID, err)
				}
			}
			sess.Status = "detached"
			sess.ClaudePID = 0
			if err := db.UpdateSession(sess); err != nil {
				return fmt.Errorf("updating session: %w", err)
			}
			_ = db.AppendHistory(name, "force-detached", "detached by attach --force")
			ui.Success("Force-detached session %q", name)
		} else {
			// PID is dead — transition to detached
			sess.Status = "detached"
			sess.ClaudePID = 0
			if err := db.UpdateSession(sess); err != nil {
				return fmt.Errorf("updating stale session: %w", err)
			}
			_ = db.AppendHistory(name, "reaped", "PID was dead on attach")
		}
	}

	// Update session to active
	sess.Status = "active"
	sess.ClaudePID = os.Getpid()
	sess.AccessedAt = time.Now()
	if err := db.UpdateSession(sess); err != nil {
		return fmt.Errorf("updating session: %w", err)
	}

	// Append "attached" history
	if err := db.AppendHistory(name, "attached", ""); err != nil {
		return fmt.Errorf("appending history: %w", err)
	}

	// Resolve claude binary
	claudeBin, err := cfg.GetClaudeBin()
	if err != nil {
		return err
	}

	// Build exec args: --resume <session_id> plus decoded claude args
	execArgs := []string{"--resume", sess.SessionID}
	execArgs = append(execArgs, sess.ClaudeArgs...)

	ui.Launch("Attaching to session %q", name)

	// Exec claude (does not return on success)
	return session.ExecClaude(claudeBin, execArgs, sess.Cwd)
}
