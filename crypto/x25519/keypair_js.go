//go:build js

package x25519

import (
	"go.bryk.io/pkg/errors"
	c "golang.org/x/crypto/curve25519"
)

// KeyPair represents a X25519 (Diffie-Hellman) public/private key.
type KeyPair struct {
	public  [32]byte
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

// Setup a key pair instance from the provided private key bytes
func fromPrivateKey(priv []byte, adjust bool) (*KeyPair, error) {
	if len(priv) != 32 {
		return nil, errors.New("private key must be 32 bytes")
	}

	// Adjust private key value
	// https://cr.yp.to/ecdh.html
	if adjust {
		priv[0] &= 248
		priv[31] &= 127
		priv[31] |= 64
	}

	// Get public key
	privateKey := [32]byte{}
	publicKey := [32]byte{}
	copy(privateKey[:], priv)
	c.ScalarBaseMult(&publicKey, &privateKey)
	wipeBytes(privateKey[:])

	// Clean and return key pair instance
	return &KeyPair{
		public:  publicKey,
		private: append([]byte(nil), priv...),
	}, nil
}
