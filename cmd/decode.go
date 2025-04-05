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
	rootCmd.AddCommand(DecodeCommand())
}

func DecodeCommand() *cobra.Command {
	var (
		outputFile, absOutput, baseName string
		err                             error
		enc                             *encoder.VideoEncoder
	)

	cmd := &cobra.Command{
		Use:   "decode [MP4 video-file]",
		Short: "Decode a video back to its original file. Be sure to explicitly specify the file extension; otherwise, the output may be incomplete.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {

			videoFile := args[0]

			if _, err = os.Stat(videoFile); err != nil {
				rootLogger.Error("Video file path error",
					zap.String("file", videoFile), zap.Error(err))

				os.Exit(1)
			}

			if strings.ToLower(filepath.Ext(videoFile)) != ".mp4" {
				rootLogger.Error("Video file should be MPEG-4 with .mp4 extension",
					zap.String("file", videoFile))

				os.Exit(1)
			}

			if outputFile == "" {
				baseName = filepath.Base(videoFile)
				outputFile = strings.TrimSuffix(baseName, filepath.Ext(baseName)) + "_decoded" + filepath.Ext(baseName)
			}

			if absOutput, err = filepath.Abs(outputFile); err != nil {
				rootLogger.Error("Failed to get absolute path",
					zap.String("output", outputFile), zap.Error(err))

				os.Exit(1)
			}

			enc = encoder.NewVideoEncoder()

			rootLogger.Info("Starting decoding",
				zap.String("input", videoFile),
				zap.String("output", absOutput))

			spinner.WithLoadingSpinner(39, 100*time.Millisecond, func() {
				if err = enc.DecodeFile(videoFile, absOutput); err != nil {
					rootLogger.Error("Decoding failed", zap.Error(err))

					os.Exit(1)
				}

			})

			rootLogger.Info("Successfully decoded file",
				zap.String("output", absOutput))
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: [videoname]_decoded)")

	return cmd
}
