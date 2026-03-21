package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RandomCodeSpace/muxc/internal/ui"
)

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list", "l"},
	Short:   "List sessions",
	RunE:    lsRun,
}

func init() {
	lsCmd.Flags().StringP("tag", "t", "", "filter by tag")
	lsCmd.Flags().StringP("status", "s", "", "filter by status")
	lsCmd.Flags().BoolP("all", "a", false, "include archived sessions")
	rootCmd.AddCommand(lsCmd)
}

func lsRun(cmd *cobra.Command, args []string) error {
	tag, _ := cmd.Flags().GetString("tag")
	status, _ := cmd.Flags().GetString("status")
	showAll, _ := cmd.Flags().GetBool("all")

	sessions, err := db.ListSessions(status, tag, showAll)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		ui.Info("No sessions found.")
		return nil
	}

	rows := make([]ui.SessionRow, len(sessions))
	for i, s := range sessions {
		tags := "-"
		if len(s.Tags) > 0 {
			tagValues := make([]string, len(s.Tags))
			for j, t := range s.Tags {
				tagValues[j] = t.Value
			}
			tags = strings.Join(tagValues, ", ")
		}
		rows[i] = ui.SessionRow{
			Status:   s.Status,
			Name:     s.Name,
			Cwd:      ui.ShortenPath(s.Cwd),
			Accessed: ui.RelativeTime(s.AccessedAt),
			Tags:     tags,
		}
	}

	fmt.Println()
	ui.RenderSessionTable(rows)
	return nil
}
