package jwt

import (
	"go.bryk.io/pkg/jose/jwk"
)

// ValidatorOption elements provide a functional-style configuration mechanism
// for token validators.
type ValidatorOption func(v *Validator) error

// WithValidationKeys registers the keys provided in the JWK set to be used
// for token validation.
func WithValidationKeys(set jwk.Set) ValidatorOption {
	return func(v *Validator) error {
		keys, err := expandSet(set)
		if err != nil {
			return err
		}
		v.keys = append(v.keys, keys...)
		return nil
	}
}
