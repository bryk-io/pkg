package did

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/mr-tron/base58"
)

// Generate a SHA256 digest value from the provided data.
func getHash(data []byte) []byte {
	h := sha256.New()
	if _, err := h.Write(data); err != nil {
		return nil
	}
	return h.Sum(nil)
}

// https://datatracker.ietf.org/doc/html/draft-multiformats-multibase-03
func multibaseEncode(data []byte) string {
	return "z" + base58.Encode(data)
}

// https://datatracker.ietf.org/doc/html/draft-multiformats-multibase-03
func multibaseDecode(src string) ([]byte, error) {
	base := src[:1]
	data := src[1:]
	// https://datatracker.ietf.org/doc/html/draft-multiformats-multibase-03#appendix-D.1
	switch base {
	// base58btc
	case "z":
		return base58.Decode(data)
	// base16
	case "f":
		return hex.DecodeString(data)
	// base64 (no padding)
	case "m":
		return base64.RawStdEncoding.DecodeString(data)
	// base64pad (with padding - MIME encoding)
	case "M":
		return base64.StdEncoding.DecodeString(data)
	// base64url (no padding)
	case "u":
		return base64.RawURLEncoding.DecodeString(data)
	// base64urlpad (with padding)
	case "U":
		return base64.URLEncoding.DecodeString(data)
	default:
		return nil, fmt.Errorf("unsupported base identifier: %s", base)
	}
}
