package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RandomCodeSpace/muxc/internal/claude"
	"github.com/RandomCodeSpace/muxc/internal/ui"
)

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list", "l"},
	Short:   "List sessions",
	RunE:    lsRun,
}

func init() {
	lsCmd.Flags().StringP("status", "s", "", "filter by status (active/detached)")
	rootCmd.AddCommand(lsCmd)
}

func lsRun(cmd *cobra.Command, args []string) error {
	status, _ := cmd.Flags().GetString("status")

	sessions, err := claude.ListSessions()
	if err != nil {
		return err
	}

	// Filter by status if requested
	if status != "" {
		var filtered []claude.Session
		for _, s := range sessions {
			if s.Status == status {
				filtered = append(filtered, s)
			}
		}
		sessions = filtered
	}

	if len(sessions) == 0 {
		ui.Info("No sessions found.")
		return nil
	}

	rows := make([]ui.SessionRow, len(sessions))
	for i, s := range sessions {
		shortID := s.SessionID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		rows[i] = ui.SessionRow{
			Status:   s.Status,
			Name:     s.Name,
			ShortID:  shortID,
			Cwd:      ui.ShortenPath(s.Cwd),
			Accessed: ui.RelativeTime(s.ModTime),
		}
	}

	fmt.Println()
	ui.RenderSessionTable(rows)
	return nil
}
