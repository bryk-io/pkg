package paseto

import (
	"crypto"
	"io"

	"go.bryk.io/pkg/errors"
)

// Supported protocol version identifiers.
const (
	// V1L = `v1.local`. Shared-key authenticated encrypted token.
	V1L ProtocolVersion = "v1.local"

	// V1P = `v1.public`. Digitally signed token, RSA-2048 (PSS).
	V1P ProtocolVersion = "v1.public"

	// V2L = `v2.local`. Shared-key authenticated encrypted token.
	V2L ProtocolVersion = "v2.local"

	// V2P = `v2.public`. Digitally signed token, Ed25519.
	V2P ProtocolVersion = "v2.public"

	// V3L = `v3.local`. Shared-key authenticated encrypted token.
	V3L ProtocolVersion = "v3.local"

	// V3P = `v3.public`. Digitally signed token, NIST P-384.
	V3P ProtocolVersion = "v3.public"

	// V4L = `v4.local`. Shared-key authenticated encrypted token.
	V4L ProtocolVersion = "v4.local"

	// V4P = `v4.public`. Digitally signed token, Ed25519.
	V4P ProtocolVersion = "v4.public"
)

// ProtocolVersion identifiers can be used to specify the technical
// details and purpose of an issued token.
type ProtocolVersion string

// KeyRecord provides a portable representation of cryptographic keys used
// in PASETO. Commonly used for persistent storage and import/export operations.
type KeyRecord struct {
	// ID is used to uniquely identify a key instance within a generator.
	ID string `json:"id" yaml:"id"`

	// Protocol version identifier.
	Protocol string `json:"protocol" yaml:"protocol"`

	// Secret cryptographic material (keep it safe).
	Secret string `json:"secret" yaml:"secret"`
}

// Key instances provide the required cryptographic capabilities to
// sign/encrypt PASETO tokens.
type Key interface {
	// ID returns a deterministic and unique identifier for the key instance.
	ID() string

	// SetID adjust the `id` value for the key instance.
	SetID(id string)

	// IsValid determines if the key is valid for a specific version/purpose
	// token type.
	IsValid(tokenType string) bool

	// Export the cryptographic material of the instance.
	Export() (*KeyRecord, error)

	// Import the cryptographic material of the instance.
	Import(kr *KeyRecord) error
}

// EncryptionKey can be used to generated `*.local` tokens. Meaning
// tokens that are encrypted rather than digitally signed.
type EncryptionKey interface {
	// Secret value useful for encryption/decryption. If the operations
	// are not supported an error must be returned.
	Secret() ([]byte, error)
}

// SigningKey can be used to generated `*.public` token. Meaning tokens
// that are digitally signed rather than encrypted.
type SigningKey interface {
	// Public returns the cryptographic public key associated with the
	// key instance or `nil` if not applicable.
	Public() crypto.PublicKey

	// Sign the provided message. If the operation is not supported an
	// error must be returned.
	Sign(rand io.Reader, message []byte, opts crypto.SignerOpts) ([]byte, error)

	// Verify a previously generated digital signature. If the operation is not supported
	// the method must return "false" by default.
	Verify(message, signature []byte) bool
}

// NewKey creates a new cryptographic key instance for the specified
// protocol version and identified by `id`.
func NewKey(id string, pv ProtocolVersion) (Key, error) {
	switch pv {
	case V1L:
		return newHMAC(id, pv)
	case V1P:
		k := new(rsaKey)
		if err := k.new(id, 2048); err != nil {
			return nil, err
		}
		return k, nil
	case V2L:
		return newHMAC(id, pv)
	case V2P:
		k := new(edKey)
		if err := k.new(id, pv); err != nil {
			return nil, err
		}
		return k, nil
	case V3L:
		return newHMAC(id, pv)
	case V3P:
		k := new(ecdsaKey)
		if err := k.new(id, pv); err != nil {
			return nil, err
		}
		return k, nil
	case V4L:
		return newHMAC(id, pv)
	case V4P:
		k := new(edKey)
		if err := k.new(id, pv); err != nil {
			return nil, err
		}
		return k, nil
	default:
		return nil, errors.New("invalid protocol version")
	}
}

// ImportKey restores a key instance from its portable representation.
func ImportKey(rec *KeyRecord) (Key, error) {
	k, err := NewKey(rec.ID, ProtocolVersion(rec.Protocol))
	if err != nil {
		return nil, err
	}
	if err = k.Import(rec); err != nil {
		return nil, err
	}
	return k, nil
}

func newHMAC(id string, pv ProtocolVersion) (Key, error) {
	k := new(hmacKey)
	if err := k.new(id, pv); err != nil {
		return nil, err
	}
	return k, nil
}
