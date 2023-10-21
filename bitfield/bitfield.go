package bitfield

type Bitfield []byte

// HasPiece returns true if the bitfield contains the piece at index
func (b Bitfield) HasPiece(index int) bool {
	byteIndex := index / 8
	offset := index % 8
	if byteIndex >= len(b) {
		return false
	}
	// the 0-index bit is the first rightmost bit, etc.
	return b[byteIndex]>>(7-offset)&1 != 0
}

// SetPiece sets the bitfield at index to 1
func (b Bitfield) SetPiece(index int) {
	byteIndex := index / 8
	offset := index % 8
	if byteIndex >= len(b) {
		return
	}
	b[byteIndex] |= 1 << (7 - offset)
}
