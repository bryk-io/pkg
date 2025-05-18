package jwk

import (
	"bytes"
	"crypto"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"go.bryk.io/pkg/errors"
	"go.bryk.io/pkg/jose/jwa"
)

// Base64 encoding used consistently by all standard keys.
var b64 = base64.RawURLEncoding

// New returns a new cryptographic key to sign tokens for the provided
// 'alg' identifier.
// https://www.rfc-editor.org/rfc/rfc7518.html#section-3.1
func New(alg jwa.Alg) (Key, error) {
	if alg == jwa.NONE {
		return nil, errors.New("no key is required for alg 'NONE'")
	}
	// nolint: forcetypeassert, errcheck
	switch alg[0:2] {
	case "HS":
		k, err := newHMAC(64)
		if err == nil {
			k.(*hmacKey).alg = alg
		}
		return k, err
	case "RS":
		k, err := newRSA(2048, false)
		if err == nil {
			k.(*rsaKey).alg = alg
		}
		return k, err
	case "PS":
		k, err := newRSA(2048, true)
		if err == nil {
			k.(*rsaKey).alg = alg
		}
		return k, err
	case "ES":
		k, err := newEC(alg)
		if err == nil {
			k.(*ecKey).alg = alg
		}
		return k, err
	default:
		return nil, errors.Errorf("invalid 'alg' value '%s'", alg)
	}
}

// Import a cryptographic key from its portable JWK representation.
func Import(jwk Record) (Key, error) {
	k, err := New(jwa.Alg(jwk.Alg))
	if err != nil {
		return nil, err
	}
	if err = k.Import(jwk); err != nil {
		return nil, err
	}
	return k, nil
}

// Calculate a JWK thumbprint as defined by RFC7638.
//
// https://www.rfc-editor.org/rfc/rfc7638.html
func thumbprint(k Key, segments []string) (string, error) {
	// get map of key parameters
	js, _ := json.Marshal(k.Export(true))
	params := make(map[string]any)
	if err := json.Unmarshal(js, &params); err != nil {
		return "", err
	}

	// create deterministic output of required segments
	j := len(segments)
	sb := bytes.NewBuffer(nil)
	sb.Write([]byte("{"))
	for i := range j {
		fmt.Fprintf(sb, "\"%s\":\"%s\"", segments[i], params[segments[i]])
		if i < j-1 {
			sb.Write([]byte(","))
		}
	}
	sb.Write([]byte("}"))

	// return b64-encoded SHA256 value of thumbprint
	hash := sha256.Sum256(sb.Bytes())
	return base64.URLEncoding.EncodeToString(hash[:]), nil
}

// Key represents a cryptographic key used to sign and verify JWT instances.
// Different key types are required to support the different algorithms (i.e.
// methods) included in the RFC-7519 specification.
type Key interface {
	// ID returns a unique identifier for the key. If no `id` was
	// explicitly set for the key, a deterministic fingerprint will
	// be returned instead.
	ID() string

	// SetID adjust the `id` value for the key instance.
	SetID(id string)

	// Alg returns the JWA cryptographic algorithm identifier intended for the key.
	Alg() jwa.Alg

	// Thumbprint returns a unique key identifier as defined by RFC-7638.
	//
	// https://www.rfc-editor.org/rfc/rfc7638.html
	Thumbprint() (string, error)

	// Public returns the public key corresponding to the opaque,
	// private key.
	Public() crypto.PublicKey

	// Sign will produce a valid digital signature. The original `data` will be
	// hashed using the provided `hh` hash function.
	Sign(rand io.Reader, data []byte, hh crypto.SignerOpts) (signature []byte, err error)

	// Verify the authenticity of a provided signature against the original data.
	Verify(hh crypto.Hash, data, signature []byte) bool

	// Export a portable representation of the key instance.
	// When `safe` is true, the private key information won't be included
	// in the exported data.
	// https://www.rfc-editor.org/rfc/rfc7517.html
	Export(safe bool) Record

	// Import a key instance from a previously generated record.
	Import(src Record) error

	// MarshalBinary encodes the receiver into a binary form and returns the result
	MarshalBinary() ([]byte, error)

	// UnmarshalBinary decodes the `data` produced by `MarshalBinary` into the
	// receiver.
	UnmarshalBinary(data []byte) error
}

