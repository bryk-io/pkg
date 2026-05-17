package paseto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"

	"go.bryk.io/pkg/errors"
	cryptoutils "go.bryk.io/pkg/internal/crypto"
)

type rsaKey struct {
	sk *rsa.PrivateKey // secret key
	ch crypto.Hash     // crypto function used
	pv ProtocolVersion // protocol version
	id string          // identifier
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

func (k *rsaKey) IsValid(tokenType string) bool {
	return tokenType == string(k.pv)
}

func (k *rsaKey) Public() crypto.PublicKey {
	return k.sk.PublicKey
}

func (k *rsaKey) Sign(rand io.Reader, message []byte, _ crypto.SignerOpts) ([]byte, error) {
	ih := k.ch.New()
	if _, err := ih.Write(message); err != nil {
		return nil, err
	}
	msg := ih.Sum(nil)
	opts := &rsa.PSSOptions{
		Hash:       k.ch,
		SaltLength: rsa.PSSSaltLengthEqualsHash,
	}
	return rsa.SignPSS(rand, k.sk, k.ch, msg, opts)
}

func (k *rsaKey) Verify(message, signature []byte) bool {
	ih := k.ch.New()
	if _, err := ih.Write(message); err != nil {
		return false
	}
	msg := ih.Sum(nil)
	opts := &rsa.PSSOptions{
		Hash:       k.ch,
		SaltLength: rsa.PSSSaltLengthEqualsHash,
	}
	return rsa.VerifyPSS(&k.sk.PublicKey, k.ch, msg, signature, opts) == nil
}

func (k *rsaKey) MarshalBinary() ([]byte, error) {
	bl := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(k.sk),
	}
	return pem.EncodeToMemory(bl), nil
}

func (k *rsaKey) UnmarshalBinary(data []byte) error {
	bl, _ := pem.Decode(data)
	if bl == nil {
		return errors.New("invalid PEM data")
	}
	var err error
	k.sk, err = x509.ParsePKCS1PrivateKey(bl.Bytes)
	k.ch = crypto.SHA384
	k.pv = V1P
	return err
}

func (k *rsaKey) Export() (*KeyRecord, error) {
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

func (k *rsaKey) Import(kr *KeyRecord) error {
	sk, err := b64.DecodeString(kr.Secret)
	if err != nil {
		return errors.New("invalid secret value")
	}
	if err := k.UnmarshalBinary(sk); err != nil {
		return err
	}
	k.id = kr.ID
	k.ch = crypto.SHA384
	k.pv = V1P
	return nil
}

func (k *rsaKey) new(id string, bits int) error {
	var err error
	k.sk, err = rsa.GenerateKey(rand.Reader, bits)
	k.id = id
	k.ch = crypto.SHA384
	k.pv = V1P
	return err
}
