package jwa

import (
	"crypto"
	"crypto/elliptic"

	"go.bryk.io/pkg/errors"
)

// Alg values provide valid cryptographic algorithm identifiers as described
// by RFC-7518.
//
// Methods specify proper underlying configuration and settings required to
// generate and validate JWT instances using different hashing and signature
// mechanisms as defined in the specification. The method is also used to
// set the 'alg' header value.
//
// https://www.rfc-editor.org/rfc/rfc7518.html#section-3.1
type Alg string

const (
	// NONE - Insecure token, i.e, empty signature segment.
	NONE Alg = "none"
	// HS256 - HMAC using SHA-256.
	HS256 Alg = "HS256"
	// HS384 - HMAC using SHA-384.
	HS384 Alg = "HS384"
	// HS512 - HMAC using SHA-512.
	HS512 Alg = "HS512"
	// RS256 - RSASSA-PKCS1-v1_5 using SHA-256.
	RS256 Alg = "RS256"
	// RS384 - RSASSA-PKCS1-v1_5 using SHA-384.
	RS384 Alg = "RS384"
	// RS512 - RSASSA-PKCS1-v1_5 using SHA-512.
	RS512 Alg = "RS512"
	// PS256 - RSASSA-PSS using SHA-256 and MGF1 with SHA-256.
	PS256 Alg = "PS256"
	// PS384 - RSASSA-PSS using SHA-384 and MGF1 with SHA-384.
	PS384 Alg = "PS384"
	// PS512 - RSASSA-PSS using SHA-512 and MGF1 with SHA-512.
	PS512 Alg = "PS512"
	// ES256 - ECDSA using P-256 and SHA-256.
	ES256 Alg = "ES256"
	// ES384 - ECDSA using P-384 and SHA-384.
	ES384 Alg = "ES384"
	// ES512 - ECDSA using P-521 and SHA-512.
	ES512 Alg = "ES512"
)

// HashFunction returns the proper crypto function for the algorithm identifier.
func (a Alg) HashFunction() (crypto.Hash, error) {
	alg := string(a)
	switch s := alg[len(alg)-3:]; s {
	case "256":
		return crypto.SHA256, nil
	case "384":
		return crypto.SHA384, nil
	case "512":
		return crypto.SHA512, nil
	default:
		return crypto.SHA256, errors.Errorf("invalid hash suffix '%s'", s)
	}
}

// Curve returns the proper Elliptic curve for the algorithm identifier.
func (a Alg) Curve() (elliptic.Curve, error) {
	switch a {
	case ES256:
		return elliptic.P256(), nil
	case ES384:
		return elliptic.P384(), nil
	case ES512:
		return elliptic.P521(), nil
	default:
		return nil, errors.Errorf("invalid curve identifier %s", a)
	}
}
