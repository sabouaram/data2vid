package frame

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/sabouaram/data2vid/internal/checksum"
	"github.com/sabouaram/data2vid/internal/constants"
)

var fileMutex sync.Mutex

// CreateFrames generates PNG frames from input file data
func CreateFrames(tempDir string, input io.Reader, fileSize int64, frameWidth, frameHeight int) ([]string, error) {

	var (
		framePaths []string
		chunk      = make([]byte, constants.MaxPayloadPerFrame)
		sequence   = 0
		n          int
		err        error
		framePath  string
	)

	for {

		if n, err = io.ReadFull(input, chunk); err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("read error: %w", err)
		}

		if n == 0 {
			break
		}

		// create frame for this chunk
		framePath = filepath.Join(tempDir, fmt.Sprintf("frame_%04d.png", sequence))

		if err = CreateSingleFrame(chunk[:n], sequence, fileSize, framePath, frameWidth, frameHeight); err != nil {
			return nil, fmt.Errorf("frame creation failed: %w", err)
		}

		framePaths = append(framePaths, framePath)

		sequence++
	}

	return framePaths, nil
}

// CreateSingleFrame creates a single PNG frame from data
func CreateSingleFrame(data []byte, sequence int, totalSize int64, outputPath string, frameWidth, frameHeight int) error {

	var (
		outFile *os.File
		err     error
		bitPos  = 0
		img     *image.Gray
	)

	// header
	// +-------------+-------------+-------------+-------------+-------------+------------+
	// | Magic String|  Total Size | Sequence #  | Chunk Size  |    Data     |  Header    |
	// | (6 bytes)   |  (8 bytes)  | (4 bytes)   | (4 bytes)   |  Checksum   |  Checksum  |
	// |             |             |             |             |  (8 bytes)  |  (2 bytes) |
	// +-------------+-------------+-------------+-------------+-------------+-------------+
	// | 0     5     | 6        13 | 14      17  | 18      21  | 22       29 | 30      31  |
	// +-------------+-------------+-------------+-------------+-------------+-------------+
	header := make([]byte, constants.HeaderSize)
	copy(header[:6], []byte(constants.MagicString))
	binary.BigEndian.PutUint64(header[6:14], uint64(totalSize))     // Total file size
	binary.BigEndian.PutUint32(header[14:18], uint32(sequence))     // Sequence number
	binary.BigEndian.PutUint32(header[18:22], uint32(len(data)))    // Chunk size
	binary.BigEndian.PutUint64(header[22:30], checksum.CRC64(data)) // Data checksum
	copy(header[30:32], checksum.ComputeChecksum(header[:30])[:2])  // Header checksum

	frameData := append(header, data...)

	// gray img
	img = image.NewGray(image.Rect(0, 0, frameWidth, frameHeight))

	// default white
	for y := 0; y < frameHeight; y++ {
		for x := 0; x < frameWidth; x++ {
			img.Set(x, y, color.White)
		}
	}

	// setting pixels
	for _, b := range frameData {
		for bit := 7; bit >= 0; bit-- {
			//  x,y position from bit position
			x := (bitPos % frameWidth)
			y := (bitPos / frameWidth)

			if y >= frameHeight {
				//  entire frame done
				break
			}

			// 1 -> black - 0 -> white
			if (b & (1 << bit)) != 0 {
				img.Set(x, y, color.Black)
			}

			bitPos++
		}
	}

	fileMutex.Lock()
	defer fileMutex.Unlock()

	// Save as PNG
	if outFile, err = os.Create(outputPath); err != nil {
		return fmt.Errorf("create file error: %w", err)
	}

	defer outFile.Close()

	if err = png.Encode(outFile, img); err != nil {
		return fmt.Errorf("png encode error: %w", err)
	}

	return nil
}

