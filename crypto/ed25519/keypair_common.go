package ed25519

import (
	e "crypto/ed25519"
	"crypto/sha512"
	"encoding/pem"
	"math/big"

	"go.bryk.io/pkg/errors"
	cryptoutils "go.bryk.io/pkg/internal/crypto"
	c "golang.org/x/crypto/curve25519"
)

// PEM header.
const keyType = "ED25519 PRIVATE KEY"

// New randomly generated Ed25519 (Digital Signature) key pair. Each
// KP needs to be securely removed from memory by calling the "Destroy"
// method.
func New() (*KeyPair, error) {
	_, priv, err := e.GenerateKey(nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate new random key")
	}
	defer wipeBytes(priv)
	return fromPrivateKey(priv)
}

// Unmarshal will restore a key pair instance from the provided
// PEM-encoded private key.
func Unmarshal(src []byte) (*KeyPair, error) {
	kp := new(KeyPair)
	if err := kp.UnmarshalBinary(src); err != nil {
		return nil, err
	}
	return kp, nil
}

// FromSeed deterministically generates a keypair instance using the
// provided seed material. The KP instance needs to be securely removed
// from memory by calling the "Destroy" method.
func FromSeed(seed []byte) (*KeyPair, error) {
	secret, err := cryptoutils.Expand(seed, e.SeedSize, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to expand seed")
	}
	defer wipeBytes(secret)
	priv := e.NewKeyFromSeed(secret)
	defer wipeBytes(priv)
	return fromPrivateKey(priv)
}

// FromPrivateKey restores a key pair instance using the provided
// private key value.
func FromPrivateKey(priv []byte) (*KeyPair, error) {
	if len(priv) != e.PrivateKeySize {
		return nil, errors.New("invalid private key")
	}
	return fromPrivateKey(priv)
}

// Verify performs a digital signature verification.
func Verify(message, signature, publicKey []byte) bool {
	if len(signature) != e.SignatureSize {
		return false
	}
	if len(publicKey) != e.PublicKeySize {
		return false
	}
	return e.Verify(publicKey, message, signature)
}

// ToCurve25519 converts an Ed25519 public key to a Curve25519 public key.
func ToCurve25519(pub [32]byte) []byte {
	// https://github.com/FiloSottile/age/blob/2194f6962c8bb3bca8a55f313d5b9302596b593b/agessh/agessh.go#L180-L209
	cp, _ := new(big.Int).SetString("57896044618658097711785492504343953926634992332820282019728792003956564819949", 10)

	// ed25519.PublicKey is a little endian representation of the y-coordinate,
	// with the most significant bit set based on the sign of the x-coordinate.
	bigEndianY := make([]byte, e.PublicKeySize)
	for i, b := range pub {
		bigEndianY[e.PublicKeySize-i-1] = b
	}
	bigEndianY[0] &= 0b0111_1111

	// The Montgomery u-coordinate is derived through the bilinear map
	//
	//     u = (1 + y) / (1 - y)
	//
	// See https://blog.filippo.io/using-ed25519-keys-for-encryption.
	y := new(big.Int).SetBytes(bigEndianY)
	denom := big.NewInt(1)
	denom.ModInverse(denom.Sub(denom, y), cp) // 1 / (1 - y)
	u := y.Mul(y.Add(y, big.NewInt(1)), denom)
	u.Mod(u, cp)

	// Return public bytes
	out := make([]byte, c.PointSize)
	uBytes := u.Bytes()
	for i, b := range uBytes {
		out[len(uBytes)-i-1] = b
	}
	return out
}

// UnmarshalBinary will restore a key pair instance from the provided
// PEM-encoded private key. The KP instance needs to be securely removed
// from memory by calling the "Destroy" method.
func (k *KeyPair) UnmarshalBinary(data []byte) error {
	bl, _ := pem.Decode(data)
	if bl == nil {
		return errors.New("invalid PEM data")
	}
	if bl.Type != keyType {
		return errors.Errorf("invalid key type: '%s'", bl.Type)
	}
	if len(bl.Bytes) != e.PrivateKeySize {
		return errors.New("invalid key size")
	}
	kp, err := fromPrivateKey(bl.Bytes)
	if err != nil {
		return err
	}
	k.Destroy()
	*k = *kp
	return nil
}

// MarshalBinary returns the PEM-encoded private key.
func (k *KeyPair) MarshalBinary() ([]byte, error) {
	bl := &pem.Block{
		Type:  keyType,
		Bytes: k.PrivateKey(),
	}
	return pem.EncodeToMemory(bl), nil
}

// PublicKey returns the public key bytes of the key pair instance.
func (k *KeyPair) PublicKey() [32]byte {
	return k.pub
}

// Sign generates a digital signature for the provided content.
func (k *KeyPair) Sign(message []byte) []byte {
	priv := k.PrivateKey()
	if priv == nil {
		return nil
	}
	// crypto/ed25519 requires the key to live in Go-managed memory, so use a
	// copy of the locked buffer contents.
	key := make([]byte, len(priv))
	copy(key, priv)
	defer wipeBytes(key)
	return e.Sign(key, message)
}

// Verify performs a digital signature verification.
func (k *KeyPair) Verify(message, signature []byte) bool {
	if len(signature) != e.SignatureSize {
		return false
	}
	pub := e.PublicKey(k.pub[:])
	return e.Verify(pub, message, signature)
}

// DH calculates a byte sequence which is the shared secret output from an
// Elliptic Curve Diffie-Hellman of the provided public key. In case of error
// the function return 'nil'.
func (k *KeyPair) DH(pub [32]byte) []byte {
	_, priv := k.toX25519()
	res, err := c.X25519(priv[:], pub[:])
	if err != nil {
		return nil
	}
	return res
}

// ToX25519 returns a X25519 public key generated using the original keypair
// as seed material. The returned public key is suitable to perform secure
// Diffie-Hellman agreements using a single keypair instance and matches the
// conversion provided by ToCurve25519.
func (k *KeyPair) ToX25519() [32]byte {
	pub, _ := k.toX25519()
	return pub
}

// Return an X25519 key pair using original private key as seed material.
// The conversion follows the standard algorithm: the X25519 scalar is
// derived from the Ed25519 seed (SHA-512 of the first 32 key bytes, then
// clamped) and the X25519 public key is the birational-map conversion of
// the Ed25519 public key.
func (k *KeyPair) toX25519() (pub [32]byte, priv [32]byte) {
	privKey := k.PrivateKey()
	if privKey == nil {
		return
	}

	// Get Curve25519 private key
	// https://cr.yp.to/ecdh.html
	h := sha512.Sum512(privKey[:e.SeedSize])
	copy(priv[:], h[:c.ScalarSize])
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64

	// Get Curve25519 public key
	cpub := ToCurve25519(k.PublicKey())
	copy(pub[:], cpub)
	return
}
