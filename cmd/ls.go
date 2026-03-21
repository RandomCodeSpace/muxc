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

	fmt.Printf("%-3s %-20s %-30s %-14s %s\n", " ", "NAME", "CWD", "ACCESSED", "TAGS")
	fmt.Printf("%-3s %-20s %-30s %-14s %s\n", " ", "----", "---", "--------", "----")

	for _, s := range sessions {
		icon := ui.StatusIcon(s.Status)
		tags := "-"
		if len(s.Tags) > 0 {
			tagValues := make([]string, len(s.Tags))
			for i, t := range s.Tags {
				tagValues[i] = t.Value
			}
			tags = strings.Join(tagValues, ", ")
		}
		fmt.Printf("%-3s %-20s %-30s %-14s %s\n",
			icon,
			s.Name,
			ui.ShortenPath(s.Cwd),
			ui.RelativeTime(s.AccessedAt),
			tags,
		)
	}

	ui.Info("%d session(s) listed.", len(sessions))
	return nil
}
