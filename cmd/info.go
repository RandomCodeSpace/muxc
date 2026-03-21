package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RandomCodeSpace/muxc/internal/session"
	"github.com/RandomCodeSpace/muxc/internal/ui"
)

var infoCmd = &cobra.Command{
	Use:               "info <name>",
	Short:             "Show detailed information about a session",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: sessionNameCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		sess, err := db.GetSession(name)
		if err != nil {
			return err
		}

		// Check PID liveness and transition if dead
		if sess.Status == "active" && !session.CheckPID(sess.ClaudePID) {
			sess.Status = "detached"
			sess.ClaudePID = 0
			_ = db.UpdateSession(sess)
		}

		// Session name
		fmt.Printf("ℹ️  Session: %s\n", sess.Name)

		// Status line
		statusLine := fmt.Sprintf("   Status: %s %s", ui.StatusIcon(sess.Status), sess.Status)
		if sess.Status == "active" && sess.ClaudePID > 0 {
			statusLine += fmt.Sprintf(" (PID %d)", sess.ClaudePID)
		}
		fmt.Println(statusLine)

		// Session ID
		fmt.Printf("   Session ID: %s\n", sess.SessionID)

		// Directory
		fmt.Printf("   Directory: %s\n", ui.ShortenPath(sess.Cwd))

		// Timestamps
		fmt.Printf("   Created: %s (%s)\n", sess.CreatedAt.Format("2006-01-02 15:04:05"), ui.RelativeTime(sess.CreatedAt))
		fmt.Printf("   Accessed: %s (%s)\n", sess.AccessedAt.Format("2006-01-02 15:04:05"), ui.RelativeTime(sess.AccessedAt))

		// Claude args
		if sess.ClaudeArgs != "" {
			var decoded []string
			if err := json.Unmarshal([]byte(sess.ClaudeArgs), &decoded); err == nil {
				fmt.Printf("   Claude args: %s\n", strings.Join(decoded, " "))
			} else {
				fmt.Printf("   Claude args: %s\n", sess.ClaudeArgs)
			}
		}

		// Tags
		if len(sess.Tags) > 0 {
			tags := make([]string, len(sess.Tags))
			for i, t := range sess.Tags {
				tags[i] = t.Value
			}
			fmt.Printf("🏷️  Tags: %s\n", strings.Join(tags, ", "))
		} else {
			fmt.Println("🏷️  Tags: (none)")
		}

		// Notes
		if sess.Notes != "" {
			fmt.Println("📝 Notes:")
			for _, line := range strings.Split(sess.Notes, "\n") {
				fmt.Printf("   %s\n", line)
			}
		} else {
			fmt.Println("📝 Notes: (none)")
		}

		// Recent history (last 10)
		fmt.Println("📜 Recent history:")
		history := sess.History
		start := 0
		if len(history) > 10 {
			start = len(history) - 10
		}
		if len(history) == 0 {
			fmt.Println("   (none)")
		} else {
			for _, h := range history[start:] {
				ts := h.Timestamp.Format("2006-01-02 15:04:05")
				if h.Details != "" {
					fmt.Printf("   %s  %-12s %s\n", ts, h.Event, h.Details)
				} else {
					fmt.Printf("   %s  %s\n", ts, h.Event)
				}
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
