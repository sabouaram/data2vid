package encoder

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/sabouaram/data2vid/internal/constants"
	"github.com/sabouaram/data2vid/internal/frame"
	"github.com/sabouaram/data2vid/internal/video"
	"github.com/spf13/viper"
)

// VideoEncoder handles encoding and decoding of files to/from video
type VideoEncoder struct {
	frameWidth  int
	frameHeight int
	frameRate   int
	tempDir     string
	mutex       sync.Mutex
}

// NewVideoEncoder creates a new encoder with default constant settings
func NewVideoEncoder(cfg *viper.Viper) *VideoEncoder {
	encoder := &VideoEncoder{
		frameWidth:  constants.DefaultWidth,
		frameHeight: constants.DefaultHeight,
		frameRate:   constants.DefaultFrameRate,
	}

	if cfg != nil {
		if cfg.GetInt("Width") != 0 {
			encoder.frameWidth = cfg.GetInt("Width")
		}

		if cfg.GetInt("Height") != 0 {
			encoder.frameHeight = cfg.GetInt("Height")
		}
	}

	constants.MaxPayloadPerFrame = ((encoder.frameHeight * encoder.frameWidth) / 8) - constants.HeaderSize

	return encoder
}

// EncodeFile encodes any file type into an MP4 video file
func (e *VideoEncoder) EncodeFile(inputPath, outputVideo string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	var (
		tempDir    string
		err        error
		inputFile  *os.File
		fileInfo   os.FileInfo
		framePaths []string
	)

	// temp dir
	if tempDir, err = os.MkdirTemp("", "ytdata"); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	e.tempDir = tempDir

	defer os.RemoveAll(tempDir)

	if inputFile, err = os.Open(inputPath); err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}

	defer inputFile.Close()

	if fileInfo, err = inputFile.Stat(); err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Create frame images (PNG) from the file data chunks  ->
	// Frame Format Design:
	//
	// 1. Header Structure (32 bytes total):
	//
	// +-------------+-------------+-------------+-------------+-------------+------------+
	// | Magic String|  Total Size | Sequence #  | Chunk Size  |    Data     |  Header    |
	// | (6 bytes)   |  (8 bytes)  | (4 bytes)   | (4 bytes)   |  Checksum   |  Checksum  |
	// |             |             |             |             |  (8 bytes)  |  (2 bytes) |
	// +-------------+-------------+-------------+-------------+-------------+-------------+
	// | 0     5     | 6        13 | 14      17  | 18      21  | 22       29 | 30      31  |
	// +-------------+-------------+-------------+-------------+-------------+-------------+
	//
	// 2. Data Encoding:
	//
	// +---------------------------+
	// |        Frame Header       |  32 bytes
	// +---------------------------+
	// |                           |
	// |         Payload           |  Variable length (up to maxPayloadPerFrame)
	// |                           |
	// +---------------------------+
	//
	// 3. Pixel Mapping:
	//
	// Each byte of combined header+data is split into 8 bits
	// Each bit is represented as one pixel in the image:
	//   - Bit 0 = White pixel
	//   - Bit 1 = Black pixel
	//
	// +---+---+---+---+---+---+---+---+---+---+---+
	// | 0 | 1 | 0 | 1 | 0 | 0 | 1 | 1 | 0 | 1 | ...  Bits from header and data
	// +---+---+---+---+---+---+---+---+---+---+---+
	//   ↓   ↓   ↓   ↓   ↓   ↓   ↓   ↓   ↓   ↓
	// +---+---+---+---+---+---+---+---+---+---+---+
	// | W | B | W | B | W | W | B | B | W | B | ...  Pixels in image (W=White, B=Black)
	// +---+---+---+---+---+---+---+---+---+---+---+
	//
	// Pixels are filled row by row, left to right, top to bottom:
	//
	// +---+---+---+---+---+
	// | 0 | 1 | 2 | 3 | 4 |
	// +---+---+---+---+---+
	// | 5 | 6 | 7 | 8 | 9 |
	// +---+---+---+---+---+
	// |10 |11 |12 |13 |14 |
	// +---+---+---+---+---+
	//
	// For a 1280x720 frame, this allows storing approximately 115,168 bytes of data
	// (1280*720/8 bits - 32 bytes for the header)
	if framePaths, err = e.createFrames(inputFile, fileInfo.Size()); err != nil {
		return fmt.Errorf("failed to create frames: %w", err)
	}

	// video from frames : using ffmpeg pkg: libx264 codec with yuv420p
	if err = e.createVideo(framePaths, outputVideo); err != nil {
		return fmt.Errorf("failed to create video: %w", err)
	}

	return nil
}

// DecodeFile extracts and reconstructs the original file from video frames
func (e *VideoEncoder) DecodeFile(videoPath, outputPath string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	return video.DecodeFile(e, videoPath, outputPath)
}

// createFrames generates PNG frames from file data
func (e *VideoEncoder) createFrames(input io.Reader, fileSize int64) ([]string, error) {
	return frame.CreateFrames(e.tempDir, input, fileSize, e.frameWidth, e.frameHeight)
}

// createVideo combines frames into a video file
func (e *VideoEncoder) createVideo(framePaths []string, outputVideo string) error {
	return video.CreateVideo(e.tempDir, framePaths, outputVideo)
}

// ProcessFrameWithSequence extracts data from a frame and returns the payload  total size and sequence number
func (e *VideoEncoder) ProcessFrameWithSequence(framePath string) ([]byte, uint64, int, error) {
	return frame.ProcessFrameWithSequence(framePath, e.frameWidth, e.frameHeight)
}
