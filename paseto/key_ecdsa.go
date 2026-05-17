package paseto

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"io"
	"math/big"

	"go.bryk.io/pkg/errors"
	cryptoutils "go.bryk.io/pkg/internal/crypto"
)

type ecdsaKey struct {
	id  string            // identifier
	sk  *ecdsa.PrivateKey // secret key
	pv  ProtocolVersion   // protocol version
	crv elliptic.Curve    // curve parameters
}

func (k *ecdsaKey) ID() string {
	if k.id != "" {
		return k.id
	}
	k.id = cryptoutils.RandomID()
	return k.id
}

func (k *ecdsaKey) SetID(id string) {
	k.id = id
}

func (k *ecdsaKey) IsValid(tokenType string) bool {
	return tokenType == string(k.pv)
}

func (k *ecdsaKey) Public() crypto.PublicKey {
	return k.sk.Public()
}

// Sign the provided message. "message" will be automatically passed over
// SHA-384 before being signed. Returns the produced signature as: `r || s`.
func (k *ecdsaKey) Sign(rand io.Reader, message []byte, _ crypto.SignerOpts) ([]byte, error) {
	// SHA-384
	hh := crypto.SHA384
	ih := hh.New()
	if _, err := ih.Write(message); err != nil {
		return nil, err
	}
	msg := ih.Sum(nil)

	r, s, err := ecdsa.Sign(rand, k.sk, msg)
	if err != nil {
		return nil, err
	}
	sig := make([]byte, 0, len(r.Bytes())+len(s.Bytes()))
	sig = append(sig, r.Bytes()...)
	sig = append(sig, s.Bytes()...)
	return sig, nil
}

// Verify "signature" was generated for "message". The signature is expected
// to be encoded as: `r || s`. "message" will be automatically passed over
// SHA-384 before being verified.
func (k *ecdsaKey) Verify(message, signature []byte) bool {
	if len(signature) != 96 {
		return false // invalid signature size
	}
	pub, ok := k.Public().(*ecdsa.PublicKey)
	if !ok {
		return false // invalid public key
	}

	// SHA-384
	hh := crypto.SHA384
	ih := hh.New()
	if _, err := ih.Write(message); err != nil {
		return false
	}
	msg := ih.Sum(nil)

	r := big.NewInt(0)
	s := big.NewInt(0)
	r.SetBytes(signature[:48]) // leftmost 48 bytes
	s.SetBytes(signature[48:]) // rightmost 48 bytes
	return ecdsa.Verify(pub, msg, r, s)
}

func (k *ecdsaKey) MarshalBinary() ([]byte, error) {
	kb, err := x509.MarshalECPrivateKey(k.sk)
	if err != nil {
		return nil, errors.New("failed to marshal generated key")
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: kb,
	}), nil
}

func (k *ecdsaKey) UnmarshalBinary(data []byte) error {
	bl, _ := pem.Decode(data)
	if bl == nil {
		return errors.New("invalid PEM data")
	}
	var err error
	k.sk, err = x509.ParseECPrivateKey(bl.Bytes)
	k.pv = V3P
	k.crv = elliptic.P384()
	return err
}

func (k *ecdsaKey) Export() (*KeyRecord, error) {
	sk, err := k.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return &KeyRecord{
		ID:       k.ID(),
		Protocol: string(k.pv),
		Secret:   b64.EncodeToString(sk),
	}, nil
}

func (k *ecdsaKey) Import(kr *KeyRecord) error {
	sk, err := b64.DecodeString(kr.Secret)
	if err != nil {
		return errors.New("invalid secret value")
	}
	if err := k.UnmarshalBinary(sk); err != nil {
		return err
	}
	k.id = kr.ID
	k.pv = ProtocolVersion(kr.Protocol)
	k.crv = elliptic.P384()
	return nil
}

// Return the compressed format of the public key; as specified in SEC 1,
// Version 2.0, Section 2.3.3.
func (k *ecdsaKey) pubBytes() []byte {
	pub := k.sk.PublicKey
	return elliptic.MarshalCompressed(k.crv, pub.X, pub.Y)
}

func (k *ecdsaKey) new(id string, pv ProtocolVersion) error {
	crv := elliptic.P384()
	sk, err := ecdsa.GenerateKey(crv, rand.Reader)
	if err != nil {
		return err
	}
	k.id = id
	k.sk = sk
	k.pv = pv
	k.crv = crv
	return nil
}
