package tred

import (
	"encoding/binary"
)

type headerBlock []byte

// Retrieve the header section from a byte array.
// 	version (1) | cipher (1) | payload length (2) | seq (4) | nonce (8)
func header(b []byte) headerBlock {
	return b[:headerSize]
}

// Version return package's version byte.
func (h headerBlock) Version() byte {
	return h[0]
}

// Cipher return package's used AEAD cipher.
func (h headerBlock) Cipher() byte {
	return h[1]
}

// Len return package's payload length.
func (h headerBlock) Len() int {
	return int(binary.LittleEndian.Uint16(h[2:])) + 1
}

// SequenceNumber return package's seq number.
func (h headerBlock) SequenceNumber() uint32 {
	return binary.LittleEndian.Uint32(h[4:])
}

// SetVersion adjust the package's version byte.
func (h headerBlock) SetVersion(version byte) {
	h[0] = version
}

// SetCipher adjust the package's AEAD cipher used.
func (h headerBlock) SetCipher(suite byte) {
	h[1] = suite
}

// SetLen adjust the package's payload length.
func (h headerBlock) SetLen(length int) {
	binary.LittleEndian.PutUint16(h[2:], uint16(length-1))
}

// SetSequenceNumber adjust the package's seq number.
func (h headerBlock) SetSequenceNumber(num uint32) {
	binary.LittleEndian.PutUint32(h[4:], num)
}

// SetNonce adjust the package's nonce value.
func (h headerBlock) SetNonce(nonce [8]byte) {
	copy(h[8:], nonce[:])
}
