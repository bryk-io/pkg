package jwt

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"dario.cat/mergo"
	"go.bryk.io/pkg/errors"
	"go.bryk.io/pkg/jose/jwa"
	"go.bryk.io/pkg/jose/jwk"
)

// Standard base64 encoding used.
var b64 = base64.RawURLEncoding

// Digitally sign the provided token instance.
func sign(token *Token, key jwk.Key) error {
	// No need to sign insecure tokens
	m := token.Header().Algorithm
	if m == string(jwa.NONE) {
		return nil
	}

	// Get token material
	mt, err := token.material()
	if err != nil {
		return err
	}

	// Get hash function
	hf, err := key.Alg().HashFunction()
	if err != nil {
		return err
	}

	// Sign token material
	token.sg, err = key.Sign(rand.Reader, mt, hf)
	return err
}

// Verify the digital signature on a token instance.
func verify(token string, key jwk.Key) error {
	// parse input
	t, err := Parse(token)
	if err != nil {
		return err
	}

	// Get hash function
	hm, err := key.Alg().HashFunction()
	if err != nil {
		return err
	}

	// Verify signature
	segments := strings.Split(token, ".")
	if !key.Verify(hm, []byte(fmt.Sprintf("%s.%s", segments[0], segments[1])), t.sg) {
		return errors.New("invalid signature")
	}

	// All good!
	return nil
}

// Encode the provided element using the standard base64 url-encoding method as
// defined in RFC-4648.
func encode(el interface{}, toJSON bool) ([]byte, error) {
	var err error
	data, _ := el.([]byte)
	if toJSON {
		data, err = json.Marshal(el)
		if err != nil {
			return nil, err
		}
	}
	return []byte(b64.EncodeToString(data)), nil
}

// Flatten the provided items into a single map structure.
func merge(item ...interface{}) (map[string]interface{}, error) {
	res := make(map[string]interface{})
	for _, el := range item {
		// Re-encode element into a map structure
		b, err := json.Marshal(el)
		if err != nil {
			return nil, errors.New("failed to encode item")
		}
		m := make(map[string]interface{})
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, errors.New("failed to re-encode item")
		}

		// Map element into final result, override entries is disabled by default
		if err := mergo.Map(&res, m); err != nil {
			return nil, errors.New("failed to merge item")
		}
	}
	return res, nil
}

// Retrieve a key instance based on its `id`.
func getKey(id string, keys []jwk.Key) jwk.Key {
	for _, k := range keys {
		if k.ID() == id {
			return k
		}
	}
	return nil
}

// Parse and import the cryptographic keys included in `set`.
func expandSet(set jwk.Set) ([]jwk.Key, error) {
	list := make([]jwk.Key, len(set.Keys))
	for i, kr := range set.Keys {
		k, err := jwk.Import(kr)
		if err != nil {
			return nil, err
		}
		list[i] = k
	}
	return list, nil
}

// Ensures `alg` is supported by the JWK set `keys`.
func isSupported(alg jwa.Alg, keys []jwk.Key) bool {
	for _, k := range keys {
		if k.Alg() == alg {
			return true
		}
	}
	return false
}
