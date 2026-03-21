package cmd

import (
	"os"

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

func SetVersion(v string) { version = v }

var rootCmd = &cobra.Command{
	Use:           "muxc",
	Short:         "muxc -- tmux-like session manager for Claude Code",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip DB init for help/version/completion
		if cmd.Name() == "help" || cmd.Name() == "version" || cmd.Name() == "completion" {
			return nil
		}
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
			return err
		}
		db, err = store.Open(cfg.DBPath())
		if err != nil {
			return err
		}
		// Reap dead sessions on every invocation
		session.ReapDeadSessions(db)
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if db != nil {
			db.Close()
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default: run ls
		return lsRun(cmd, args)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ui.Die("%v", err)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.muxc/config.yaml)")
}

// sessionNameCompletion provides tab-completion for session names
func sessionNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if db == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names, _ := db.ListSessionNames(toComplete)
	return names, cobra.ShellCompDirectiveNoFileComp
}
