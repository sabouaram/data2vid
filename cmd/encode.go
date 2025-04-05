package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sabouaram/data2vid/cmd/spinner"
	"github.com/sabouaram/data2vid/internal/encoder"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func init() {
	rootCmd.AddCommand(EncodeCommand())
}

func EncodeCommand() *cobra.Command {
	var (
		outputVideo, absOutput string
		err                    error
		enc                    *encoder.VideoEncoder
	)

	cmd := &cobra.Command{
		Use:   "encode [input-file]",
		Short: "Encode a file into video format",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			inputFile := args[0]

			if _, err = os.Stat(inputFile); err != nil {
				rootLogger.Error("Input file path error",
					zap.String("file", inputFile), zap.Error(err))

				os.Exit(1)
			}

			if outputVideo == "" {
				baseName := filepath.Base(inputFile)
				outputVideo = strings.TrimSuffix(baseName, filepath.Ext(baseName)) + ".mp4"

			} else {
				if strings.ToLower(filepath.Ext(outputVideo)) != ".mp4" {
					outputVideo = strings.TrimSuffix(outputVideo, strings.ToLower(filepath.Ext(outputVideo))) + ".mp4"

					rootLogger.Info("Forcing MP4 output format",
						zap.String("output", outputVideo))
				}
			}

			if absOutput, err = filepath.Abs(outputVideo); err != nil {
				rootLogger.Error("Failed to get absolute path",
					zap.String("output", outputVideo), zap.Error(err))

				os.Exit(1)
			}

			enc = encoder.NewVideoEncoder(rootCfg)

			rootLogger.Info("Starting encoding",
				zap.String("input", inputFile),
				zap.String("output", absOutput))

			spinner.WithLoadingSpinner(39, 100*time.Millisecond, func() {
				if err = enc.EncodeFile(inputFile, absOutput); err != nil {
					rootLogger.Error("Encoding failed", zap.Error(err))

					os.Exit(1)
				}
			})

			rootLogger.Info("Successfully encoded file",
				zap.String("output", absOutput))
		},
	}

	cmd.Flags().StringVarP(&outputVideo, "output", "o", "", "Output video file path (default: [inputname].mp4)")

	return cmd
}
