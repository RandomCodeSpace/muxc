package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print muxc version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("muxc %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
