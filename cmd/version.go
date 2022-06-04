package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var BuildVersion = "Unknown"
var CommitID string

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Shows the version information",
	Run: func(cmd *cobra.Command, args []string) {
		//	Show the version number
		fmt.Printf("\nDashboard service version %s", BuildVersion)

		//	Show the commitID if available:
		if CommitID != "" {
			fmt.Printf(" (%s)", CommitID[:7])
		}

		//	Trailing space and newline
		fmt.Println(" ")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
