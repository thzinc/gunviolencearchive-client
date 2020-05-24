package cli

import (
	"github.com/spf13/cobra"
	"github.com/syncromatics/go-kit/log"
)

var rootCmd = &cobra.Command{
	Use: "cli",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("Wow")
	},
}

// Execute runs the command line interface to the client
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err.Error())
	}
}
