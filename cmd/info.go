package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RandomCodeSpace/muxc/internal/claude"
	"github.com/RandomCodeSpace/muxc/internal/ui"
)

var infoCmd = &cobra.Command{
	Use:               "info <name>",
	Short:             "Show detailed information about a session",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: sessionNameCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		sess, err := claude.GetSession(name)
		if err != nil {
			return err
		}

		fmt.Printf("ℹ️  Session: %s\n", sess.Name)

		statusLine := fmt.Sprintf("   Status: %s %s", ui.StatusIcon(sess.Status), sess.Status)
		if sess.Status == "active" && sess.PID > 0 {
			statusLine += fmt.Sprintf(" (PID %d)", sess.PID)
		}
		fmt.Println(statusLine)

		fmt.Printf("   Session ID: %s\n", sess.SessionID)
		fmt.Printf("   Project: %s\n", claude.DecodeProjectHash(sess.Project))
		fmt.Printf("   Directory: %s\n", ui.ShortenPath(sess.Cwd))

		if !sess.StartedAt.IsZero() {
			fmt.Printf("   Started: %s (%s)\n", sess.StartedAt.Format("2006-01-02 15:04:05"), ui.RelativeTime(sess.StartedAt))
		}
		fmt.Printf("   Last modified: %s (%s)\n", sess.ModTime.Format("2006-01-02 15:04:05"), ui.RelativeTime(sess.ModTime))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
