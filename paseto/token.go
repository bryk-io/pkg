package paseto

import (
	"encoding/json"
	"strings"

	"go.bryk.io/pkg/errors"
)

const (
	pLocal  = "local"
	pPublic = "public"
)

var validVersions = []string{
	"v1",
	"v2",
	"v3",
	"v4",
}

var validPurposes = []string{
	pLocal,
	pPublic,
}

// Check functions allow to execute verifications against a PASETO token.
type Check func(token *Token) error

// Parse a provided token string to a valid PASETO instance.
func Parse(token string) (*Token, error) {
	ts := strings.Split(token, ".")
	if len(ts) < 3 || len(ts) > 4 {
		return nil, errors.New("invalid token string")
	}
	if !in(ts[0], validVersions) {
		return nil, errors.Errorf("invalid version code: %s", ts[0])
	}
	if !in(ts[1], validPurposes) {
		return nil, errors.Errorf("invalid purpose code: %s", ts[1])
	}
	bdy, err := b64.DecodeString(ts[2])
	if err != nil {
		return nil, errors.New("invalid token payload")
	}
	t := &Token{
		vrn: ts[0],
		pps: ts[1],
		bdy: bdy,
	}
	if len(ts) == 4 {
		ftr, err := b64.DecodeString(ts[3])
		if err != nil {
			return nil, errors.New("invalid token footer")
		}
		t.ftr = ftr
	}
	return t, nil
}

// Token represents a cryptographically secure, compact, and URL-safe representation
// of claims that may be transferred between two parties. The claims are encoded in
// JSON, version-tagged, and either encrypted using shared-key cryptography or signed
// using public-key cryptography.
type Token struct {
	// Version header
	vrn string

	// Purpose header
	pps string

	// Token body, signed or encrypted based on version and purpose
	bdy []byte

	// Footer contents, optional
	ftr []byte

	// Token payload, decrypted for pLocal tokens
	pld []byte
}

// Version identifier based on the protocol definitions.
// https://tools.ietf.org/html/draft-paragon-paseto-rfc-00#section-2
func (t *Token) Version() string {
	return t.vrn
}

// Purpose identifier based on the protocol definitions.
// https://tools.ietf.org/html/draft-paragon-paseto-rfc-00#section-2
func (t *Token) Purpose() string {
	return t.pps
}

// Header returns the token "version.purpose"
// https://tools.ietf.org/html/draft-paragon-paseto-rfc-00#section-2
func (t *Token) Header() string {
	return strings.Join([]string{t.vrn, t.pps}, ".")
}

// Footer returns the token footer value, if any, as a base64-encoded string.
func (t *Token) Footer() string {
	return b64.EncodeToString(t.ftr)
}

// String returns a textual representation of the token instance.
//
//	With footer:
//	  version.purpose.payload.footer
//	Without footer:
//	  version.purpose.payload
//
// https://tools.ietf.org/html/draft-paragon-paseto-rfc-00#section-2
func (t *Token) String() string {
	segments := []string{
		t.vrn,
		t.pps,
		b64.EncodeToString(t.bdy),
	}
	if t.ftr != nil {
		segments = append(segments, b64.EncodeToString(t.ftr))
	}
	return strings.Join(segments, ".")
}

// KeyID returns the main identifier for the cryptographic key used to encrypt
// or sign the token. If they "kid" is not available in the footer, an empty
// string is returned by default.
// https://tools.ietf.org/html/draft-paragon-paseto-rfc-00#section-6.1.1
func (t *Token) KeyID() string {
	ftr := footerClaims{}
	if err := t.DecodeFooter(&ftr); err != nil {
		return ""
	}
	return ftr.KeyID
}

// DecodeFooter parses the content in the footer segment and stores the result
// in the value pointed by "v".
func (t *Token) DecodeFooter(v interface{}) error {
	return json.Unmarshal(t.ftr, v)
}

// DecodePayload parses the content in the payload segment and stores the result
// in the value pointed by "v". Encrypted tokens need to be unsealed before accessing
// its contents.
func (t *Token) DecodePayload(v interface{}) error {
	// encrypted and sealed (i.e. payload is not available)
	if t.isEncrypted() && t.pld == nil {
		return errors.New("token needs to be unsealed")
	}

	// encrypted but unsealed (i.e. payload is available)
	if t.isEncrypted() && t.pld != nil {
		return json.Unmarshal(t.pld, v)
	}

	// token protocol version
	tvp := ProtocolVersion(t.Header())

	// RSA signed token
	if tvp == V1P {
		if t.pld == nil {
			t.pld = t.bdy[:len(t.bdy)-256] // exclude rightmost 256 bytes used for the signature
		}
		return json.Unmarshal(t.pld, v)
	}

	// Ed25519 signed token
	if tvp == V2P || tvp == V4P {
		if t.pld == nil {
			t.pld = t.bdy[:len(t.bdy)-64] // exclude rightmost 64 bytes used for the signature
		}
		return json.Unmarshal(t.pld, v)
	}

	// ECDSA signed token
	if tvp == V3P {
		if t.pld == nil {
			t.pld = t.bdy[:len(t.bdy)-96] // exclude rightmost 96 bytes used for the signature
		}
		return json.Unmarshal(t.pld, v)
	}

	return errors.New("invalid token type")
}

// RegisteredClaims decoded from the token's payload. Encrypted tokens need to
// be unsealed before accessing its contents.
func (t *Token) RegisteredClaims() (*RegisteredClaims, error) {
	// pLocal encrypted token
	if t.Purpose() == pLocal && t.pld == nil {
		return nil, errors.New("token needs to be unsealed")
	}
	rc := RegisteredClaims{}

	// pLocal decrypted token
	if t.Purpose() == pLocal && t.pld != nil {
		return &rc, json.Unmarshal(t.pld, &rc)
	}

	// token protocol version
	tvp := ProtocolVersion(t.Header())

	// RSA signed token
	if tvp == V1P {
		if t.pld == nil {
			t.pld = t.bdy[:len(t.bdy)-256] // exclude rightmost 256 bytes used for the signature
		}
		return &rc, json.Unmarshal(t.pld, &rc)
	}

	// Ed25519 signed token
	if tvp == V2P || tvp == V4P {
		if t.pld == nil {
			t.pld = t.bdy[:len(t.bdy)-64] // exclude rightmost 64 bytes used for the signature
		}
		return &rc, json.Unmarshal(t.pld, &rc)
	}

	// ECDSA signed token
	if tvp == V3P {
		if t.pld == nil {
			t.pld = t.bdy[:len(t.bdy)-96] // exclude rightmost 96 bytes used for the signature
		}
		return &rc, json.Unmarshal(t.pld, &rc)
	}

	return &rc, nil
}

// Validate will apply the provided validator functions to the token instance.
func (t *Token) Validate(checks ...Check) error {
	for _, vl := range checks {
		if err := vl(t); err != nil {
			return err
		}
	}
	return nil
}

// Is the token encrypted/sealed?
func (t *Token) isEncrypted() bool {
	return t.Purpose() == pLocal
}
