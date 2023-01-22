package did

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	secp "github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/mr-tron/base58"
	"go.bryk.io/pkg/crypto/ed25519"
	e "golang.org/x/crypto/ed25519"
)

// VerificationKey represents a cryptographic key according to the "Linked Data
// Cryptographic Suites".
// https://w3c-ccg.github.io/ld-cryptosuite-registry/
type VerificationKey struct {
	// Unique identifier for the key reference.
	ID string `json:"id" yaml:"id"`

	// Cryptographic suite identifier.
	Type KeyType `json:"type" yaml:"type"`

	// Subject controlling the corresponding private key.
	Controller string `json:"controller" yaml:"controller"`

	// Extensions used on the key instance.
	Extensions []Extension `json:"extensions,omitempty" yaml:"extensions,omitempty"`

	// Public key material encoded in the MULTIBASE format.
	// https://datatracker.ietf.org/doc/html/draft-multiformats-multibase-03
	Public string `json:"publicKeyMultibase,omitempty" yaml:"publicKeyMultibase,omitempty"`

	// Public key material encoded as base58.
	// https://w3c-ccg.github.io/security-vocab/contexts/security-v1.jsonld
	PublicKeyBase58 string `json:"publicKeyBase58,omitempty" yaml:"publicKeyBase58,omitempty"`

	// Private portion of the cryptographic key.
	Private []byte `json:"private,omitempty" yaml:"private,omitempty"`
}

// String uses the key ID value as its textual representation.
func (k *VerificationKey) String() string {
	return k.ID
}

// Bytes returns the byte representation of the public key properly decoding
// it from a value entry.
func (k *VerificationKey) Bytes() ([]byte, error) {
	if k.Type == KeyTypeEd {
		return multibaseDecode(k.Public)
	}
	return base58.Decode(k.PublicKeyBase58)
}

// Sign the provided input and return the generated signature value.
func (k *VerificationKey) Sign(data []byte) ([]byte, error) {
	return k.sign(data)
}

// Verify the validity of the provided input and signature values.
func (k *VerificationKey) Verify(data, signature []byte) bool {
	return k.verify(data, signature)
}

// ProduceSignatureLD generates a valid linked data signature for the provided
// data, usually a canonicalized version of JSON-LD document.
// https://w3c-dvcg.github.io/ld-signatures/#signature-algorithm
func (k *VerificationKey) ProduceSignatureLD(data []byte, domain string) (*SignatureLD, error) {
	// Set signature options
	sig := &SignatureLD{
		Type:    k.Type.SignatureType(),
		Context: []string{securityContext},
		Domain:  domain,
		Creator: k.ID,
		Created: time.Now().UTC().Format(time.RFC3339),
	}

	// Generate signature input value
	input, err := sig.GetInput(data)
	if err != nil {
		return nil, err
	}

	// Get signature
	sig.Value, err = k.sign(input)
	if err != nil {
		return nil, err
	}
	return sig, nil
}

// VerifySignatureLD validates the authenticity and integrity of a linked data
// signature using the public key instance.
// https://w3c-dvcg.github.io/ld-signatures/#signature-verification-algorithm
func (k *VerificationKey) VerifySignatureLD(data []byte, sig *SignatureLD) bool {
	// Get signature options
	sigOptions := &SignatureLD{
		Type:    k.Type.SignatureType(),
		Context: sig.Context,
		Domain:  sig.Domain,
		Nonce:   sig.Nonce,
		Creator: sig.Creator,
		Created: sig.Created,
	}

	// Generate signature input value
	input, err := sigOptions.GetInput(data)
	if err != nil {
		return false
	}

	// Verify signature value
	return k.verify(input, sig.Value)
}

