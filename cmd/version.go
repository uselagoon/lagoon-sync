package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of lagoon-sync",
	Long:  "Print the version number of lagoon-sync",
	Run: func(v *cobra.Command, args []string) {
		fmt.Println("Lagoon Sync Version:", rootCmd.Version)
	},
}
