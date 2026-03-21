package cmd

import (
	"encoding/json"
	"fmt"
	"os"
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
	rootCmd.AddCommand(attachCmd)
}

func attachRun(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("session name required (interactive picker coming soon)")
	}

	name := args[0]

	sess, err := db.GetSession(name)
	if err != nil {
		return fmt.Errorf("session %q not found: %w", name, err)
	}

	// If session is marked active, check if PID is actually alive
	if sess.Status == "active" {
		if session.CheckPID(sess.ClaudePID) {
			return fmt.Errorf("session %q is already active (PID %d); detach it first", name, sess.ClaudePID)
		}
		// PID is dead — transition to detached
		sess.Status = "detached"
		sess.ClaudePID = 0
		if err := db.UpdateSession(sess); err != nil {
			return fmt.Errorf("updating stale session: %w", err)
		}
		_ = db.AppendHistory(sess.ID, "reaped", "PID was dead on attach")
	}

	// Update session to active
	sess.Status = "active"
	sess.ClaudePID = os.Getpid()
	sess.AccessedAt = time.Now()
	if err := db.UpdateSession(sess); err != nil {
		return fmt.Errorf("updating session: %w", err)
	}

	// Append "attached" history
	if err := db.AppendHistory(sess.ID, "attached", ""); err != nil {
		return fmt.Errorf("appending history: %w", err)
	}

	// Decode ClaudeArgs from JSON array
	var claudeArgs []string
	if sess.ClaudeArgs != "" {
		if err := json.Unmarshal([]byte(sess.ClaudeArgs), &claudeArgs); err != nil {
			ui.Warn("failed to decode claude args: %v", err)
			claudeArgs = nil
		}
	}

	// Resolve claude binary
	claudeBin, err := cfg.GetClaudeBin()
	if err != nil {
		return err
	}

	// Build exec args: --resume <session_id> plus decoded claude args
	execArgs := []string{"--resume", sess.SessionID}
	execArgs = append(execArgs, claudeArgs...)

	ui.Launch("Attaching to session %q", name)

	// Close DB before exec (exec replaces the process)
	db.Close()
	db = nil

	// Exec claude (does not return on success)
	return session.ExecClaude(claudeBin, execArgs, sess.Cwd)
}
