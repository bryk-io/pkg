package ulid

import (
	"time"

	"go.bryk.io/pkg/errors"
)

// maxTime is the maximum Unix time in milliseconds that can be
// represented in a ULID.
var maxTime = ULID{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}.Timestamp()

// Returns `t` time as Unix milliseconds. Because of the way ULID
// stores time, times from the year 10889 produces undefined results.
func fromTime(t time.Time) uint64 {
	return uint64(t.Unix())*1000 + uint64(t.Nanosecond()/int(time.Millisecond))
}

// Converts Unix milliseconds in the format returned by the `fromTime`
// function to a time.Time.
func toTime(ms uint64) time.Time {
	s := int64(ms / 1e3)
	ns := int64((ms % 1e3) * 1e6)
	return time.Unix(s, ns)
}

// nolint: gocyclo
func parse(v []byte, strict bool, id *ULID) error {
	// Check if a base32 encoded ULID is the right length.
	if len(v) != encodedSize {
		return errors.New(ErrDataSize)
	}

	// Check if all the characters in a base32 encoded ULID are part of the
	// expected base32 character set.
	if strict &&
		(dec[v[0]] == 0xFF ||
			dec[v[1]] == 0xFF ||
			dec[v[2]] == 0xFF ||
			dec[v[3]] == 0xFF ||
			dec[v[4]] == 0xFF ||
			dec[v[5]] == 0xFF ||
			dec[v[6]] == 0xFF ||
			dec[v[7]] == 0xFF ||
			dec[v[8]] == 0xFF ||
			dec[v[9]] == 0xFF ||
			dec[v[10]] == 0xFF ||
			dec[v[11]] == 0xFF ||
			dec[v[12]] == 0xFF ||
			dec[v[13]] == 0xFF ||
			dec[v[14]] == 0xFF ||
			dec[v[15]] == 0xFF ||
			dec[v[16]] == 0xFF ||
			dec[v[17]] == 0xFF ||
			dec[v[18]] == 0xFF ||
			dec[v[19]] == 0xFF ||
			dec[v[20]] == 0xFF ||
			dec[v[21]] == 0xFF ||
			dec[v[22]] == 0xFF ||
			dec[v[23]] == 0xFF ||
			dec[v[24]] == 0xFF ||
			dec[v[25]] == 0xFF) {
		return errors.New(ErrInvalidCharacters)
	}

	// Check if the first character in a base32 encoded ULID will overflow. This
	// happens because the base32 representation encodes 130 bits, while the
	// ULID is only 128 bits.
	if v[0] > '7' {
		return errors.New(ErrOverflow)
	}

	// Use an optimized unrolled loop (from https://github.com/RobThree/NUlid)
	// to decode a base32 ULID.
	// 6 bytes timestamp (48 bits)
	(*id)[0] = (dec[v[0]] << 5) | dec[v[1]]
	(*id)[1] = (dec[v[2]] << 3) | (dec[v[3]] >> 2)
	(*id)[2] = (dec[v[3]] << 6) | (dec[v[4]] << 1) | (dec[v[5]] >> 4)
	(*id)[3] = (dec[v[5]] << 4) | (dec[v[6]] >> 1)
	(*id)[4] = (dec[v[6]] << 7) | (dec[v[7]] << 2) | (dec[v[8]] >> 3)
	(*id)[5] = (dec[v[8]] << 5) | dec[v[9]]

	// 10 bytes of entropy (80 bits)
	(*id)[6] = (dec[v[10]] << 3) | (dec[v[11]] >> 2)
	(*id)[7] = (dec[v[11]] << 6) | (dec[v[12]] << 1) | (dec[v[13]] >> 4)
	(*id)[8] = (dec[v[13]] << 4) | (dec[v[14]] >> 1)
	(*id)[9] = (dec[v[14]] << 7) | (dec[v[15]] << 2) | (dec[v[16]] >> 3)
	(*id)[10] = (dec[v[16]] << 5) | dec[v[17]]
	(*id)[11] = (dec[v[18]] << 3) | dec[v[19]]>>2
	(*id)[12] = (dec[v[19]] << 6) | (dec[v[20]] << 1) | (dec[v[21]] >> 4)
	(*id)[13] = (dec[v[21]] << 4) | (dec[v[22]] >> 1)
	(*id)[14] = (dec[v[22]] << 7) | (dec[v[23]] << 2) | (dec[v[24]] >> 3)
	(*id)[15] = (dec[v[24]] << 5) | dec[v[25]]
	return nil
}

// Encoding is the base 32 encoding alphabet used in ULID strings.
const encoding = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// Byte to index table for O(1) lookups when unmarshaling.
// We use 0xFF as sentinel value for invalid indexes.
var dec = [...]byte{
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x01,
	0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E,
	0x0F, 0x10, 0x11, 0xFF, 0x12, 0x13, 0xFF, 0x14, 0x15, 0xFF,
	0x16, 0x17, 0x18, 0x19, 0x1A, 0xFF, 0x1B, 0x1C, 0x1D, 0x1E,
	0x1F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x0A, 0x0B, 0x0C,
	0x0D, 0x0E, 0x0F, 0x10, 0x11, 0xFF, 0x12, 0x13, 0xFF, 0x14,
	0x15, 0xFF, 0x16, 0x17, 0x18, 0x19, 0x1A, 0xFF, 0x1B, 0x1C,
	0x1D, 0x1E, 0x1F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
}

// length of a text encoded ULID.
const encodedSize = 26
