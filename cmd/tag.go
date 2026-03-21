package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RandomCodeSpace/muxc/internal/ui"
)

var tagCmd = &cobra.Command{
	Use:               "tag <name> <add|rm> <tag>",
	Short:             "Add or remove tags on a session",
	Args:              cobra.ExactArgs(3),
	ValidArgsFunction: sessionNameCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		action := args[1]
		tagValue := args[2]

		switch action {
		case "add":
			if err := db.AddTag(name, tagValue); err != nil {
				return err
			}
			_ = db.AppendHistory(name, "tag-add", tagValue)
			ui.Success("Added tag %q to session %q", tagValue, name)

		case "rm":
			if err := db.RemoveTag(name, tagValue); err != nil {
				return err
			}
			_ = db.AppendHistory(name, "tag-rm", tagValue)
			ui.Success("Removed tag %q from session %q", tagValue, name)

		default:
			return fmt.Errorf("unknown action %q: use \"add\" or \"rm\"", action)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)
}
