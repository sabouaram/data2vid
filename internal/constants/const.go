package constants

var MaxPayloadPerFrame = ((DefaultHeight * DefaultWidth) / 8) - HeaderSize

const (
	// default frame width
	DefaultWidth = 1280

	// default frame height
	DefaultHeight = 720

	// default frame rate
	DefaultFrameRate = 1

	// encoding identifier
	MagicString = "YTDSv3" // 6 byte magic string

	// frame header size
	HeaderSize = 32
)
