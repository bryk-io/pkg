package jwt

import (
	"encoding/json"
	"fmt"
	"strings"

	"go.bryk.io/pkg/errors"
	xjson "go.bryk.io/pkg/internal/json"
)

// Token represents a specific JWT token instance.
type Token struct {
	he Header
	pl interface{}
	sg []byte
}

// Parse returns a functional token instance from its compact string representation.
func Parse(token string) (*Token, error) {
	segments := strings.Split(token, ".")
	if len(segments) != 3 {
		return nil, errors.New("invalid token string")
	}
	t := &Token{}

	// Decode header
	data, err := b64.DecodeString(segments[0])
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, &t.he); err != nil {
		return nil, err
	}

	// Decode payload
	data, err = b64.DecodeString(segments[1])
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, &t.pl); err != nil {
		return nil, err
	}

	// Decode signature (if present)
	if strings.TrimSpace(segments[2]) != "" {
		data, err = b64.DecodeString(segments[2])
		if err != nil {
			return nil, err
		}
		t.sg = data
	}

	// All good!
	return t, nil
}

// String returns a properly encoded and formatted textual representation of
// the token.
func (t *Token) String() string {
	hb, err := t.segment("he")
	if err != nil {
		return err.Error()
	}
	pb, err := t.segment("pl")
	if err != nil {
		return err.Error()
	}
	sb, err := t.segment("sg")
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("%s.%s.%s", hb, pb, sb)
}

// Bytes returns the binary contents of a properly encoded and formatted textual
// representation of the token.
func (t *Token) Bytes() []byte {
	return []byte(t.String())
}

// Header segment of the token instance.
func (t *Token) Header() Header {
	return t.he
}

// RegisteredClaims returns the "registered" claims section of the token.
func (t *Token) RegisteredClaims() (RegisteredClaims, error) {
	pl := RegisteredClaims{}
	if err := t.Decode(&pl); err != nil {
		return pl, errors.New("failed to decode token payload")
	}
	return pl, nil
}

// Decode will load the token payload segment (i.e., claims content) into the
// provided holder.
func (t *Token) Decode(v interface{}) error {
	pb, err := json.Marshal(t.pl)
	if err != nil {
		return err
	}
	return json.Unmarshal(pb, &v)
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

// Get a single claim from the token's payload. The provided `jp` value MUST be
// a valid JSON pointer as defined by RFC-6901.
// For example:
//   - "/iss"
//   - "/scope"
//   - "/custom/list/0"
//
// https://datatracker.ietf.org/doc/html/rfc6901
func (t *Token) Get(jp string) (interface{}, error) {
	p, err := xjson.ParseJP(jp)
	if err != nil {
		return nil, err
	}
	return p.Eval(t.pl)
}

// Retrieve a properly encoded specific segment of the token.
func (t *Token) segment(seg string) ([]byte, error) {
	switch seg {
	case "he":
		return encode(t.he, true)
	case "pl":
		return encode(t.pl, true)
	case "sg":
		return encode(t.sg, false)
	}
	return nil, errors.New("invalid segment")
}

// Token material to be signed.
func (t *Token) material() ([]byte, error) {
	hb, err := t.segment("he")
	if err != nil {
		return nil, err
	}
	pb, err := t.segment("pl")
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("%s.%s", hb, pb)), nil
}
