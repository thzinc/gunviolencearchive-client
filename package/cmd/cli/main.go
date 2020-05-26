package cli

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	log     *zap.SugaredLogger
	verbose bool
	// RootCmd is the root command of the command line interface
	RootCmd = &cobra.Command{
		Use:   "gva",
		Short: "Unofficial command line interface to the Gun Violence Archive",
		Long:  "This program is unaffiliated with the Gun Violence Archive",
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
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
}

// Execute runs the command line interface to the client
func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		panic(err)
	}
}
