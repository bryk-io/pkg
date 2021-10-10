package did

import (
	"crypto/rand"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/sha3"
)

// IDStringVerifier allows the user to provide a custom verifier method for the specific
// id string segment. Successful verifications must return a nil error.
type IDStringVerifier func(string) error

type idStringMode int

const (
	// ModeUUID generates a random specific id string based on a valid UUID value.
	ModeUUID idStringMode = iota

	// ModeHash generates a random specific id string based on a valid SHA3-256 value.
	ModeHash
)

func randomUUID() string {
	return uuid.New().String()
}

func randomHash() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return fmt.Sprintf("%x", sha3.Sum256(b))
}