// ProduceProof will generate a valid linked data proof for the provided data,
// usually a canonicalized version of JSON-LD document.
// https://w3c-dvcg.github.io/ld-proofs
func (k *VerificationKey) ProduceProof(data []byte, purpose, domain string) (*ProofLD, error) {
	// Set proof options
	p := &ProofLD{
		Context:            []string{securityContext},
		Type:               k.Type.SignatureType(),
		Domain:             domain,
		Created:            time.Now().UTC().Format(time.RFC3339),
		Purpose:            purpose,
		VerificationMethod: k.ID,
	}

	// Generate proof input value
	input, err := p.GetInput(data)
	if err != nil {
		return nil, err
	}

	// Set signature type and value
	p.Value, err = k.sign(input)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// VerifyProof will evaluate the authenticity and integrity of a linked data
// proof using the public key instance.
// https://w3c-ccg.github.io/ld-proofs/#create-verify-hash-algorithm
func (k *VerificationKey) VerifyProof(data []byte, proof *ProofLD) bool {
	// Get proof options
	p := &ProofLD{
		Context:            proof.Context,
		Type:               k.Type.SignatureType(),
		Domain:             proof.Domain,
		Created:            proof.Created,
		Purpose:            proof.Purpose,
		VerificationMethod: k.ID,
		Nonce:              proof.Nonce,
	}

	// Generate proof input value and return verification result
	// https://w3c-ccg.github.io/ld-proofs/#create-verify-hash-algorithm
	input, err := p.GetInput(data)
	if err != nil {
		return false
	}
	return k.verify(input, proof.Value)
}

// AddExtension can be used to register additional contextual information in the key instance.
// If another extension with the same id and version information, the data will be updated.
func (k *VerificationKey) AddExtension(ext Extension) {
	for i, ee := range k.Extensions {
		if ee.ID == ext.ID && ee.Version == ext.Version {
			k.Extensions[i] = ext
			return
		}
	}
	k.Extensions = append(k.Extensions, ext)
}

// GetExtension retrieves the information available for a given extension and decode it into
// the  provided holder instance (usually a pointer to a structure type). If no information is
// available or a decoding problems occurs an error will be returned.
func (k *VerificationKey) GetExtension(id string, version string, holder interface{}) error {
	for _, ee := range k.Extensions {
		if ee.ID == id && ee.Version == version {
			return ee.load(holder)
		}
	}
	return errors.New("no extension")
}

// Sign the provided data.
func (k *VerificationKey) sign(data []byte) ([]byte, error) {
	if len(k.Private) == 0 {
		return nil, errors.New("no private key available")
	}

	switch k.Type {
	case KeyTypeEd:
		pvt := e.PrivateKey(k.Private)
		return e.Sign(pvt, data), nil
	case KeyTypeRSA:
		block, _ := pem.Decode(k.Private)
		pvt, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		return rsaSign(pvt, data)
	case KeyTypeSecp256k1:
		pvt := secp.PrivKeyFromBytes(k.Private)
		if pvt == nil {
			return nil, errors.New("failed to decode private key")
		}
		ss := ecdsa.Sign(pvt, data)
		if ss == nil {
			return nil, errors.New("failed to sign message")
		}
		return ss.Serialize(), nil
	default:
		return nil, errors.New("invalid key type")
	}
}

// Verify the provided signature and original data.
func (k *VerificationKey) verify(data, signature []byte) bool {
	// Get public key bytes
	pubBytes, err := k.Bytes()
	if err != nil {
		return false
	}

	// Verify signature value
	switch k.Type {
	case KeyTypeEd:
		pub := e.PublicKey(pubBytes)
		return e.Verify(pub, data, signature)
	case KeyTypeRSA:
		block, _ := pem.Decode(pubBytes)
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return false
		}
		pk, ok := pub.(*rsa.PublicKey)
		if !ok {
			return false
		}
		return rsaVerify(pk, data, signature) == nil
	case KeyTypeSecp256k1:
		pub, err := secp.ParsePubKey(pubBytes)
		if err != nil {
			return false
		}
		sig, err := ecdsa.ParseDERSignature(signature)
		if err != nil {
			return false
		}
		return sig.Verify(data, pub)
	default:
		return false
	}
}

// Generates a new cryptographic key.
func newCryptoKey(kt KeyType) (*VerificationKey, error) {
	var pub []byte
	pk := &VerificationKey{Type: kt}

	// Create new key pair
	switch kt {
	case KeyTypeRSA:
		var err error
		pub, pk.Private, err = newRSAKey()
		if err != nil {
			return nil, err
		}
	case KeyTypeEd:
		key, err := ed25519.New()
		if err != nil {
			return nil, wrap(err, "failed to create new Ed25519 key")
		}

		kp := key.PublicKey()
		pub = make([]byte, 32)
		pk.Private = make([]byte, len(key.PrivateKey()))
		copy(pub, kp[:])
		copy(pk.Private, key.PrivateKey())
		key.Destroy()
	case KeyTypeSecp256k1:
		key, err := secp.GeneratePrivateKey()
		if err != nil {
			return nil, wrap(err, "failed to create new secp256k1 key")
		}
		pk.Private = key.Serialize()
		pub = key.PubKey().SerializeCompressed()
	}

	// Set encoded value
	if kt == KeyTypeEd {
		pk.Public = multibaseEncode(pub)
	} else {
		pk.PublicKeyBase58 = base58.Encode(pub)
	}
	return pk, nil
}

// Load an existing cryptographic key.
func loadExistingKey(private []byte, kt KeyType) (*VerificationKey, error) {
	pk := &VerificationKey{
		Type:    kt,
		Private: private,
	}

	// Use a challenge to validate key usage
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return nil, wrap(err, "failed to create challenge")
	}

	// Decode private key
	var (
		pub []byte
		err error
	)
	switch kt {
	case KeyTypeEd:
		pub, err = validateKeyEd(private, challenge)
		if err != nil {
			return nil, err
		}
	case KeyTypeSecp256k1:
		pub, err = validateKeySecp256k1(private, challenge)
		if err != nil {
			return nil, err
		}
	case KeyTypeRSA:
		pub, err = validateKeyRSA(private, challenge)
		if err != nil {
			return nil, err
		}
	}

	// Set encoded value
	if kt == KeyTypeEd {
		pk.Public = multibaseEncode(pub)
	} else {
		pk.PublicKeyBase58 = base58.Encode(pub)
	}
	return pk, nil
}

