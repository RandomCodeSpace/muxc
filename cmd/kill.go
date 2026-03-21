package cmd

import (
	"fmt"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/RandomCodeSpace/muxc/internal/session"
	"github.com/RandomCodeSpace/muxc/internal/ui"
)

var killCmd = &cobra.Command{
	Use:               "kill <name>",
	Short:             "Kill an active session's Claude process",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: sessionNameCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		force, _ := cmd.Flags().GetBool("force")

		sess, err := db.GetSession(name)
		if err != nil {
			return err
		}

		if sess.Status != "active" {
			return fmt.Errorf("session %q is not active (status: %s)", name, sess.Status)
		}

		if !session.CheckPID(sess.ClaudePID) {
			// PID is not alive, just mark detached
			sess.Status = "detached"
			sess.ClaudePID = 0
			if err := db.UpdateSession(sess); err != nil {
				return err
			}
			ui.Warn("PID %d is not alive; marked session %q as detached", sess.ClaudePID, name)
			return nil
		}

		sig := syscall.SIGTERM
		sigName := "SIGTERM"
		if force {
			sig = syscall.SIGKILL
			sigName = "SIGKILL"
		}

		if err := syscall.Kill(sess.ClaudePID, sig); err != nil {
			return fmt.Errorf("failed to send %s to PID %d: %w", sigName, sess.ClaudePID, err)
		}

		// Wait up to 5 seconds for process to die
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if !session.CheckPID(sess.ClaudePID) {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}

		if session.CheckPID(sess.ClaudePID) {
			ui.Warn("PID %d did not exit within 5s after %s", sess.ClaudePID, sigName)
		}

		sess.Status = "detached"
		sess.ClaudePID = 0
		if err := db.UpdateSession(sess); err != nil {
			return err
		}

		_ = db.AppendHistory(sess.ID, "killed", fmt.Sprintf("sent %s", sigName))

		ui.Success("Killed session %q (%s)", name, sigName)
		return nil
	},
}

func init() {
	killCmd.Flags().BoolP("force", "f", false, "Send SIGKILL instead of SIGTERM")
	rootCmd.AddCommand(killCmd)
}
