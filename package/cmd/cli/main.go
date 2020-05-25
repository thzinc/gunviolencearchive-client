package cli

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	log     *zap.SugaredLogger
	verbose bool
	rootCmd = &cobra.Command{
		Use: "cli",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			config := zap.NewProductionConfig()
			config.Encoding = "console"
			if verbose {
				config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
			} else {
				config.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
			}
			l, _ := config.Build()
			log = l.Sugar()
		},
	}
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
}

// Execute runs the command line interface to the client
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err.Error())
	}
}
