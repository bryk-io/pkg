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

func (k *ecKey) Thumbprint() (string, error) {
	return thumbprint(k, []string{"crv", fieldKTY, "x", "y"})
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
	sb := s.Bytes()
	rbp := make([]byte, kbs) // nolint: prealloc
	sbp := make([]byte, kbs) // nolint: prealloc
	copy(rbp[kbs-len(rb):], rb)
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
		KeyType: keyTypeEC,
		Use:     UseSignature,
		Alg:     string(k.alg),
		KeyOps:  []string{KeyOpVerify},
		Crv:     k.sk.Curve.Params().Name,
		X:       b64.EncodeToString(k.sk.X.Bytes()),
		Y:       b64.EncodeToString(k.sk.Y.Bytes()),
	}
	if !safe {
		rec.KeyOps = append(rec.KeyOps, KeyOpSign)
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

// Validate checks the EC key for compliance with RFC 7517 and RFC 7518
// security requirements. It verifies the curve is valid for the algorithm
// and that the key coordinates are on the curve.
func (k *ecKey) Validate() error {
	// EC keys must have a valid algorithm
	if k.alg == "" {
		return errors.New("EC key algorithm is not set")
	}
	// Check key is not nil
	if k.sk == nil {
		return errors.New("EC key is nil")
	}
	// Validate algorithm is an EC algorithm
	if len(k.alg) < 2 || k.alg[0:2] != "ES" {
		return errors.Errorf("unsupported EC algorithm: %s", k.alg)
	}
	// Get the expected curve for the algorithm
	expectedCurve, err := k.alg.Curve()
	if err != nil {
		return errors.Wrap(err, "failed to get curve for algorithm")
	}
	// Verify the key's curve matches the expected curve
	if k.sk.Curve != expectedCurve {
		return errors.Errorf("curve mismatch: key uses %s but algorithm %s requires %s",
			k.sk.Curve.Params().Name, k.alg, expectedCurve.Params().Name)
	}
	// Verify the curve is one of the supported curves
	// RFC 7518 Section 3.4: Only P-256, P-384, P-521 are supported
	curveName := k.sk.Curve.Params().Name
	switch curveName {
	case "P-256", "P-384", "P-521":
		// Supported curves
	default:
		return errors.Errorf("unsupported curve: %s (only P-256, P-384, P-521 are supported)",
			curveName)
	}
	// Validate the public key coordinates are on the curve
	// This is a critical security check
	if k.sk.X == nil || k.sk.Y == nil {
		return errors.New("EC public key coordinates are not set")
	}
	// Check that the point is on the curve by verifying the curve equation
	// y² = x³ + ax + b (mod p)
	// nolint: staticcheck // IsOnCurve is deprecated but still the correct
	// approach for ecdsa package validation. The crypto/ecdh package is the
	// recommended alternative but is not a drop-in replacement for ecdsa.
	if !k.sk.Curve.IsOnCurve(k.sk.X, k.sk.Y) {
		return errors.New("EC public key point is not on the curve")
	}
	// If private key is present, validate it
	if k.sk.D != nil {
		// Ensure D is valid (positive and less than curve order)
		if k.sk.D.Sign() <= 0 {
			return errors.New("EC private key must be positive")
		}
		// D should be less than the curve order
		curveOrder := k.sk.Curve.Params().N
		if k.sk.D.Cmp(curveOrder) >= 0 {
			return errors.New("EC private key is not within valid range")
		}
	}
	return nil
}
