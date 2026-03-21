package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/RandomCodeSpace/muxc/internal/session"
	"github.com/RandomCodeSpace/muxc/internal/store"
	"github.com/RandomCodeSpace/muxc/internal/ui"
)

var newCmd = &cobra.Command{
	Use:   "new <name> [flags] [-- <claude-args>...]",
	Short: "Create and launch a new Claude session",
	Args:  cobra.ArbitraryArgs,
	RunE:  newRun,
}

var (
	newCwd  string
	newTags []string
)

func init() {
	newCmd.Flags().StringVar(&newCwd, "cwd", "", "working directory for the session (default: current dir)")
	newCmd.Flags().StringSliceVar(&newTags, "tag", nil, "tags for the session (repeatable)")
	rootCmd.AddCommand(newCmd)
}

func newRun(cmd *cobra.Command, args []string) error {
	// Determine positional args and claude passthrough args
	var name string
	var claudeArgs []string

	dashIdx := cmd.ArgsLenAtDash()
	if dashIdx == -1 {
		// No "--" separator
		if len(args) < 1 {
			return fmt.Errorf("session name is required")
		}
		name = args[0]
	} else {
		// Args before "--" are positional, args after are claude passthrough
		positional := args[:dashIdx]
		claudeArgs = args[dashIdx:]
		if len(positional) < 1 {
			return fmt.Errorf("session name is required")
		}
		name = positional[0]
	}

	// Resolve working directory
	cwd := newCwd
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
	}

	// Generate session UUID
	sessionID := uuid.New().String()

	// Marshal claude args as JSON array
	claudeArgsJSON, err := json.Marshal(claudeArgs)
	if err != nil {
		return fmt.Errorf("marshaling claude args: %w", err)
	}

	// Create session in DB
	now := time.Now()
	sess := &store.Session{
		Name:       name,
		SessionID:  sessionID,
		ClaudePID:  os.Getpid(),
		Cwd:        cwd,
		Status:     "active",
		ClaudeArgs: string(claudeArgsJSON),
		AccessedAt: now,
	}

	if err := db.CreateSession(sess); err != nil {
		return fmt.Errorf("creating session: %w", err)
	}

	// Add tags
	for _, t := range newTags {
		if err := db.AddTag(sess.ID, t); err != nil {
			return fmt.Errorf("adding tag %q: %w", t, err)
		}
	}

	// Append "created" history entry
	if err := db.AppendHistory(sess.ID, "created", ""); err != nil {
		return fmt.Errorf("appending history: %w", err)
	}

	// Resolve claude binary
	claudeBin, err := cfg.GetClaudeBin()
	if err != nil {
		return err
	}

	// Build claude args
	execArgs := []string{"--session-id", sessionID, "--name", name}
	execArgs = append(execArgs, claudeArgs...)

	ui.Launch("Creating session %q (id: %s)", name, sessionID[:8])

	// Close DB before exec (exec replaces the process)
	db.Close()
	db = nil

	// Exec claude (does not return on success)
	return session.ExecClaude(claudeBin, execArgs, cwd)
}
