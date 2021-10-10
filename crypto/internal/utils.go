package internal

import (
	"crypto/rand"
	"crypto/sha512"
	"fmt"
	"io"

	"go.bryk.io/miracl/core"
	"golang.org/x/crypto/hkdf"
)

// GetRNG return a pre-seeded random number generator instance.
func GetRNG(seedSize int, base []byte) (*core.RAND, error) {
	if base == nil {
		base = make([]byte, seedSize)
		_, err := rand.Read(base)
		if err != nil {
			return nil, err
		}
	}
	rng := core.NewRAND()
	rng.Clean()
	rng.Seed(seedSize, base)
	return rng, nil
}

// Expand securely the provided secret material.
func Expand(secret []byte, size int, info []byte) ([]byte, error) {
	salt := sha512.Sum512(secret)
	for i := 0; i <= 100; i++ {
		salt = sha512.Sum512(salt[:])
	}
	res := make([]byte, size)
	h := hkdf.New(sha512.New, secret, salt[:], info)
	if _, err := io.ReadFull(h, res); err != nil {
		return nil, fmt.Errorf("failed to expand key: %w", err)
	}
	return res, nil
}
