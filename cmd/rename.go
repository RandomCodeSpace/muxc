package cmd

import (
	"fmt"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/RandomCodeSpace/muxc/internal/session"
	"github.com/RandomCodeSpace/muxc/internal/ui"
)

var validNameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

var renameCmd = &cobra.Command{
	Use:               "rename <old> <new>",
	Short:             "Rename a session",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: sessionNameCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		oldName := args[0]
		newName := args[1]

		// Validate new name
		if len(newName) > 64 {
			return fmt.Errorf("name %q exceeds 64 characters", newName)
		}
		if !validNameRe.MatchString(newName) {
			return fmt.Errorf("name %q is invalid: must be alphanumeric, hyphens, or underscores only", newName)
		}

		sess, err := db.GetSession(oldName)
		if err != nil {
			return err
		}

		if sess.Status == "active" && session.CheckPID(sess.ClaudePID) {
			return fmt.Errorf("cannot rename active session %q (PID %d is alive); kill it first", oldName, sess.ClaudePID)
		}

		sess.Name = newName
		if err := db.UpdateSession(sess); err != nil {
			return err
		}

		_ = db.AppendHistory(sess.ID, "renamed", fmt.Sprintf("%s -> %s", oldName, newName))

		ui.Success("Renamed session %q to %q", oldName, newName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
}
