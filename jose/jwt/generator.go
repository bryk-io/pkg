package jwt

import (
	"strings"
	"sync"
	"time"

	"go.bryk.io/pkg/errors"
	"go.bryk.io/pkg/jose/jwa"
	"go.bryk.io/pkg/jose/jwk"
)

// Generator instances can be used to generate new tokens and validate
// the ones previously issued.
type Generator struct {
	name string
	keys []jwk.Key
	none bool
	mu   sync.Mutex
}

// NewGenerator returns a new generator instance ready to be used.
// A provider offers a JWT generation and validation interface. The
// provided `issuer` value will be included in the `issuer` header for
// all generated tokens.
func NewGenerator(issuer string, opts ...GeneratorOption) (*Generator, error) {
	g := &Generator{
		name: issuer,
		none: false,
		keys: []jwk.Key{},
	}
	for _, opt := range opts {
		if err := opt(g); err != nil {
			return nil, err
		}
	}
	return g, nil
}

// Name returns the generator identifier. This identifier will be set in the
// "iss" registered claim for all generated tokens.
func (g *Generator) Name() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.name
}

// IsSupported ensures the provided JWT method is supported by the generator
// based on the keys available.
func (g *Generator) IsSupported(alg jwa.Alg) bool {
	// Support for 'NONE' has been disabled
	if alg == jwa.NONE {
		return g.none
	}

	// Verify support with available keys
	g.mu.Lock()
	defer g.mu.Unlock()
	return isSupported(alg, g.keys)
}

// Issue a new token signed using the selected key and based on the provided
// configuration parameters. Pass "none" as the `keyID` to issue a jwa.NONE
// token, if supported by the generator.
func (g *Generator) Issue(keyID string, params *TokenParameters) (*Token, error) {
	// Verify "none" is supported if requested
	if strings.ToLower(keyID) == "none" && !g.none {
		return nil, errors.New("unsupported method")
	}

	// Verify/populate provided parameters
	if err := params.verify(); err != nil {
		return nil, err
	}

	// Verify key name
	var key jwk.Key
	if strings.ToLower(keyID) != "none" {
		key = getKey(keyID, g.keys)
		if key == nil {
			return nil, errors.Errorf("invalid key name '%s'", keyID)
		}

		// Set token `alg` value based on the key selected
		params.Method = string(key.Alg())
	}

	// Payload
	now := time.Now()
	var pl interface{} = RegisteredClaims{
		Issuer:         g.name,
		IssuedAt:       now.Unix(),
		ExpirationTime: now.Add(params.exp).Unix(),
		NotBefore:      now.Add(params.nbf).Unix(),
		Subject:        params.Subject,
		Audience:       params.Audience,
		JTI:            params.UniqueIdentifier,
	}

	// Handle custom data
	if params.CustomClaims != nil {
		var err error
		pl, err = merge(pl, params.CustomClaims)
		if err != nil {
			return nil, err
		}
	}

	// Generate token and sign instance
	token := &Token{
		pl: pl,
		he: Header{
			Type:        "JWT",
			Algorithm:   params.Method,
			ContentType: params.ContentType,
		},
	}
	if key != nil {
		token.he.KeyID = key.ID()
		if err := sign(token, key); err != nil {
			return nil, errors.New("failed to sign token")
		}
	}
	return token, nil
}

// Validate a previously generated token instance.
//  1. Is the string a valid JWT?
//  2. Is 'alg' supported by the generator?
//  3. Is the digital signature valid?
//  4. Run all provided checks
func (g *Generator) Validate(token string, checks ...Check) error {
	t, err := Parse(token)
	if err != nil {
		return err
	}

	// Verify 'alg' is supported
	if !g.IsSupported(jwa.Alg(t.Header().Algorithm)) {
		return errors.New("unsupported 'alg' header")
	}

	// Verify signature for secure tokens
	if t.Header().Algorithm != string(jwa.NONE) {
		key := getKey(t.Header().KeyID, g.keys)
		if key == nil {
			return errors.New("invalid key identifier")
		}
		if err = verify(token, key); err != nil {
			return err
		}
	}

	// Basic payload validations
	return t.Validate(append(checks, IssuerCheck(g.name))...)
}

// AddKey will register a new cryptographic key with the token generator. If the
// identifier is already used the method will return an error.
func (g *Generator) AddKey(keys ...jwk.Key) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, k := range keys {
		for _, ek := range g.keys {
			if ek.ID() == k.ID() {
				return errors.Errorf("duplicated key id: '%s'", k.ID())
			}
		}
		g.keys = append(g.keys, k)
	}
	return nil
}

// RemoveKey can be used to decommission an existing cryptographic key from the
// token generator. If the identifier doesn't exist, calling this method is a no-op.
func (g *Generator) RemoveKey(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	rm := -1
	for i, k := range g.keys {
		if k.ID() == id {
			rm = i
			break
		}
	}
	if rm > -1 {
		g.keys[rm] = g.keys[len(g.keys)-1] // copy last element to index `rm`.
		g.keys[len(g.keys)-1] = nil        // erase last element (write zero value).
		g.keys = g.keys[:len(g.keys)-1]    // truncate slice.
	}
}

// ExportKeys returns the cryptographic keys available on the generator as a
// portable JWK set. When `safe` is true, the private key information won't be
// included in the exported data.
// https://www.rfc-editor.org/rfc/rfc7517.html#section-5
func (g *Generator) ExportKeys(safe bool) jwk.Set {
	set := jwk.Set{Keys: make([]jwk.Record, len(g.keys))}
	for i, k := range g.keys {
		set.Keys[i] = k.Export(safe)
	}
	return set
}
