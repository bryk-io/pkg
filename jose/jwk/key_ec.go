package jwk

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"

	"go.bryk.io/pkg/errors"
	cryptoutils "go.bryk.io/pkg/internal/crypto"
	"go.bryk.io/pkg/jose/jwa"
)

// EC generates a new random Elliptic-Curve cryptographic key
// based on the provided curve identifier.
func newEC(alg jwa.Alg) (Key, error) {
	crv, err := alg.Curve()
	if err != nil {
		return nil, err
	}
	k := new(ecKey)
	k.sk, err = ecdsa.GenerateKey(crv, rand.Reader)
	if err != nil {
		return nil, err
	}
	return k, nil
}

type ecKey struct {
	sk  *ecdsa.PrivateKey
	id  string
	alg jwa.Alg
}

func (k *ecKey) ID() string {
	if k.id != "" {
		return k.id
	}
	k.id = cryptoutils.RandomID()
	return k.id
}

func (k *ecKey) SetID(id string) {
	k.id = id
}

func (k *ecKey) Alg() jwa.Alg {
	return k.alg
}

func (k *ecKey) Sign(rr io.Reader, data []byte, hh crypto.SignerOpts) ([]byte, error) {
	// No private key
	if k.sk == nil || k.sk.D == nil {
		return nil, errors.New("key is 'verify' only")
	}

	// Get digest of original data
	ih := hh.HashFunc().New()
	if _, err := ih.Write(data); err != nil {
		return nil, err
	}
	msg := ih.Sum(nil)

	// Sign message
	r, s, err := ecdsa.Sign(rr, k.sk, msg[:])
	if err != nil {
		return nil, err
	}

	// Verify key size is secure to use with the selected hash method
	hhs := hh.HashFunc().Size() * 8
	if hhs == 512 {
		hhs = 521 // Adjustment for P521
	}
	kbs := k.sk.Curve.Params().BitSize / 8
	if k.sk.Curve.Params().BitSize%8 > 0 {
		kbs++
	}
	if k.sk.Curve.Params().BitSize != hhs {
		return nil, fmt.Errorf("invalid key size (%d) for selected hash method (%d)", k.sk.Curve.Params().BitSize, hhs)
	}

	// Encode signature
	rb := r.Bytes()
	rbp := make([]byte, kbs)
	copy(rbp[kbs-len(rb):], rb)
	sb := s.Bytes()
	sbp := make([]byte, kbs)
	copy(sbp[kbs-len(sb):], sb)
	signature := append(rbp, sbp...)
	return signature, nil
}

func (k *ecKey) Verify(hh crypto.Hash, data, signature []byte) bool {
	// Get digest of original data
	ih := hh.New()
	if _, err := ih.Write(data); err != nil {
		return false
	}
	msg := ih.Sum(nil)

	// Decode signature
	keySize := hh.Size()
	// Adjustment for P521
	if keySize == 64 {
		keySize = 66
	}
	if len(signature) != keySize*2 {
		// Wrong signature length
		return false
	}
	r := big.NewInt(0).SetBytes(signature[:keySize])
	s := big.NewInt(0).SetBytes(signature[keySize:])

	// Verify signature
	return ecdsa.Verify(&k.sk.PublicKey, msg[:], r, s)
}

func (k *ecKey) Public() crypto.PublicKey {
	return k.sk.PublicKey
}

func (k *ecKey) MarshalBinary() ([]byte, error) {
	kb, err := x509.MarshalECPrivateKey(k.sk)
	if err != nil {
		return nil, errors.New("failed to marshal generated key")
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: kb,
	}), nil
}

func (k *ecKey) UnmarshalBinary(data []byte) error {
	bl, _ := pem.Decode(data)
	if bl == nil {
		return errors.New("invalid PEM data")
	}
	var err error
	k.sk, err = x509.ParseECPrivateKey(bl.Bytes)
	return err
}

func (k *ecKey) Export(safe bool) Record {
	rec := Record{
		KeyID:   k.ID(),
		KeyType: "EC",
		Use:     "sig",
		Alg:     string(k.alg),
		KeyOps:  []string{"verify"},
		Crv:     k.sk.Curve.Params().Name,
		X:       b64.EncodeToString(k.sk.X.Bytes()),
		Y:       b64.EncodeToString(k.sk.Y.Bytes()),
	}
	if !safe {
		rec.KeyOps = append(rec.KeyOps, "sign")
		rec.D = b64.EncodeToString(k.sk.D.Bytes())
	}
	return rec
}

func (k *ecKey) Import(r Record) error {
	// validate curve identifier
	crv, err := jwa.Alg(r.Alg).Curve()
	if err != nil {
		return err
	}

	// decode public key
	xB, err := b64.DecodeString(r.X)
	if err != nil {
		return errors.Wrap(err, "invalid 'x' value")
	}
	x := new(big.Int).SetBytes(xB)
	yB, err := b64.DecodeString(r.Y)
	if err != nil {
		return errors.Wrap(err, "invalid 'y' value")
	}
	y := new(big.Int).SetBytes(yB)
	pub := ecdsa.PublicKey{
		X: x,
		Y: y,
	}
	k.id = r.KeyID
	k.alg = jwa.Alg(r.Alg)
	k.sk = &ecdsa.PrivateKey{
		D:         nil,
		PublicKey: pub,
	}
	k.sk.Curve = crv

	// no private key available
	if r.D == "" {
		return nil
	}

	// decode private key
	dB, err := b64.DecodeString(r.D)
	if err != nil {
		return errors.Wrap(err, "invalid 'd' value")
	}
	k.sk.D = new(big.Int).SetBytes(dB)
	return nil
}