// ProcessFrameWithSequence extracts data from a frame and returns the payload - total size and sequence number
func ProcessFrameWithSequence(framePath string, frameWidth, frameHeight int) ([]byte, uint64, int, error) {

	var (
		file          *os.File
		err           error
		img           image.Image
		frameData     bytes.Buffer
		currentByte   byte = 0
		data, payload []byte
		bitCount      = 0
	)

	fileMutex.Lock()
	defer fileMutex.Unlock()

	if file, err = os.Open(framePath); err != nil {
		return nil, 0, 0, fmt.Errorf("failed to open frame: %w", err)
	}

	defer file.Close()

	if img, _, err = image.Decode(file); err != nil {
		return nil, 0, 0, fmt.Errorf("image decode failed: %w", err)
	}

	//  dimensions verif
	if img.Bounds().Dx() != frameWidth || img.Bounds().Dy() != frameHeight {
		return nil, 0, 0, fmt.Errorf("invalid dimensions (%dx%d)", img.Bounds().Dx(), img.Bounds().Dy())
	}

	// extract binary data from black/white pixels
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			// grayscale value
			r, g, b, _ := img.At(x, y).RGBA()

			// convert to grayscale
			gray := (r + g + b) / 3

			// shift current byte and add new bit (0:white --- 1:black)
			currentByte = currentByte << 1
			if gray < 32768 { //  pixel is closer to black than white
				currentByte |= 1
			}

			bitCount++

			// write every 8 bits (1 Byte)
			if bitCount == 8 {
				frameData.WriteByte(currentByte)

				currentByte = 0
				bitCount = 0
			}

			// enough bytes for a header => check for magic string
			if frameData.Len() >= len(constants.MagicString) {
				data = frameData.Bytes()

				if bytes.Contains(data, []byte(constants.MagicString)) {

					// found magic string
					magicPos := bytes.Index(data, []byte(constants.MagicString))

					if magicPos > 0 {
						data = data[magicPos:]
						frameData.Reset()
						frameData.Write(data)
					}

					// complete header => stop collecting
					if frameData.Len() >= constants.HeaderSize {

						data = frameData.Bytes()

						// checksum verif
						if !bytes.Equal(checksum.ComputeChecksum(data[:30])[:2], data[30:32]) {
							continue
						}

						// metadata parsing
						_ = binary.BigEndian.Uint64(data[6:14])

						chunkSize := binary.BigEndian.Uint32(data[18:22])

						// remaining neede bytes
						bytesNeeded := int(chunkSize) + constants.HeaderSize - frameData.Len()
						if bytesNeeded <= 0 {
							// enough data
							break
						}
					}
				}
			}
		}
	}

	data = frameData.Bytes()

	// find magic string header
	magic := []byte(constants.MagicString)
	headerPos := bytes.Index(data, magic)

	if headerPos == -1 {
		return nil, 0, 0, errors.New("magic string not found")
	}

	if headerPos > 0 {
		data = data[headerPos:]
	}

	// header verif size
	if len(data) < constants.HeaderSize {
		return nil, 0, 0, fmt.Errorf("incomplete header (%d bytes)", len(data))
	}

	// header checksum
	if !bytes.Equal(checksum.ComputeChecksum(data[:30])[:2], data[30:32]) {
		return nil, 0, 0, errors.New("header checksum mismatch")
	}

	// parse metadata
	totalSize := binary.BigEndian.Uint64(data[6:14])
	sequence := int(binary.BigEndian.Uint32(data[14:18]))
	chunkSize := binary.BigEndian.Uint32(data[18:22])
	storedChecksum := binary.BigEndian.Uint64(data[22:30])

	// validate chunk size
	if int(chunkSize) > len(data)-constants.HeaderSize {
		return nil, 0, 0, fmt.Errorf("invalid chunk size %d", chunkSize)
	}

	// extract payload
	payload = data[constants.HeaderSize : constants.HeaderSize+int(chunkSize)]

	if checksum.CRC64(payload) != storedChecksum {
		return nil, 0, 0, errors.New("payload checksum mismatch")
	}

	return payload, totalSize, sequence, nil
}
