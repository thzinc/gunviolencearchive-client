package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	log     *zap.SugaredLogger
	debug   bool
	rootCmd = &cobra.Command{
		Use: "cli",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("wow")
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			config := zap.NewProductionConfig()
			config.Encoding = "console"
			if debug {
				config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
			}
			l, _ := config.Build()
			log = l.Sugar()
		},
	}
)

// Execute runs the command line interface to the client
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err.Error())
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debug, "verbose", "v", false, "Enable verbose logging")
}
