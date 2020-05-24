package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	goversion "go.hein.dev/go-version"
)

var (
	version    = "dev"
	commit     = "none"
	date       = "unknown"
	shortened  = false
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version of the client CLI",
		Run: func(cmd *cobra.Command, args []string) {
			var response string
			versionOutput := goversion.New(version, commit, date)

			if shortened {
				response = versionOutput.ToShortened()
			} else {
				response = versionOutput.ToJSON()
			}
			fmt.Printf("%+v", response)
			return

		},
	}
)

func init() {
	versionCmd.Flags().BoolVarP(&shortened, "short", "s", false, "Use shortened output for version information.")
	rootCmd.AddCommand(versionCmd)
}
