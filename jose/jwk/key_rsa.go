package jwk

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"math/big"

	"go.bryk.io/pkg/errors"
	cryptoutils "go.bryk.io/pkg/internal/crypto"
	"go.bryk.io/pkg/jose/jwa"
)

// RSA generates a new random RSA cryptographic key based on the
// provided parameters.
func newRSA(bits int, pss bool) (Key, error) {
	k := new(rsaKey)
	if err := k.new(bits, pss); err != nil {
		return nil, err
	}
	return k, nil
}

type rsaKey struct {
	key *rsa.PrivateKey
	pss bool
	id  string
	alg jwa.Alg
}

func (k *rsaKey) ID() string {
	if k.id != "" {
		return k.id
	}
	k.id = cryptoutils.RandomID()
	return k.id
}

func (k *rsaKey) SetID(id string) {
	k.id = id
}

func (k *rsaKey) Alg() jwa.Alg {
	return k.alg
}

func (k *rsaKey) Thumbprint() (string, error) {
	return thumbprint(k, []string{"e", fieldKTY, "n"})
}

func (k *rsaKey) Sign(rr io.Reader, data []byte, hh crypto.SignerOpts) ([]byte, error) {
	// No private key
	if k.key == nil || k.key.D == nil {
		return nil, errors.New("key is 'verify' only")
	}

	hf := hh.HashFunc()
	ih := hf.New()
	if _, err := ih.Write(data); err != nil {
		return nil, err
	}
	msg := ih.Sum(nil)
	if !k.pss {
		return rsa.SignPKCS1v15(rr, k.key, hf, msg)
	}

	// PSS enabled
	opts := &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
		Hash:       hh.HashFunc(),
	}
	return rsa.SignPSS(rr, k.key, hf, msg, opts)
}

func (k *rsaKey) Verify(hh crypto.Hash, data, signature []byte) bool {
	ih := hh.New()
	if _, err := ih.Write(data); err != nil {
		return false
	}
	msg := ih.Sum(nil)
	if !k.pss {
		return rsa.VerifyPKCS1v15(&k.key.PublicKey, hh, msg, signature) == nil
	}

	// PSS enabled
	opts := &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
		Hash:       hh,
	}
	return rsa.VerifyPSS(&k.key.PublicKey, hh, msg, signature, opts) == nil
}

func (k *rsaKey) Public() crypto.PublicKey {
	return k.key.Public()
}

func (k *rsaKey) MarshalBinary() ([]byte, error) {
	bl := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(k.key),
	}
	return pem.EncodeToMemory(bl), nil
}

func (k *rsaKey) UnmarshalBinary(data []byte) error {
	bl, _ := pem.Decode(data)
	if bl == nil {
		return errors.New("invalid PEM data")
	}
	var err error
	k.key, err = x509.ParsePKCS1PrivateKey(bl.Bytes)
	return err
}

func (k *rsaKey) Export(safe bool) Record {
	kyt := keyTypeRSA
	if k.pss {
		kyt = keyTypePSS
	}
	rec := Record{
		KeyID:   k.ID(),
		KeyType: kyt,
		Use:     UseSignature,
		Alg:     string(k.alg),
		KeyOps:  []string{KeyOpVerify},
		N:       b64.EncodeToString(k.key.N.Bytes()),
		E:       b64.EncodeToString(big.NewInt(int64(k.key.E)).Bytes()),
	}
	if !safe {
		rec.KeyOps = append(rec.KeyOps, KeyOpSign)
		rec.D = b64.EncodeToString(k.key.D.Bytes())
		rec.P = b64.EncodeToString(k.key.Primes[0].Bytes())
		rec.Q = b64.EncodeToString(k.key.Primes[1].Bytes())
		rec.DP = b64.EncodeToString(k.key.Precomputed.Dp.Bytes())
		rec.DQ = b64.EncodeToString(k.key.Precomputed.Dq.Bytes())
		rec.Qi = b64.EncodeToString(k.key.Precomputed.Qinv.Bytes())
	}
	return rec
}

