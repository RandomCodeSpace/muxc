package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RandomCodeSpace/muxc/internal/claude"
	"github.com/RandomCodeSpace/muxc/internal/session"
	"github.com/RandomCodeSpace/muxc/internal/ui"
)

var (
	claudeBin string
	version   string = "dev"
)

// reservedNames are subcommand names that cannot be used as session names.
var reservedNames = map[string]bool{
	"ls": true, "list": true, "l": true,
	"info": true, "completion": true, "version": true,
	"help": true,
}

func SetVersion(v string) { version = v }

var rootCmd = &cobra.Command{
	Use:               "muxc [<session>] [flags] [-- <claude-args>...]",
	Short:             "muxc — Claude session viewer and launcher",
	SilenceUsage:      true,
	SilenceErrors:     true,
	Args:              cobra.ArbitraryArgs,
	ValidArgsFunction: sessionNameCompletion,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "help" || cmd.Name() == "version" || cmd.Name() == "completion" {
			return nil
		}
		var err error
		claudeBin, err = claude.GetClaudeBin()
		if err != nil {
			return err
		}
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
	rootCmd.Flags().StringVar(&flagCwd, "cwd", "", "working directory for new session (default: current dir)")
	rootCmd.Flags().StringP("status", "s", "", "filter by status (active/detached)")
}

var flagCwd string

// sessionNameCompletion provides tab-completion for session names.
func sessionNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	refs, _ := claude.ListSessionRefs(toComplete)
	return refs, cobra.ShellCompDirectiveNoFileComp
}

// unifiedRun handles `muxc <name>` — resumes if session exists, creates if not.
func unifiedRun(cmd *cobra.Command, args []string) error {
	if len(args) == 0 && cmd.ArgsLenAtDash() == -1 {
		return lsRun(cmd, args)
	}

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

	parsedName, _ := claude.ParseSessionRef(name)
	if reservedNames[parsedName] {
		return fmt.Errorf("%q is a reserved command name — choose a different session name", parsedName)
	}

	// Try to find existing session
	sess, nameMatchCount, err := claude.GetSessionByRef(name)
	if err == nil {
		if nameMatchCount > 1 {
			ui.Info("📋 %d sessions named %q — using most recent. Run muxc ls to see all IDs, use muxc %s:<id> to select.",
				nameMatchCount, sess.Name, sess.Name)
		}
		return attachFlow(sess, claudeArgs)
	}

	// Name exists but ID prefix didn't match — don't offer to create
	if nameMatchCount > 0 {
		return err
	}

	// Session doesn't exist — confirm creation
	fmt.Printf("Session %q not found. Create it? [Y/n]: ", parsedName)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "" && answer != "y" && answer != "yes" {
		return nil
	}

	return createFlow(parsedName, claudeArgs)
}

// createFlow launches a new Claude session.
func createFlow(name string, claudeArgs []string) error {
	cwd := flagCwd
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
	}

	execArgs := []string{"--name", name}
	execArgs = append(execArgs, claudeArgs...)

	ui.Launch("🚀 Creating session %q", name)
	result := session.RunClaude(claudeBin, execArgs, cwd)

	if result.SessionID == "" {
		ui.Warn("⚠️  Could not capture Claude session ID — resume may not work for session %q", name)
	}

	return result.Err
}

// attachFlow resumes an existing session via claude --resume.
func attachFlow(sess *claude.Session, claudeArgs []string) error {
	// If session is active, tell user and exit
	if sess.Status == "active" && claude.CheckPID(sess.PID) {
		return fmt.Errorf("session %q is already active (PID %d)", sess.Name, sess.PID)
	}

	if sess.SessionID == "" {
		ui.Warn("⚠️  No session ID for %q — starting fresh in %s", sess.Name, ui.ShortenPath(sess.Cwd))
		return createFlow(sess.Name, claudeArgs)
	}

	execArgs := []string{"--resume", sess.SessionID}
	execArgs = append(execArgs, claudeArgs...)

	ui.Launch("🔗 Resuming session %q", sess.Name)
	result := session.RunClaudeResume(claudeBin, execArgs, sess.Cwd)

	// If resume failed, fall back to fresh session
	if result.ResumeFailure {
		ui.Warn("⚠️  Could not resume session %q — starting fresh in %s", sess.Name, ui.ShortenPath(sess.Cwd))
		return createFlow(sess.Name, claudeArgs)
	}

	return result.Err
}
