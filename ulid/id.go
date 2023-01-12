package ulid

import (
	"bytes"
	"crypto/rand"
	"io"
	"time"

	"go.bryk.io/pkg/errors"
)

/*
A ULID is a 16 byte Universally Unique Lexicographically Sortable Identifier

	The components are encoded as 16 octets.
	Each component is encoded with the MSB first (network byte order).
	0                   1                   2                   3
	0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|                      32_bit_uint_time_high                    |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|     16_bit_uint_time_low      |       16_bit_uint_random      |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|                       32_bit_uint_random                      |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	|                       32_bit_uint_random                      |
	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
*/
type ULID [16]byte

// Common error codes.
var (
	// ErrDataSize is returned when parsing or unmarshaling ULIDs with
	// the wrong data size.
	ErrDataSize = "ulid: bad data size when unmarshaling"

	// ErrInvalidCharacters is returned when parsing or unmarshaling
	// ULIDs with invalid base32 encodings.
	ErrInvalidCharacters = "ulid: bad data characters when unmarshaling"

	// ErrBigTime is returned when constructing a ULID with a time that
	// is larger than `MaxTime`.
	ErrBigTime = "ulid: time too big"

	// ErrOverflow is returned when unmarshaling a ULID whose first
	// character is larger than 7, thereby exceeding the valid bit depth
	// of 128.
	ErrOverflow = "ulid: overflow when unmarshaling"
)

// New ULID instance using the current UTC time and `crypto.rand` as
// source of entropy.
func New() (id ULID, err error) {
	ms := fromTime(time.Now().UTC())
	if err = id.setTime(ms); err != nil {
		return id, err
	}
	_, err = io.ReadFull(rand.Reader, id[6:])
	return id, err
}

// Bytes returns bytes slice representation of ULID.
func (id ULID) Bytes() []byte {
	return id[:]
}

// String returns a lexicographically sortable string encoded ULID
// (26 characters, non-standard base 32) e.g. 01AN4Z07BY79KA1307SR9X4MV3.
// Format: `tttttttttteeeeeeeeeeeeeeee` where `t` is time and `e` is
// entropy.
func (id ULID) String() string {
	dst, _ := id.MarshalText()
	return string(dst)
}

// Timestamp returns the Unix time in milliseconds encoded in the ULID.
func (id ULID) Timestamp() uint64 {
	return uint64(id[5]) | uint64(id[4])<<8 |
		uint64(id[3])<<16 | uint64(id[2])<<24 |
		uint64(id[1])<<32 | uint64(id[0])<<40
}

// Time returns the `time.Time` encoded in the ULID.
func (id ULID) Time() time.Time {
	return toTime(id.Timestamp())
}

// Entropy returns the entropy from the ULID.
func (id ULID) Entropy() []byte {
	e := make([]byte, 10)
	copy(e, id[6:])
	return e
}

// Compare returns an integer comparing id and other lexicographically.
// The result will be:
//   - 0 if id==other
//   - -1 if id < other
//   - +1 if id > other.
func (id ULID) Compare(other ULID) int {
	return bytes.Compare(id[:], other[:])
}

// MarshalText implements the encoding.TextMarshaler interface by
// returning the string encoded ULID.
func (id ULID) MarshalText() ([]byte, error) {
	dst := make([]byte, encodedSize)

	// Optimized unrolled loop ahead.
	// From https://github.com/RobThree/NUlid

	// 10 byte timestamp
	dst[0] = encoding[(id[0]&224)>>5]
	dst[1] = encoding[id[0]&31]
	dst[2] = encoding[(id[1]&248)>>3]
	dst[3] = encoding[((id[1]&7)<<2)|((id[2]&192)>>6)]
	dst[4] = encoding[(id[2]&62)>>1]
	dst[5] = encoding[((id[2]&1)<<4)|((id[3]&240)>>4)]
	dst[6] = encoding[((id[3]&15)<<1)|((id[4]&128)>>7)]
	dst[7] = encoding[(id[4]&124)>>2]
	dst[8] = encoding[((id[4]&3)<<3)|((id[5]&224)>>5)]
	dst[9] = encoding[id[5]&31]

	// 16 bytes of entropy
	dst[10] = encoding[(id[6]&248)>>3]
	dst[11] = encoding[((id[6]&7)<<2)|((id[7]&192)>>6)]
	dst[12] = encoding[(id[7]&62)>>1]
	dst[13] = encoding[((id[7]&1)<<4)|((id[8]&240)>>4)]
	dst[14] = encoding[((id[8]&15)<<1)|((id[9]&128)>>7)]
	dst[15] = encoding[(id[9]&124)>>2]
	dst[16] = encoding[((id[9]&3)<<3)|((id[10]&224)>>5)]
	dst[17] = encoding[id[10]&31]
	dst[18] = encoding[(id[11]&248)>>3]
	dst[19] = encoding[((id[11]&7)<<2)|((id[12]&192)>>6)]
	dst[20] = encoding[(id[12]&62)>>1]
	dst[21] = encoding[((id[12]&1)<<4)|((id[13]&240)>>4)]
	dst[22] = encoding[((id[13]&15)<<1)|((id[14]&128)>>7)]
	dst[23] = encoding[(id[14]&124)>>2]
	dst[24] = encoding[((id[14]&3)<<3)|((id[15]&224)>>5)]
	dst[25] = encoding[id[15]&31]

	return dst, nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface by
// parsing the data as string encoded ULID.
//
// ErrDataSize is returned if the len(v) is different from an encoded
// ULID's length. Invalid encodings produce undefined ULIDs.
func (id *ULID) UnmarshalText(v []byte) error {
	return parse(v, false, id)
}

// MarshalBinary implements the encoding.BinaryMarshaler interface by
// returning the ULID as a byte slice.
func (id ULID) MarshalBinary() ([]byte, error) {
	dst := make([]byte, len(id))
	copy(dst, id[:])
	return dst, nil
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface by
// copying the passed data and converting it to a ULID. ErrDataSize is
// returned if the data length is different from ULID length.
func (id *ULID) UnmarshalBinary(data []byte) error {
	if len(data) != len(*id) {
		return errors.New(ErrDataSize)
	}
	copy((*id)[:], data)
	return nil
}

// SetTime sets the time component of the ULID to the given Unix time
// in milliseconds.
func (id *ULID) setTime(ms uint64) error {
	if ms > maxTime {
		return errors.New(ErrBigTime)
	}
	(*id)[0] = byte(ms >> 40)
	(*id)[1] = byte(ms >> 32)
	(*id)[2] = byte(ms >> 24)
	(*id)[3] = byte(ms >> 16)
	(*id)[4] = byte(ms >> 8)
	(*id)[5] = byte(ms)
	return nil
}
