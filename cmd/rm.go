package cmd

import (
	"fmt"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/RandomCodeSpace/muxc/internal/session"
	"github.com/RandomCodeSpace/muxc/internal/ui"
)

var rmCmd = &cobra.Command{
	Use:               "rm <name>",
	Short:             "Remove a session from the database",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: sessionNameCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		force, _ := cmd.Flags().GetBool("force")

		sess, err := db.GetSession(name)
		if err != nil {
			return err
		}

		if sess.Status == "active" && session.CheckPID(sess.ClaudePID) {
			if !force {
				return fmt.Errorf("session %q is active (PID %d); use --force to kill and remove", name, sess.ClaudePID)
			}
			// Kill the process first
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
		}

		if err := db.DeleteSession(name); err != nil {
			return err
		}

		ui.Success("Removed session %q", name)
		return nil
	},
}

func init() {
	rmCmd.Flags().BoolP("force", "f", false, "Kill active session before removing")
	rootCmd.AddCommand(rmCmd)
}
