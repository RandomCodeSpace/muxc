package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RandomCodeSpace/muxc/internal/ui"
)

var noteCmd = &cobra.Command{
	Use:               "note <name> [text...]",
	Short:             "Set or edit session notes",
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: sessionNameCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		sess, err := db.GetSession(name)
		if err != nil {
			return err
		}

		var noteText string

		if len(args) > 1 {
			// Text provided on command line — append to existing notes
			newText := strings.Join(args[1:], " ")
			if sess.Notes != "" {
				sess.Notes += "\n" + newText
			} else {
				sess.Notes = newText
			}
			noteText = newText
		} else {
			// Open $EDITOR
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vi"
			}

			tmpFile, err := os.CreateTemp("", "muxc-note-*.txt")
			if err != nil {
				return fmt.Errorf("failed to create temp file: %w", err)
			}
			tmpPath := tmpFile.Name()
			defer os.Remove(tmpPath)

			// Write existing notes to temp file
			if sess.Notes != "" {
				if _, err := tmpFile.WriteString(sess.Notes); err != nil {
					tmpFile.Close()
					return err
				}
			}
			tmpFile.Close()

			editorCmd := exec.Command(editor, tmpPath)
			editorCmd.Stdin = os.Stdin
			editorCmd.Stdout = os.Stdout
			editorCmd.Stderr = os.Stderr

			if err := editorCmd.Run(); err != nil {
				return fmt.Errorf("editor exited with error: %w", err)
			}

			content, err := os.ReadFile(tmpPath)
			if err != nil {
				return fmt.Errorf("failed to read temp file: %w", err)
			}

			sess.Notes = strings.TrimRight(string(content), "\n")
			noteText = "edited via $EDITOR"
		}

		if err := db.UpdateSession(sess); err != nil {
			return err
		}

		_ = db.AppendHistory(name, "note", noteText)

		ui.Success("Updated notes for session %q", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(noteCmd)
}
