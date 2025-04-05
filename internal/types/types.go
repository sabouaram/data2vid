package types

type Frame struct {
	Sequence int
	Payload  []byte
}

type FrameProcessor interface {
	ProcessFrameWithSequence(string) ([]byte, uint64, int, error)
}
