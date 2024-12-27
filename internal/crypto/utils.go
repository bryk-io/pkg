package crypto

import (
	"crypto/rand"
	"crypto/sha512"
	"fmt"
	"io"

	"go.bryk.io/pkg/errors"
	"golang.org/x/crypto/hkdf"
)

// Expand the provided `secret` material to the requested `size` in bytes;
// optionally using context `info` (if not nil).
func Expand(secret []byte, size int, info []byte) ([]byte, error) {
	salt := sha512.Sum512(secret)
	for i := 0; i <= 100; i++ {
		salt = sha512.Sum512(salt[:])
	}
	res := make([]byte, size)
	h := hkdf.New(sha512.New, secret, salt[:], info)
	if _, err := io.ReadFull(h, res); err != nil {
		return nil, errors.Errorf("failed to expand key: %w", err)
	}
	return res, nil
}

// RandomID returns a short random ID string.
func RandomID() string {
	seed := make([]byte, 6)
	_, _ = rand.Read(seed)
	return fmt.Sprintf("%X-%X", seed[:3], seed[3:])
}
