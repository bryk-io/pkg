package paseto

import (
	"strings"
	"sync"

	"go.bryk.io/pkg/errors"
)

// Generator instances can be used to generate new tokens and validate
// the ones previously issued.
type Generator struct {
	name string
	keys map[string]Key
	mu   sync.Mutex
}

// NewGenerator will return a new provider instance ready to be used.
// A provider offers a PASETO generation and validation interface. The provided
// "name" value will be included in the "iss" registered claim for all generated
// tokens.
func NewGenerator(name string) *Generator {
	return &Generator{
		name: name,
		keys: make(map[string]Key),
	}
}

// Name returns the generator identifier. This identifier will be set in the
// "iss" registered claim for all generated tokens.
func (g *Generator) Name() string {
	return g.name
}

// AddKey will register a new cryptographic key with the token generator.
func (g *Generator) AddKey(keys ...Key) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, k := range keys {
		id := k.ID()
		if _, ok := g.keys[id]; ok {
			return errors.Errorf("duplicated key: '%s'", id)
		}
		g.keys[id] = k
	}
	return nil
}

// GetKey returns a previously registered cryptographic key.
func (g *Generator) GetKey(id string) (Key, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	k, ok := g.keys[id]
	if !ok {
		return nil, errors.New("invalid key identifier")
	}
	return k, nil
}

// RemoveKey can be used to decommission an existing cryptographic key from the
// token generator. If the identifier doesn't exist, calling this method is a no-op.
func (g *Generator) RemoveKey(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.keys, id)
}

// Issue a token instance using the provided configuration parameters.
// The issuer will be automatically set to the generator's "name".
func (g *Generator) Issue(keyID string, params *TokenParameters) (*Token, error) {
	// Get key
	k, err := g.GetKey(keyID)
	if err != nil {
		return nil, err
	}

	// Validate key/token type
	if !k.IsValid(params.tokenType()) {
		return nil, errors.New("invalid key/token type")
	}

	// Get token footer
	ftrB, err := params.getFooter(k.ID())
	if err != nil {
		return nil, errors.Errorf("failed to encode footer claims: %w", err)
	}

	// Get token payload
	pldB, err := params.getPayload(g.name)
	if err != nil {
		return nil, errors.Errorf("failed to encode custom claims: %w", err)
	}

	// Token body
	ia := []byte(params.ImplicitAssertions)
	bdy, err := g.seal(params.tokenType(), k, pldB, ftrB, ia)
	if err != nil {
		return nil, err
	}

	t := &Token{
		vrn: params.Version,
		pps: params.Purpose,
		bdy: bdy,
		pld: pldB,
		ftr: ftrB,
	}
	return t, nil
}

// Validate a previously generated token.
//  1. Ensure the string is a valid PASETO
//  2. Ensure the payload was encrypted or signed by the generator
//  3. Ensure the "iss" claim matches the generator's name
//  4. Run additional provided checks
func (g *Generator) Validate(token string, checks ...Check) error {
	// Validate token string
	t, err := Parse(token)
	if err != nil {
		return err
	}

	// Get key
	kid := t.KeyID()
	if kid == "" {
		return errors.New("no 'kid' available")
	}
	k, err := g.GetKey(kid)
	if err != nil {
		return err
	}

	// Validate key/token type
	if !k.IsValid(t.Header()) {
		return errors.New("invalid key/token type")
	}

	// Validate encryption/signature
	pld, err := g.unseal(t, k, nil)
	if err != nil {
		return err
	}
	t.pld = pld

	// Run payload validations
	return t.Validate(append(checks, IssuerCheck(g.name))...)
}

// Unseal the token's payload. Attempting to unseal a not encrypted token
// is a no-op without returning an error.
func (g *Generator) Unseal(t *Token) error {
	// Not encrypted
	if !t.isEncrypted() {
		return nil // no-op
	}

	// Token is already unsealed
	if t.pld != nil {
		return nil
	}

	// Get key
	kid := t.KeyID()
	if kid == "" {
		return errors.New("no 'kid' available")
	}
	k, err := g.GetKey(kid)
	if err != nil {
		return err
	}

	// Validate key/token type
	if !k.IsValid(t.Header()) {
		return errors.New("invalid key/token type")
	}

	// Unseal payload
	pld, err := g.unseal(t, k, nil)
	if err != nil {
		return err
	}
	t.pld = pld
	return nil
}

// Return the signed/encrypted token body
//
//	tt  = token type
//	k   = key to encrypt/sign the generated token
//	pld = payload contents
//	ftr = footer contents, optional
//	ia  = implicit assertions, optional
func (g *Generator) seal(tt string, k Key, pld, ftr, ia []byte) ([]byte, error) {
	if strings.HasSuffix(tt, pLocal) {
		ek, ok := k.(EncryptionKey)
		if !ok {
			return nil, errors.New("invalid key")
		}
		return encrypt(tt, ek, pld, ftr, ia)
	}
	sk, ok := k.(SigningKey)
	if !ok {
		return nil, errors.New("invalid key")
	}
	return sign(tt, sk, pld, ftr, ia)
}

// Returns the unencrypted/verified token payload
//
//	t  = token
//	k  = cryptographic key
//	ia = implicit assertions, optional
func (g *Generator) unseal(t *Token, k Key, ia []byte) ([]byte, error) {
	if t.isEncrypted() {
		ek, ok := k.(EncryptionKey)
		if !ok {
			return nil, errors.New("invalid key")
		}
		return decrypt(t, ek, ia)
	}
	sk, ok := k.(SigningKey)
	if !ok {
		return nil, errors.New("invalid key")
	}
	return verify(t, sk, ia)
}
