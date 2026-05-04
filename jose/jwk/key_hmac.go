package jwk

import (
	"bytes"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"io"

	"go.bryk.io/pkg/errors"
	cryptoutils "go.bryk.io/pkg/internal/crypto"
	"go.bryk.io/pkg/jose/jwa"
)

// HMAC produces a new randomly generated cryptographic key
// of the specified size.
func newHMAC(size int) (Key, error) {
	k := new(hmacKey)
	if err := k.new(size); err != nil {
		return nil, err
	}
	return k, nil
}

type hmacKey struct {
	key []byte
	id  string
	alg jwa.Alg
}

func (k *hmacKey) ID() string {
	if k.id != "" {
		return k.id
	}
	k.id = cryptoutils.RandomID()
	return k.id
}

func (k *hmacKey) SetID(id string) {
	k.id = id
}

func (k *hmacKey) Alg() jwa.Alg {
	return k.alg
}

func (k *hmacKey) Thumbprint() (string, error) {
	return thumbprint(k, []string{"k", fieldKTY})
}

func (k *hmacKey) Sign(_ io.Reader, data []byte, hh crypto.SignerOpts) ([]byte, error) {
	// No private key
	if k.key == nil {
		return nil, errors.New("key is 'verify' only")
	}
	hf := hh.HashFunc()
	hm := hmac.New(hf.New, k.key)
	if _, err := hm.Write(data); err != nil {
		return nil, err
	}
	return hm.Sum(nil), nil
}

func (k *hmacKey) Verify(hh crypto.Hash, data, signature []byte) bool {
	hm := hmac.New(hh.New, k.key)
	if _, err := hm.Write(data); err != nil {
		return false
	}
	res := hm.Sum(nil)
	return bytes.Equal(res, signature)
}

func (k *hmacKey) Public() crypto.PublicKey {
	// HMAC keys are symmetric
	return nil
}

func (k *hmacKey) MarshalBinary() ([]byte, error) {
	dst := make([]byte, b64.EncodedLen(len(k.key)))
	b64.Encode(dst, k.key)
	return dst, nil
}

func (k *hmacKey) UnmarshalBinary(data []byte) error {
	k.key = make([]byte, b64.DecodedLen(len(data)))
	_, err := b64.Decode(k.key, data)
	return err
}

func (k *hmacKey) Export(safe bool) Record {
	rec := Record{
		KeyID:   k.ID(),
		KeyType: keyTypeOct,
		Use:     UseEncryption,
		Alg:     string(k.alg),
		KeyOps:  []string{KeyOpEncrypt, KeyOpDecrypt},
	}
	if !safe {
		rec.K = b64.EncodeToString(k.key)
	}
	return rec
}

func (k *hmacKey) Import(r Record) error {
	k.id = r.KeyID
	k.alg = jwa.Alg(r.Alg)

	// no private key available
	if r.K == "" {
		return nil
	}

	// decode private key
	var err error
	k.key, err = b64.DecodeString(r.K)
	return err
}

func (k *hmacKey) new(size int) error {
	var err error
	sec := make([]byte, size)
	if _, err = rand.Read(sec); err != nil {
		return err
	}
	k.key, err = cryptoutils.Expand(sec, size, nil)
	return err
}

// Validate checks the HMAC key for compliance with RFC 7517 and RFC 7518
// security requirements. It verifies the key size meets the minimum
// requirements for the algorithm.
func (k *hmacKey) Validate() error {
	// HMAC keys must have a valid algorithm
	if k.alg == "" {
		return errors.New("HMAC key algorithm is not set")
	}
	// Check key is not nil
	if k.key == nil {
		return errors.New("HMAC key is nil")
	}
	// Get minimum key size based on algorithm
	// RFC 7518 Section 3.2: HMAC keys must be >= hash size
	var minKeySize int
	switch k.alg {
	case jwa.HS256:
		minKeySize = 32 // 256 bits
	case jwa.HS384:
		minKeySize = 48 // 384 bits
	case jwa.HS512:
		minKeySize = 64 // 512 bits
	default:
		return errors.Errorf("unsupported HMAC algorithm: %s", k.alg)
	}
	if len(k.key) < minKeySize {
		return errors.Errorf("HMAC key size %d is less than minimum required %d for %s",
			len(k.key), minKeySize, k.alg)
	}
	return nil
}
