package jwt

import (
	"go.bryk.io/pkg/jose/jwk"
)

// GeneratorOption elements provide a functional-style configuration mechanism
// for token generators.
type GeneratorOption func(g *Generator) error

// WithSupportForNone enables support for the 'NONE' JWA algorithm; this
// is disabled by default. 'NONE' tokens are considered insecure.
func WithSupportForNone() GeneratorOption {
	return func(g *Generator) error {
		g.none = true
		return nil
	}
}

// WithKey registers a cryptographic key on the generator instance.
func WithKey(k jwk.Key) GeneratorOption {
	return func(g *Generator) error {
		return g.AddKey(k)
	}
}

// WithKeySet registers a JWK set of keys on the generator instance.
func WithKeySet(set jwk.Set) GeneratorOption {
	return func(g *Generator) error {
		keys, err := expandSet(set)
		if err != nil {
			return err
		}
		for _, k := range keys {
			if err = g.AddKey(k); err != nil {
				return err
			}
		}
		return nil
	}
}
