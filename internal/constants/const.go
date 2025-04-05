package constants

const (
	// default frame width
	DefaultWidth = 1280

	// default frame height
	DefaultHeight = 720

	// default frame rate
	DefaultFrameRate = 1

	//  maximum data/frame
	MaxPayloadPerFrame = ((1280 * 720) / 8) - 32 //formula => ((width * height) / 8) - headerSize

	// encoding identifier
	MagicString = "YTDSv3" // 6 byte magic string

	// frame header size
	HeaderSize = 32
)
