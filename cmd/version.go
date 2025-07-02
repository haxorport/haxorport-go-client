package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the application version
const Version = "1.0.0"

// versionCmd is the command to display version
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version",
	Long:  `Display Haxorport Client version.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Haxorport Client v%s\n", Version)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
