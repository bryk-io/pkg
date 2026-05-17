package paseto

import (
	"crypto"
	"io"

	"go.bryk.io/pkg/crypto/ed25519"
	"go.bryk.io/pkg/errors"
	cryptoutils "go.bryk.io/pkg/internal/crypto"
)

type edKey struct {
	kp *ed25519.KeyPair
	pv ProtocolVersion
	id string
}

func (k *edKey) ID() string {
	if k.id != "" {
		return k.id
	}
	k.id = cryptoutils.RandomID()
	return k.id
}

func (k *edKey) SetID(id string) {
	k.id = id
}

func (k *edKey) IsValid(tokenType string) bool {
	return tokenType == string(k.pv)
}

func (k *edKey) Public() crypto.PublicKey {
	pub := make([]byte, 32)
	exp := k.kp.PublicKey()
	copy(pub, exp[:])
	return crypto.PublicKey(pub)
}

func (k *edKey) Sign(_ io.Reader, message []byte, _ crypto.SignerOpts) ([]byte, error) {
	return k.kp.Sign(message), nil
}

func (k *edKey) Verify(message, signature []byte) bool {
	return k.kp.Verify(message, signature)
}

func (k *edKey) MarshalBinary() ([]byte, error) {
	return k.kp.MarshalBinary()
}

func (k *edKey) UnmarshalBinary(data []byte) error {
	var err error
	k.kp, err = ed25519.Unmarshal(data)
	return err
}

func (k *edKey) Export() (*KeyRecord, error) {
	sec, err := k.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return &KeyRecord{
		ID:       k.ID(),
		Protocol: string(k.pv),
		Secret:   b64.EncodeToString(sec),
	}, nil
}

func (k *edKey) Import(kr *KeyRecord) error {
	sk, err := b64.DecodeString(kr.Secret)
	if err != nil {
		return errors.New("invalid secret value")
	}
	if err = k.UnmarshalBinary(sk); err != nil {
		return err
	}
	k.id = kr.ID
	k.pv = ProtocolVersion(kr.Protocol)
	return nil
}

func (k *edKey) new(id string, pv ProtocolVersion) error {
	kp, err := ed25519.New()
	if err != nil {
		return err
	}
	k.kp = kp
	k.id = id
	k.pv = pv
	return nil
}
