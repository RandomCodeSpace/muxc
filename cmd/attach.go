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

	// Resolve claude binary
	claudeBin, err := cfg.GetClaudeBin()
	if err != nil {
		return err
	}

	// Update session to active
	sess.Status = "active"
	sess.AccessedAt = time.Now()
	if err := db.UpdateSession(sess); err != nil {
		return fmt.Errorf("updating session: %w", err)
	}

	var result session.RunResult

	// Try to resume if we have a session ID, otherwise start fresh
	if sess.SessionID != "" {
		if err := db.AppendHistory(name, "attached", fmt.Sprintf("resuming session %s", sess.SessionID)); err != nil {
			return fmt.Errorf("appending history: %w", err)
		}

		execArgs := []string{"--resume", sess.SessionID}
		execArgs = append(execArgs, sess.ClaudeArgs...)

		ui.Launch("Attaching to session %q", name)

		result = session.RunClaudeResume(claudeBin, execArgs, sess.Cwd)

		// If resume failed, fall back to starting a fresh session
		if result.ResumeFailure {
			ui.Warn("Could not resume session %q — starting fresh in %s", name, ui.ShortenPath(sess.Cwd))
			_ = db.AppendHistory(name, "resume-failed", fmt.Sprintf("session ID %s not found by Claude", sess.SessionID))

			result = launchFresh(claudeBin, name, sess.Cwd, sess.ClaudeArgs)
		}
	} else {
		ui.Warn("No session ID for %q — starting fresh in %s", name, ui.ShortenPath(sess.Cwd))
		_ = db.AppendHistory(name, "attached", "no session ID, starting fresh")

		result = launchFresh(claudeBin, name, sess.Cwd, sess.ClaudeArgs)
	}

	// Update PID and session ID if we got them
	if result.PID > 0 {
		sess.ClaudePID = result.PID
		if result.SessionID != "" {
			sess.SessionID = result.SessionID
		}
		_ = db.UpdateSession(sess)
	}

	// Mark session as detached now that Claude has exited
	sess.Status = "detached"
	sess.ClaudePID = 0
	sess.AccessedAt = time.Now()
	_ = db.UpdateSession(sess)
	_ = db.AppendHistory(name, "detached", "claude process exited")

	return result.Err
}

// launchFresh starts a new Claude session in the given directory with all original args.
func launchFresh(claudeBin, name, cwd string, claudeArgs []string) session.RunResult {
	execArgs := []string{"--name", name}
	execArgs = append(execArgs, claudeArgs...)
	return session.RunClaude(claudeBin, execArgs, cwd)
}
