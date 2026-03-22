package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RandomCodeSpace/muxc/internal/claude"
	"github.com/RandomCodeSpace/muxc/internal/session"
	"github.com/RandomCodeSpace/muxc/internal/ui"
)

var killCmd = &cobra.Command{
	Use:               "kill <name>",
	Short:             "Kill a running tmux session",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: sessionNameCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		parsedName, _ := claude.ParseSessionRef(name)
		tmuxName := session.TmuxSessionName(parsedName)

		if !session.TmuxHasSession(tmuxBin, tmuxName) {
			return fmt.Errorf("no active session %q", parsedName)
		}
		if err := session.TmuxKillSession(tmuxBin, tmuxName); err != nil {
			return fmt.Errorf("killing session: %w", err)
		}
		ui.Info("Killed session %q", parsedName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(killCmd)
}
