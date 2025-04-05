package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	rootLogger *zap.Logger
	rootCmd    = &cobra.Command{
		Short: "Encode/decode files to/from video format",
	}
)

func Execute(logger *zap.Logger) {
	rootLogger = logger

	if err := rootCmd.Execute(); err != nil {
		rootLogger.Error("Command execution failed", zap.Error(err))

		os.Exit(1)
	}
}
