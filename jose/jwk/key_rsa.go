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
	kyt := "RSA"
	if k.pss {
		kyt = "PSS"
	}
	rec := Record{
		KeyID:   k.ID(),
		KeyType: kyt,
		Use:     "sig",
		Alg:     string(k.alg),
		KeyOps:  []string{"verify"},
		N:       b64.EncodeToString(k.key.PublicKey.N.Bytes()),
		E:       b64.EncodeToString(big.NewInt(int64(k.key.PublicKey.E)).Bytes()),
	}
	if !safe {
		rec.KeyOps = append(rec.KeyOps, "sign")
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
	k.pss = r.KeyType == "PSS"
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