// Returns a 2048 bits RSA key pair in PEM-encoded DER PKIX format.
func newRSAKey() (pub []byte, priv []byte, err error) {
	// Generate key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	// Encode private key
	privBuf := bytes.NewBuffer(nil)
	privPem := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	if err = pem.Encode(privBuf, privPem); err != nil {
		return nil, nil, err
	}

	// Encode public key
	pubBuf := bytes.NewBuffer(nil)
	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	pubPem := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}
	if err = pem.Encode(pubBuf, pubPem); err != nil {
		return nil, nil, err
	}
	return pubBuf.Bytes(), privBuf.Bytes(), nil
}

// Produce a valid signature for the SHA256 hashed data using RSASSA-PKCS1-V1_5-SIGN.
func rsaSign(priv *rsa.PrivateKey, data []byte) ([]byte, error) {
	s, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, getHash(data))
	return s, err
}

// Verify an RSA PKCS#1 v1.5 signature.
func rsaVerify(pub *rsa.PublicKey, data, signature []byte) error {
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, getHash(data), signature)
}

// Generate a SHA256 digest value from the provided data.
func getHash(data []byte) []byte {
	h := sha256.New()
	if _, err := h.Write(data); err != nil {
		return nil
	}
	return h.Sum(nil)
}

// https://datatracker.ietf.org/doc/html/draft-multiformats-multibase-03
func multibaseEncode(data []byte) string {
	return "z" + base58.Encode(data)
}

// https://datatracker.ietf.org/doc/html/draft-multiformats-multibase-03
func multibaseDecode(src string) ([]byte, error) {
	base := src[:1]
	data := src[1:]
	// https://datatracker.ietf.org/doc/html/draft-multiformats-multibase-03#appendix-D.1
	switch base {
	case "z":
		return base58.Decode(data)
	case "f":
		return hex.DecodeString(data)
	case "m":
		return base64.RawStdEncoding.DecodeString(data)
	case "M":
		return base64.StdEncoding.DecodeString(data)
	case "u":
		return base64.RawURLEncoding.DecodeString(data)
	case "U":
		return base64.URLEncoding.DecodeString(data)
	default:
		return nil, fmt.Errorf("unsupported base identifier: %s", base)
	}
}

// Validate the provided 'private' key is Ed25519. Return
// the corresponding public key.
func validateKeyEd(private, challenge []byte) ([]byte, error) {
	// Validate private key length
	if len(private) != e.PrivateKeySize {
		return nil, errors.New("invalid Ed25519 private key")
	}

	// Validate provided key
	pvt := e.PrivateKey(private)
	s := e.Sign(pvt, challenge)
	pk, ok := pvt.Public().(e.PublicKey)
	if !ok {
		return nil, errors.New("invalid Ed25519 public key")
	}
	if !e.Verify(pk, challenge, s) {
		return nil, errors.New("invalid Ed25519 private key")
	}

	// Load public key contents
	pub := make([]byte, 32)
	copy(pub[:], pvt[32:])
	return pub, nil
}

// Validate the provided 'private' key is Secp256k1. Return
// the corresponding public key (compressed).
func validateKeySecp256k1(private, challenge []byte) ([]byte, error) {
	pvt := secp.PrivKeyFromBytes(private)
	if pvt == nil {
		return nil, errors.New("invalid secp256k1 private key")
	}
	pp := pvt.PubKey()
	ss := ecdsa.Sign(pvt, challenge)
	if ss == nil {
		return nil, errors.New("invalid secp256k1 private key")
	}
	if !ss.Verify(challenge, pp) {
		return nil, errors.New("invalid secp256k1 private key")
	}
	return pp.SerializeCompressed(), nil
}

// Validate the provided 'private' key is RSA. Return
// the corresponding public key (properly PEM encoded).
func validateKeyRSA(private, challenge []byte) ([]byte, error) {
	// Load private key
	block, _ := pem.Decode(private)
	if block == nil {
		return nil, errors.New("failed to decode RSA private key")
	}
	pvt, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, wrap(err, "failed to parse RSA private key")
	}

	// Validate provided key
	s, err := rsaSign(pvt, challenge)
	if err != nil {
		return nil, err
	}
	pk, ok := pvt.Public().(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("failed to parse RSA public key")
	}
	if err = rsaVerify(pk, challenge, s); err != nil {
		return nil, err
	}

	// Load public key contents
	pubBuf := bytes.NewBuffer(nil)
	pubBytes, err := x509.MarshalPKIXPublicKey(&pvt.PublicKey)
	if err != nil {
		return nil, wrap(err, "failed to parse public key")
	}
	pubPem := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}
	if err = pem.Encode(pubBuf, pubPem); err != nil {
		return nil, err
	}
	return pubBuf.Bytes(), nil
}
