package paseto

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"io"

	"go.bryk.io/pkg/errors"
	cryptoutils "go.bryk.io/pkg/internal/crypto"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

type hmacKey struct {
	id string          // identifier
	sk []byte          // secret key
	pv ProtocolVersion // protocol version
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

func (k *hmacKey) IsValid(tokenType string) bool {
	return tokenType == string(k.pv)
}

func (k *hmacKey) Secret() ([]byte, error) {
	return k.sk, nil
}

func (k *hmacKey) Export() (*KeyRecord, error) {
	return &KeyRecord{
		ID:       k.ID(),
		Protocol: string(k.pv),
		Secret:   hex.EncodeToString(k.sk),
	}, nil
}

func (k *hmacKey) Import(kr *KeyRecord) error {
	sk, err := hex.DecodeString(kr.Secret)
	if err != nil {
		return errors.New("invalid secret value")
	}
	k.sk = sk
	k.pv = ProtocolVersion(kr.Protocol)
	k.id = kr.ID
	return nil
}

func (k *hmacKey) new(id string, pv ProtocolVersion) error {
	bits := chacha20poly1305.KeySize
	seed := make([]byte, sha512.Size384)
	if _, err := rand.Read(seed); err != nil {
		return err
	}
	dk := hkdf.Expand(sha512.New384, seed, []byte("paseto-random-hmac-key"))
	k.sk = make([]byte, bits)
	k.pv = pv
	k.id = id
	_, err := io.ReadFull(dk, k.sk)
	return err
}