func (k *rsaKey) Import(r Record) error {
	// decode public key
	nB, err := b64.DecodeString(r.N)
	if err != nil {
		return errors.Wrap(err, "invalid 'N' value")
	}
	eB, err := b64.DecodeString(r.E)
	if err != nil {
		return errors.Wrap(err, "invalid 'E' value")
	}
	key := new(rsa.PrivateKey)
	key.PublicKey = rsa.PublicKey{
		N: new(big.Int).SetBytes(nB),
		E: int(new(big.Int).SetBytes(eB).Int64()),
	}
	k.alg = jwa.Alg(r.Alg)
	k.id = r.KeyID
	k.pss = r.KeyType == keyTypePSS
	k.key = key

	// no private key available
	if r.D == "" {
		return nil
	}

	// decode private key
	key.Primes = make([]*big.Int, 2)
	dB, err := b64.DecodeString(r.D)
	if err != nil {
		return errors.Wrap(err, "invalid 'd' value")
	}
	pB, err := b64.DecodeString(r.P)
	if err != nil {
		return errors.Wrap(err, "invalid 'P' value")
	}
	qB, err := b64.DecodeString(r.Q)
	if err != nil {
		return errors.Wrap(err, "invalid 'Q' value")
	}
	key.D = new(big.Int).SetBytes(dB)
	key.Primes[0] = new(big.Int).SetBytes(pB)
	key.Primes[1] = new(big.Int).SetBytes(qB)
	key.Precompute() // speed up private key operations
	if err = key.Validate(); err != nil {
		return err
	}
	k.key = key
	return nil
}

func (k *rsaKey) new(bits int, pss bool) error {
	var err error
	k.pss = pss
	k.key, err = rsa.GenerateKey(rand.Reader, bits)
	return err
}

// Validate checks the RSA key for compliance with RFC 7517 and RFC 7518
// security requirements. It verifies the key size meets minimum requirements
// and that the RSA key parameters are valid.
func (k *rsaKey) Validate() error {
	if err := k.validateBasic(); err != nil {
		return err
	}
	if err := k.validateAlgorithmType(); err != nil {
		return err
	}
	if err := k.validateKeySize(); err != nil {
		return err
	}
	if err := k.validatePublicKey(); err != nil {
		return err
	}
	if k.key.D != nil {
		if err := k.validatePrivateKey(); err != nil {
			return err
		}
	}
	return nil
}

// validateBasic checks that algorithm and key are set.
func (k *rsaKey) validateBasic() error {
	if k.alg == "" {
		return errors.New("RSA key algorithm is not set")
	}
	if k.key == nil {
		return errors.New("RSA key is nil")
	}
	return nil
}

// validateAlgorithmType checks algorithm prefix and PSS flag consistency.
func (k *rsaKey) validateAlgorithmType() error {
	if len(k.alg) < 2 {
		return errors.Errorf("unsupported RSA algorithm: %s", k.alg)
	}
	algPrefix := string(k.alg[0:2])
	switch algPrefix {
	case "RS":
		if k.pss {
			return errors.New("RSA key has PSS flag set but uses RS algorithm")
		}
	case "PS":
		if !k.pss {
			return errors.New("RSA key does not have PSS flag set but uses PS algorithm")
		}
	default:
		return errors.Errorf("unsupported RSA algorithm: %s", k.alg)
	}
	return nil
}

// validateKeySize checks minimum RSA key size requirement.
func (k *rsaKey) validateKeySize() error {
	const minRSABits = 2048
	keyBits := k.key.N.BitLen()
	if keyBits < minRSABits {
		return errors.Errorf("RSA key size %d bits is less than minimum required %d bits",
			keyBits, minRSABits)
	}
	return nil
}

// validatePublicKey checks public exponent and modulus validity.
func (k *rsaKey) validatePublicKey() error {
	if k.key.E <= 1 {
		return errors.New("RSA public exponent must be greater than 1")
	}
	if k.key.E%2 == 0 {
		return errors.New("RSA public exponent must be odd")
	}
	if k.key.N == nil || k.key.N.Sign() <= 0 {
		return errors.New("RSA modulus must be positive")
	}
	return nil
}

// validatePrivateKey checks private key components and runs standard validation.
func (k *rsaKey) validatePrivateKey() error {
	if k.key.D.Sign() <= 0 {
		return errors.New("RSA private exponent must be positive")
	}
	if len(k.key.Primes) < 2 {
		return errors.New("RSA key must have at least two prime factors")
	}
	for i, prime := range k.key.Primes {
		if prime == nil || prime.Sign() <= 0 {
			return errors.Errorf("RSA prime factor %d is invalid", i)
		}
	}
	if err := k.key.Validate(); err != nil {
		return errors.Wrap(err, "RSA key validation failed")
	}
	return nil
}
