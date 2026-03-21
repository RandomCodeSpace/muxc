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

		sess, err := db.GetSession(name)
		if err != nil {
			return err
		}

		switch action {
		case "add":
			if err := db.AddTag(sess.ID, tagValue); err != nil {
				return err
			}
			_ = db.AppendHistory(sess.ID, "tag-add", tagValue)
			ui.Success("Added tag %q to session %q", tagValue, name)

		case "rm":
			if err := db.RemoveTag(sess.ID, tagValue); err != nil {
				return err
			}
			_ = db.AppendHistory(sess.ID, "tag-rm", tagValue)
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
