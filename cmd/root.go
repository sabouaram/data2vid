package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	rootLogger *zap.Logger
	rootCfg    *viper.Viper
	rootCmd    = &cobra.Command{
		Short: "Encode/decode files to/from  mp4 video format",
	}
)

func Execute(logger *zap.Logger, cfg *viper.Viper) {
	rootLogger = logger
	rootCfg = cfg

	if err := rootCmd.Execute(); err != nil {
		rootLogger.Error("Command execution failed", zap.Error(err))

		os.Exit(1)
	}
}
