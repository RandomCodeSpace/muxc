package cmd

import (
	"fmt"
	"os"
	"time"

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

	// Create session record (session_id will be updated after Claude starts)
	now := time.Now()
	sess := &store.Session{
		Name:       name,
		SessionID:  "", // populated after Claude starts
		ClaudePID:  0,
		Cwd:        cwd,
		Status:     "active",
		ClaudeArgs: claudeArgs,
		Tags:       newTags,
		CreatedAt:  now,
		AccessedAt: now,
		History: []store.HistoryEntry{
			{Timestamp: now, Event: "created"},
		},
	}

	if err := db.CreateSession(sess); err != nil {
		return fmt.Errorf("creating session: %w", err)
	}

	// Resolve claude binary
	claudeBin, err := cfg.GetClaudeBin()
	if err != nil {
		return err
	}

	// Build claude args — use --name so Claude Code names the session
	execArgs := []string{"--name", name}
	execArgs = append(execArgs, claudeArgs...)

	ui.Launch("Creating session %q", name)

	// Run Claude as child process (captures session ID during execution)
	result := session.RunClaude(claudeBin, execArgs, cwd)

	// Update session with real PID and session ID from Claude
	if result.PID > 0 {
		sess.ClaudePID = result.PID
		sess.SessionID = result.SessionID
		_ = db.UpdateSession(sess)
	}

	// Mark session as detached now that Claude has exited
	sess.Status = "detached"
	sess.ClaudePID = 0
	sess.AccessedAt = time.Now()
	_ = db.UpdateSession(sess)
	_ = db.AppendHistory(name, "detached", "claude process exited")

	if sess.SessionID == "" {
		ui.Warn("Could not capture Claude session ID — resume may not work for session %q", name)
	}

	return result.Err
}
