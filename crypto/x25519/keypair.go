//go:build !js
// +build !js

package x25519

import (
	"github.com/awnumar/memguard"
	c "golang.org/x/crypto/curve25519"
)

// KeyPair represents a X25519 (Diffie-Hellman) public/private key.
type KeyPair struct {
	public [32]byte
	lb     *memguard.LockedBuffer
}

// PrivateKey returns the private key bytes of the key pair instance. Using
// this method may unintentionally expose secret material outside the security
// memory segment managed by the instance. Don't use it unless you really know
// what you are doing.
func (k *KeyPair) PrivateKey() []byte {
	if k.lb == nil {
		return nil
	}
	return k.lb.Bytes()
}

// Destroy will safely release the allocated mlock/VirtualLock memory.
func (k *KeyPair) Destroy() {
	if k.lb != nil {
		k.lb.Destroy()
	}
	memguard.WipeBytes(k.public[:])
	k.lb = nil
}

// Setup a key pair instance from the provided private key bytes.
func fromPrivateKey(priv []byte, adjust bool) (*KeyPair, error) {
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
	if pb, err := c.X25519(privateKey[:], c.Basepoint); err == nil {
		copy(publicKey[:], pb)
	}

	// Clean and return key pair instance
	memguard.ScrambleBytes(privateKey[:])
	memguard.WipeBytes(privateKey[:])
	return &KeyPair{
		public: publicKey,
		lb:     memguard.NewBufferFromBytes(priv),
	}, nil
}
