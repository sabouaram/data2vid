package checksum

func ComputeChecksum(data []byte) []byte {
	sum := make([]byte, 4)

	for i, b := range data {
		sum[i%4] ^= b
	}

	return sum
}

func CRC64(data []byte) uint64 {

	var crc uint64 = 0xFFFFFFFFFFFFFFFF

	for _, b := range data {
		crc ^= uint64(b) << 56

		for i := 0; i < 8; i++ {
			if crc&(1<<63) != 0 {
				crc = (crc << 1) ^ 0x42F0E1EBA9EA3693
			} else {
				crc <<= 1
			}
		}
	}

	return crc ^ 0xFFFFFFFFFFFFFFFF
}
