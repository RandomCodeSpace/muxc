package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/RandomCodeSpace/muxc/internal/config"
	"github.com/RandomCodeSpace/muxc/internal/session"
	"github.com/RandomCodeSpace/muxc/internal/store"
	"github.com/RandomCodeSpace/muxc/internal/ui"
)

var (
	cfgFile string
	cfg     *config.Config
	db      *store.Store
	version string = "dev"
)

// reservedNames are subcommand names that cannot be used as session names.
var reservedNames = map[string]bool{
	"ls": true, "list": true, "l": true,
	"attach": true, "detach": true, "kill": true,
	"info": true, "tag": true, "note": true,
	"rename": true, "archive": true, "rm": true,
	"import": true, "completion": true, "version": true,
	"help": true, "new": true,
}

func SetVersion(v string) { version = v }

var rootCmd = &cobra.Command{
	Use:               "muxc [<session>] [flags] [-- <claude-args>...]",
	Short:             "muxc -- Claude Multiplexer for Claude Code",
	SilenceUsage:      true,
	SilenceErrors:     true,
	Args:              cobra.ArbitraryArgs,
	ValidArgsFunction: sessionNameCompletion,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip init for help/version/completion
		if cmd.Name() == "help" || cmd.Name() == "version" || cmd.Name() == "completion" {
			return nil
		}
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return err
		}
		db, err = store.Open(cfg.SessionsDir())
		if err != nil {
			return err
		}
		// Reap dead sessions on every invocation
		session.ReapDeadSessions(db)
		return nil
	},
	RunE: unifiedRun,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ui.Die("%v", err)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.muxc/config.yaml)")
	rootCmd.Flags().StringVar(&flagCwd, "cwd", "", "working directory for new session (default: current dir)")
	rootCmd.Flags().StringSliceVar(&flagTags, "tag", nil, "tags for new session (repeatable)")
	rootCmd.Flags().BoolVarP(&flagForce, "force", "f", false, "force-detach active session before reattaching")
}

var (
	flagCwd   string
	flagTags  []string
	flagForce bool
)

// sessionNameCompletion provides tab-completion for session names
func sessionNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if db == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names, _ := db.ListSessionNames(toComplete)
	return names, cobra.ShellCompDirectiveNoFileComp
}

// unifiedRun handles `muxc <name>` — attaches if session exists, creates if not.
func unifiedRun(cmd *cobra.Command, args []string) error {
	// No args → list sessions (or interactive picker)
	if len(args) == 0 && cmd.ArgsLenAtDash() == -1 {
		return lsRun(cmd, args)
	}

	// Parse name and claude args from positional args
	var name string
	var claudeArgs []string

	dashIdx := cmd.ArgsLenAtDash()
	if dashIdx == -1 {
		if len(args) < 1 {
			return lsRun(cmd, args)
		}
		name = args[0]
	} else {
		positional := args[:dashIdx]
		claudeArgs = args[dashIdx:]
		if len(positional) < 1 {
			return fmt.Errorf("session name is required")
		}
		name = positional[0]
	}

	// Check reserved names
	if reservedNames[name] {
		return fmt.Errorf("%q is a reserved command name — choose a different session name", name)
	}

	// Try to find existing session
	sess, err := db.GetSession(name)
	if err == nil {
		return attachFlow(cmd, sess)
	}

	// Session doesn't exist — confirm creation
	fmt.Printf("Session %q not found. Create it? [Y/n]: ", name)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "" && answer != "y" && answer != "yes" {
		return nil
	}

	return createFlow(cmd, name, claudeArgs)
}

