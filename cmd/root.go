package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RandomCodeSpace/muxc/internal/claude"
	"github.com/RandomCodeSpace/muxc/internal/session"
	"github.com/RandomCodeSpace/muxc/internal/ui"
)

var (
	claudeBin string
	tmuxBin   string
	version   string = "dev"
)

// reservedNames are subcommand names that cannot be used as session names.
var reservedNames = map[string]bool{
	"ls": true, "list": true, "l": true,
	"info": true, "kill": true, "completion": true, "version": true,
	"help": true,
}

func SetVersion(v string) { version = v }

var rootCmd = &cobra.Command{
	Use:               "muxc [<session>] [flags] [-- <claude-args>...]",
	Short:             "muxc — Claude session manager with tmux",
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
		tmuxBin, err = resolveBin("tmux", "MUXC_TMUX_BIN")
		if err != nil {
			return fmt.Errorf("tmux not found in PATH; install tmux to use muxc")
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

// resolveBin returns a binary path from the given env var or PATH lookup.
func resolveBin(name, envVar string) (string, error) {
	if bin := os.Getenv(envVar); bin != "" {
		return bin, nil
	}
	return exec.LookPath(name)
}

// sessionNameCompletion provides tab-completion for session names.
func sessionNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	refs, _ := claude.ListSessionRefs(toComplete)
	return refs, cobra.ShellCompDirectiveNoFileComp
}

// unifiedRun handles `muxc <name>` — creates or attaches to a tmux session.
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

	tmuxName := session.TmuxSessionName(parsedName)

	// Path 1: tmux session exists → attach
	if session.TmuxHasSession(tmuxBin, tmuxName) {
		ui.Launch("🔗 Attaching to session %q", parsedName)
		return session.TmuxAttach(tmuxBin, tmuxName)
	}

	// Path 2: session data exists → resume in tmux
	sess, nameMatchCount, err := claude.GetSessionByRef(name)
	if err == nil {
		if nameMatchCount > 1 {
			ui.Info("📋 %d sessions named %q — using most recent. Run muxc ls to see all IDs, use muxc %s:<id> to select.",
				nameMatchCount, sess.Name, sess.Name)
		}
		claudeCmd := []string{claudeBin, "--resume", sess.SessionID}
		claudeCmd = append(claudeCmd, claudeArgs...)
		cwd := sess.Cwd
		if flagCwd != "" {
			cwd = flagCwd
		}
		ui.Launch("🔗 Resuming session %q in tmux", parsedName)
		if err := session.TmuxNewSession(tmuxBin, tmuxName, cwd, claudeCmd); err != nil {
			return fmt.Errorf("creating tmux session: %w", err)
		}
		return session.TmuxAttach(tmuxBin, tmuxName)
	}

	// Bad ID prefix — don't offer to create
	if nameMatchCount > 0 {
		return err
	}

	// Path 3: no session → create new
	fmt.Printf("Session %q not found. Create it? [Y/n]: ", parsedName)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "" && answer != "y" && answer != "yes" {
		return nil
	}

	cwd := flagCwd
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
	}

	claudeCmd := []string{claudeBin, "--name", parsedName}
	claudeCmd = append(claudeCmd, claudeArgs...)

	ui.Launch("🚀 Creating session %q in tmux", parsedName)
	if err := session.TmuxNewSession(tmuxBin, tmuxName, cwd, claudeCmd); err != nil {
		return fmt.Errorf("creating tmux session: %w", err)
	}
	return session.TmuxAttach(tmuxBin, tmuxName)
}
