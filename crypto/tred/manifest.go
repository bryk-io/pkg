package tred

import (
	"encoding/binary"
)

type manifestBlock []byte

// Length in bytes for the output manifestBlock.
// 	version | cipher | length | checksum
const manifestSize = 38

// Retrieve the manifest section from a byte array.
// 	version (1) | cipher (1) | length (4) | checksum (32)
func manifest(b []byte) manifestBlock {
	return b[:manifestSize]
}

// Version return package's version byte.
func (m manifestBlock) Version() byte {
	return m[0]
}

// Cipher return package's used AEAD cipher.
func (m manifestBlock) Cipher() byte {
	return m[1]
}

// Len return manifest's packets count.
func (m manifestBlock) Len() int {
	return int(binary.LittleEndian.Uint16(m[2:6])) + 1
}

// Checksum return manifest's checksum value.
func (m manifestBlock) Checksum() []byte {
	return m[6:]
}

// SetVersion adjust manifest's version byte.
func (m manifestBlock) SetVersion(version byte) {
	m[0] = version
}

// SetCipher adjust manifest's AEAD cipher byte.
func (m manifestBlock) SetCipher(suite byte) {
	m[1] = suite
}

// SetLen adjust manifest's packets count.
func (m manifestBlock) SetLen(length int) {
	binary.LittleEndian.PutUint16(m[2:6], uint16(length-1))
}

// SetChecksum adjust manifest's checksum value.
func (m manifestBlock) SetChecksum(val []byte) {
	copy(m[6:], val[:])
}