// createFlow creates a new session and launches Claude.
func createFlow(cmd *cobra.Command, name string, claudeArgs []string) error {
	// Resolve working directory
	cwd := flagCwd
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
	}

	// Create session record
	now := time.Now()
	sess := &store.Session{
		Name:       name,
		SessionID:  "", // populated after Claude starts
		ClaudePID:  0,
		Cwd:        cwd,
		Status:     "active",
		ClaudeArgs: claudeArgs,
		Tags:       flagTags,
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

	// Build claude args
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

// attachFlow attaches to an existing session (resume or fresh fallback).
func attachFlow(cmd *cobra.Command, sess *store.Session) error {
	name := sess.Name

	// If session is marked active, check if PID is actually alive
	if sess.Status == "active" {
		if session.CheckPID(sess.ClaudePID) {
			if !flagForce {
				return fmt.Errorf("session %q is already active (PID %d); detach it first (or use --force)", name, sess.ClaudePID)
			}
			// Force-detach: send SIGTERM to the existing Claude process
			if proc, err := os.FindProcess(sess.ClaudePID); err == nil {
				if err := proc.Signal(syscall.SIGTERM); err != nil {
					ui.Warn("failed to send SIGTERM to PID %d: %v", sess.ClaudePID, err)
				}
			}
			sess.Status = "detached"
			sess.ClaudePID = 0
			if err := db.UpdateSession(sess); err != nil {
				return fmt.Errorf("updating session: %w", err)
			}
			_ = db.AppendHistory(name, "force-detached", "detached by attach --force")
			ui.Success("Force-detached session %q", name)
		} else {
			// PID is dead — transition to detached
			sess.Status = "detached"
			sess.ClaudePID = 0
			if err := db.UpdateSession(sess); err != nil {
				return fmt.Errorf("updating stale session: %w", err)
			}
			_ = db.AppendHistory(name, "reaped", "PID was dead on attach")
		}
	}

	// Resolve claude binary
	claudeBin, err := cfg.GetClaudeBin()
	if err != nil {
		return err
	}

	// Update session to active
	sess.Status = "active"
	sess.AccessedAt = time.Now()
	if err := db.UpdateSession(sess); err != nil {
		return fmt.Errorf("updating session: %w", err)
	}

	var result session.RunResult

	// Try to resume if we have a session ID, otherwise start fresh
	if sess.SessionID != "" {
		_ = db.AppendHistory(name, "attached", fmt.Sprintf("resuming session %s", sess.SessionID))

		execArgs := []string{"--resume", sess.SessionID}
		execArgs = append(execArgs, sess.ClaudeArgs...)

		ui.Launch("Attaching to session %q", name)

		result = session.RunClaudeResume(claudeBin, execArgs, sess.Cwd)

		// If resume failed, fall back to starting a fresh session
		if result.ResumeFailure {
			ui.Warn("Could not resume session %q — starting fresh in %s", name, ui.ShortenPath(sess.Cwd))
			_ = db.AppendHistory(name, "resume-failed", fmt.Sprintf("session ID %s not found by Claude", sess.SessionID))

			result = launchFresh(claudeBin, name, sess.Cwd, sess.ClaudeArgs)
		}
	} else {
		ui.Warn("No session ID for %q — starting fresh in %s", name, ui.ShortenPath(sess.Cwd))
		_ = db.AppendHistory(name, "attached", "no session ID, starting fresh")

		result = launchFresh(claudeBin, name, sess.Cwd, sess.ClaudeArgs)
	}

	// Update PID and session ID if we got them
	if result.PID > 0 {
		sess.ClaudePID = result.PID
		if result.SessionID != "" {
			sess.SessionID = result.SessionID
		}
		_ = db.UpdateSession(sess)
	}

	// Mark session as detached now that Claude has exited
	sess.Status = "detached"
	sess.ClaudePID = 0
	sess.AccessedAt = time.Now()
	_ = db.UpdateSession(sess)
	_ = db.AppendHistory(name, "detached", "claude process exited")

	return result.Err
}

// launchFresh starts a new Claude session in the given directory with all original args.
func launchFresh(claudeBin, name, cwd string, claudeArgs []string) session.RunResult {
	execArgs := []string{"--name", name}
	execArgs = append(execArgs, claudeArgs...)
	return session.RunClaude(claudeBin, execArgs, cwd)
}
