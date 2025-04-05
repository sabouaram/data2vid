package video

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/sabouaram/data2vid/internal/types"
	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

var fileMutex sync.Mutex

// CreateVideo combines frames into an MP4 video file
func CreateVideo(tempDir string, framePaths []string, outputVideo string) error {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	var (
		listFile, absPath, absLast, absOutput string
		f                                     *os.File
		err                                   error
	)

	//  input file list for ffmpeg
	listFile = filepath.Join(tempDir, "filelist.txt")

	if f, err = os.Create(listFile); err != nil {
		return fmt.Errorf("failed to create file list: %w", err)
	}

	defer f.Close()

	// frames write
	for _, path := range framePaths {
		if absPath, err = filepath.Abs(path); err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}

		fmt.Fprintf(f, "file '%s'\n", absPath)
		fmt.Fprintf(f, "duration 1\n") // Each frame lasts 1 second
	}

	// ffmpeg quirk
	if len(framePaths) > 0 {
		if absLast, err = filepath.Abs(framePaths[len(framePaths)-1]); err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}

		fmt.Fprintf(f, "file '%s'\n", absLast)
	}

	if absOutput, err = filepath.Abs(outputVideo); err != nil {
		return fmt.Errorf("failed to get absolute output path: %w", err)
	}

	//  ffmpeg command using ffmpeg-go - NB: only MP4
	kwArgs := ffmpeg_go.KwArgs{
		"y": "",
	}
	kwArgs["c:v"] = "libx264"
	kwArgs["preset"] = "ultrafast"
	kwArgs["pix_fmt"] = "yuv420p"

	stream := ffmpeg_go.Input(listFile, ffmpeg_go.KwArgs{
		"f":    "concat",
		"safe": "0",
	}).Output(absOutput, kwArgs)

	// lossless param
	stream = stream.GlobalArgs("-qp", "0").
		GlobalArgs("-x264-params", "qp=0")

	if err = stream.Run(); err != nil {
		return fmt.Errorf("ffmpeg error: %w", err)
	}

	return nil
}

// DecodeFile extracts and reconstructs the original file from MP4 video frames
func DecodeFile(encoder types.FrameProcessor, videoPath, outputPath string) error {
	var (
		tempDir, framePath, tempFile                   string
		err                                            error
		framePattern                                   string
		fileSize, size                                 uint64
		frames                                         []types.Frame
		validFrames, frameCount, totalFrames, sequence int
		seenSequences                                  = make(map[int]bool)
		payload, reconstructed                         []byte
		chunks                                         [][]byte
	)

	if outputPath == "" {
		return errors.New("output path cannot be empty")
	}

	// timestamped temp director //debugging
	if tempDir, err = os.MkdirTemp("", fmt.Sprintf("ytdecode_%d_", time.Now().Unix())); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	defer func() error {
		if err = os.RemoveAll(tempDir); err != nil {
			return fmt.Errorf("failed to cleanup temp dir: %v", err)
		}
		return nil
	}()

	// extract frames
	framePattern = filepath.Join(tempDir, "frame_%04d.png")
	if err = ffmpeg_go.Input(videoPath).
		Output(framePattern, ffmpeg_go.KwArgs{
			"vsync":        "0",
			"vf":           "fps=1",
			"pix_fmt":      "gray",
			"start_number": "0",
		}).
		Run(); err != nil {

		// alternative
		err = ffmpeg_go.Input(videoPath).
			Output(framePattern, ffmpeg_go.KwArgs{
				"vsync": "0",
				"q:v":   "1",
			}).
			Run()

		if err != nil {
			return fmt.Errorf("frame extraction failed: %w", err)
		}
	}

	// extracted count
	for i := 0; ; i++ {
		framePath = filepath.Join(tempDir, fmt.Sprintf("frame_%04d.png", i))

		if _, err = os.Stat(framePath); os.IsNotExist(err) {
			break
		}

		frameCount++
	}

	if frameCount == 0 {
		return fmt.Errorf("no frames could be extracted from the video")
	}

	// process & storing frames

	for frameIndex := 0; frameIndex < frameCount; frameIndex++ {
		framePath = filepath.Join(tempDir, fmt.Sprintf("frame_%04d.png", frameIndex))

		if _, err := os.Stat(framePath); os.IsNotExist(err) {
			continue
		}

		totalFrames++

		if payload, size, sequence, err = encoder.ProcessFrameWithSequence(framePath); err != nil {
			continue
		}

		// duplicated skip
		if seenSequences[sequence] {
			continue
		}
		seenSequences[sequence] = true

		if fileSize == 0 {
			fileSize = size

		}

		frames = append(frames, types.Frame{
			Sequence: sequence,
			Payload:  payload,
		})

		validFrames++
	}

	if validFrames == 0 {
		return fmt.Errorf("no valid frames found (attempted %d)", totalFrames)
	}

	// sort frames by seq num
	sort.Slice(frames, func(i, j int) bool {
		return frames[i].Sequence < frames[j].Sequence
	})

	// extract payloads
	for _, frame := range frames {
		chunks = append(chunks, frame.Payload)
	}

	// reconstruct original file data
	reconstructed = bytes.Join(chunks, nil)

	if uint64(len(reconstructed)) > fileSize {
		reconstructed = reconstructed[:fileSize]
	} else if uint64(len(reconstructed)) < fileSize {
		return fmt.Errorf("size mismatch: expected %d bytes, got %d", fileSize, len(reconstructed))
	}

	fileMutex.Lock()
	defer fileMutex.Unlock()

	tempFile = outputPath + ".tmp"

	if err = os.WriteFile(tempFile, reconstructed, 0644); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	if err = os.Rename(tempFile, outputPath); err != nil {
		return fmt.Errorf("failed to finalize output: %w", err)
	}

	return nil
}
