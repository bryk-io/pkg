package jwt

import (
	"go.bryk.io/pkg/errors"
	"go.bryk.io/pkg/jose/jwa"
	"go.bryk.io/pkg/jose/jwk"
)

// Validator instances can be used when tokens need to be validated but
// issuing is not possible or desired. For example when retrieving the
// server's JWK key set including only public keys.
type Validator struct {
	keys []jwk.Key
}

// NewValidator returns a new token validator instance ready to be used.
func NewValidator(opts ...ValidatorOption) (*Validator, error) {
	v := &Validator{keys: []jwk.Key{}}
	for _, opt := range opts {
		if err := opt(v); err != nil {
			return nil, err
		}
	}
	return v, nil
}

// Validate a previously generated token instance.
//  1. Is the string a valid JWT?
//  2. Is 'alg' supported by the generator?
//  3. Is the digital signature valid?
//  4. Run all provided checks
func (v *Validator) Validate(token string, checks ...Check) error {
	t, err := Parse(token)
	if err != nil {
		return err
	}

	// 'NONE' tokens require only payload validations
	alg := jwa.Alg(t.Header().Algorithm)
	if alg == jwa.NONE {
		return t.Validate(checks...)
	}

	// Verify 'alg' is supported
	if !isSupported(jwa.Alg(t.Header().Algorithm), v.keys) {
		return errors.New("unsupported 'alg' header")
	}

	// Verify signature for secure tokens
	if t.Header().Algorithm != string(jwa.NONE) {
		key := getKey(t.Header().KeyID, v.keys)
		if key == nil {
			return errors.New("invalid key identifier")
		}
		if err = verify(token, key); err != nil {
			return err
		}
	}

	// Basic payload validations
	return t.Validate(checks...)
}
