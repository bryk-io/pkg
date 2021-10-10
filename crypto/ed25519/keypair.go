//go:build !js
// +build !js

package ed25519

import (
	"github.com/awnumar/memguard"
	e "golang.org/x/crypto/ed25519"
)

// KeyPair represents a Ed25519 (Sign/Verify) public/private key.
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

// Setup a key pair instance from the provided private key.
func fromPrivateKey(priv e.PrivateKey) (*KeyPair, error) {
	// Load public key to a sized byte
	pub := [32]byte{}
	copy(pub[:], priv[32:])
	return &KeyPair{
		public: pub,
		lb:     memguard.NewBufferFromBytes(priv),
	}, nil
}
