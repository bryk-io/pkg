//go:build js

package ed25519

import (
	e "crypto/ed25519"

	"go.bryk.io/pkg/errors"
)

// KeyPair represents a Ed25519 (Sign/Verify) public/private key.
type KeyPair struct {
	pub     [32]byte
	private []byte
}

// PrivateKey returns the private key bytes of the key pair instance. Using
// this method may unintentionally expose secret material outside the security
// memory segment managed by the instance. Don't use it unless you really know
// what you are doing.
func (k *KeyPair) PrivateKey() []byte {
	return k.private
}

// Destroy will safely release the allocated mlock/VirtualLock memory
func (k *KeyPair) Destroy() {
	wipeBytes(k.private)
	k.private = nil
}

// Securely removes the contents of the provided buffer.
func wipeBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// Setup a key pair instance from the provided private key.
func fromPrivateKey(priv []byte) (*KeyPair, error) {
	if len(priv) != e.PrivateKeySize {
		return nil, errors.New("invalid private key")
	}

	// Load public key to a sized byte
	pub := [32]byte{}
	copy(pub[:], priv[e.SeedSize:])
	return &KeyPair{
		pub:     pub,
		private: append([]byte(nil), priv...),
	}, nil
}