// Record is an object that represents a cryptographic key.
// https://www.rfc-editor.org/rfc/rfc7517.html#section-4
type Record struct {
	// The "kty" (key type) parameter identifies the cryptographic algorithm
	// family used with the key, such as "RSA" or "EC".  "kty" values should
	// either be registered in the IANA "JSON Web Key Types" registry
	// established by [JWA] or be a value that contains a Collision-Resistant
	// name. The "kty" value is a case-sensitive string. This member MUST be
	// present.
	//
	// JWA: https://www.rfc-editor.org/rfc/rfc7518.html
	KeyType string `json:"kty" yaml:"kty" mapstructure:"kty"`

	// The "key_ops" (key operations) parameter identifies the operation(s)
	// for which the key is intended to be used.  The "key_ops" parameter is
	// intended for use cases in which public, private, or symmetric keys
	// may be present.
	//  - "sign" (compute digital signature or MAC)
	//  - "verify" (verify digital signature or MAC)
	//  - "encrypt" (encrypt content)
	//  - "decrypt" (decrypt content and validate decryption, if applicable)
	//  - "wrapKey" (encrypt key)
	//  - "unwrapKey" (decrypt key and validate decryption, if applicable)
	//  - "deriveKey" (derive key)
	//  - "deriveBits" (derive bits not to be used as a key)
	KeyOps []string `json:"key_ops" yaml:"key_ops" mapstructure:"key_ops"`

	// The "kid" (key ID) parameter is used to match a specific key.  This
	// is used, for instance, to choose among a set of keys within a JWK Set
	// during key rollover.  The structure of the "kid" value is
	// unspecified.  When "kid" values are used within a JWK Set, different
	// keys within the JWK Set SHOULD use distinct "kid" values.
	KeyID string `json:"kid" yaml:"kid" mapstructure:"kid"`

	// The "use" (public key use) parameter identifies the intended use of
	// the public key.  The "use" parameter is employed to indicate whether
	// a public key is used for encrypting ("enc") data or verifying the signature
	// on data ("sig").
	Use string `json:"use" yaml:"use" mapstructure:"use"`

	// The "alg" (algorithm) parameter identifies the algorithm intended for
	// use with the key. The values used should either be registered in the
	// IANA "JSON Web Signature and Encryption Algorithms".
	Alg string `json:"alg" yaml:"alg" mapstructure:"alg"`

	// The "k" (key value) parameter contains the value of the symmetric (or
	// other single-valued) key.
	K string `json:"k,omitempty" yaml:"k,omitempty" mapstructure:"k,omitempty"`

	// Curve identifier.
	Crv string `json:"crv,omitempty" yaml:"crv,omitempty" mapstructure:"crv,omitempty"`

	// X coordinate of the public key.
	X string `json:"x,omitempty" yaml:"x,omitempty" mapstructure:"x,omitempty"`

	// Y coordinate of the public key.
	Y string `json:"y,omitempty" yaml:"y,omitempty" mapstructure:"y,omitempty"`

	// The "d" (ECC private key) parameter contains the Elliptic Curve
	// private key value.
	D string `json:"d,omitempty" yaml:"d,omitempty" mapstructure:"d,omitempty"`

	// The "n" (modulus) parameter contains the modulus value for the RSA
	// public key.
	N string `json:"n,omitempty" yaml:"n,omitempty" mapstructure:"n,omitempty"`

	// The "e" (exponent) parameter contains the exponent value for the RSA
	// public key.
	E string `json:"e,omitempty" yaml:"e,omitempty" mapstructure:"e,omitempty"`

	// == Private RSA key fields.

	// The "p" (first prime factor) parameter contains the first prime
	// factor.
	P string `json:"p,omitempty" yaml:"p,omitempty" mapstructure:"p,omitempty"`

	// The "q" (second prime factor) parameter contains the second prime
	// factor.
	Q string `json:"q,omitempty" yaml:"q,omitempty" mapstructure:"q,omitempty"`

	// The "dp" (first factor CRT exponent) parameter contains the Chinese
	// Remainder Theorem (CRT) exponent of the first factor.
	DP string `json:"dp,omitempty" yaml:"dp,omitempty" mapstructure:"dp,omitempty"`

	// The "dq" (second factor CRT exponent) parameter contains the CRT
	// exponent of the second factor.
	DQ string `json:"dq,omitempty" yaml:"dq,omitempty" mapstructure:"dq,omitempty"`

	// The "qi" (first CRT coefficient) parameter contains the CRT
	// coefficient of the second factor.
	Qi string `json:"qi,omitempty" yaml:"qi,omitempty" mapstructure:"qi,omitempty"`

	// == EOF Private RSA key fields.

	// The "x5u" (X.509 URL) parameter is a URI [RFC3986] that refers to a
	// resource for an X.509 public key certificate or certificate chain
	// [RFC5280].
	CertificateURL string `json:"x5u,omitempty" yaml:"x5u,omitempty" mapstructure:"x5u,omitempty"`

	// The "x5c" (X.509 certificate chain) parameter contains a chain of one
	// or more PKIX certificates [RFC5280].  The certificate chain is
	// represented as a JSON array of certificate value strings. Each string in
	// the array is a base64-encoded PKIX certificate.
	CertificateChain []string `json:"x5c,omitempty" yaml:"x5c,omitempty" mapstructure:"x5c,omitempty"`

	// The "x5t" (X.509 certificate SHA-1 thumbprint) parameter is a
	// base64url-encoded SHA-1 thumbprint (a.k.a. digest) of the DER
	// encoding of an X.509 certificate
	CertificateThumbprintSHA1 string `json:"x5t,omitempty" yaml:"x5t,omitempty" mapstructure:"x5t,omitempty"`

	// The "x5t#S256" (X.509 certificate SHA-256 thumbprint) parameter is a
	// base64url-encoded SHA-256 thumbprint (a.k.a. digest) of the DER
	// encoding of an X.509 certificate
	CertificateThumbprintSHA2 string `json:"x5t#S256,omitempty" yaml:"x5t#S256,omitempty" mapstructure:"x5t#S256,omitempty"` // nolint: lll
}

// Set is an object that represents a collection of "JSON Web Keys".
// https://www.rfc-editor.org/rfc/rfc7517.html#section-5
type Set struct {
	// The value of the "keys" parameter is an array of JWK values.
	// By default, the order of the JWK values within the array does
	// not imply an order of preference among them.
	Keys []Record `json:"keys" yaml:"keys" mapstructure:"keys"`
}
