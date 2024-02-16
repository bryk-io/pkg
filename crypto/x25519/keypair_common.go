package x25519

import (
	"crypto/rand"
	"encoding/pem"
	"fmt"

	"go.bryk.io/pkg/errors"
	cryptoutils "go.bryk.io/pkg/internal/crypto"
	c "golang.org/x/crypto/curve25519"
)

// PEM header.
const keyType = "X25519 PRIVATE KEY"

// New returns a X25519 (Diffie-Hellman) key pair instance. Each KP needs
// to be securely removed from memory by calling the "Destroy" method.
func New() (*KeyPair, error) {
	seed := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		return nil, errors.New("failed to generate random seed")
	}
	return fromPrivateKey(seed, true)
}

// FromSeed deterministically generates a keypair instance using the provided
// seed material. The KP instance needs to be securely removed from memory by
// calling the "Destroy" method.
func FromSeed(seed []byte) (*KeyPair, error) {
	// Expand provided seed to obtain the private key material
	priv, err := cryptoutils.Expand(seed, 32, nil)
	if err != nil {
		return nil, errors.New("failed to expand seed")
	}
	return fromPrivateKey(priv, true)
}

// Unmarshal will restore a key pair instance from the provided PEM-encoded
// private key. The KP instance needs to be securely removed from memory by
// calling the "Destroy" method.
func Unmarshal(src []byte) (*KeyPair, error) {
	kp := new(KeyPair)
	if err := kp.UnmarshalBinary(src); err != nil {
		return nil, err
	}
	return kp, nil
}

// MarshalBinary returns the PEM-encoded private key.
func (k *KeyPair) MarshalBinary() ([]byte, error) {
	bl := &pem.Block{
		Type:  keyType,
		Bytes: k.PrivateKey(),
	}
	return pem.EncodeToMemory(bl), nil
}

// UnmarshalBinary will restore a key pair instance from the provided
// PEM-encoded private key. The KP instance needs to be securely removed
// from memory by calling the "Destroy" method.
func (k *KeyPair) UnmarshalBinary(data []byte) error {
	bl, _ := pem.Decode(data)
	if bl.Type != keyType {
		return fmt.Errorf("invalid key type: '%s'", bl.Type)
	}
	if len(bl.Bytes) != 32 {
		return errors.New("invalid key size")
	}
	kp, err := fromPrivateKey(bl.Bytes, false)
	if err != nil {
		return err
	}
	*k = *kp
	return nil
}

// DH calculates a byte sequence which is the shared secret output from an
// Elliptic Curve Diffie-Hellman of the provided public key. In case of error
// the function return 'nil'.
func (k *KeyPair) DH(pub [32]byte) []byte {
	res, err := c.X25519(k.PrivateKey(), pub[:])
	if err != nil {
		return nil
	}
	return res
}

// PublicKey returns the public key bytes of the key pair instance.
func (k *KeyPair) PublicKey() [32]byte {
	return k.public
}
