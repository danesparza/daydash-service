package cmd

import (
	"fmt"

	"github.com/danesparza/daydash-service/version"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Shows the version information",
	Run: func(cmd *cobra.Command, args []string) {
		//	Show the version number
		fmt.Printf("\nDashboard service version %s", version.String())

		//	Show the commitID if available:
		if version.CommitID != "" {
			fmt.Printf(" (%s)", version.CommitID[:7])
		}

		//	Trailing space and newline
		fmt.Println(" ")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
