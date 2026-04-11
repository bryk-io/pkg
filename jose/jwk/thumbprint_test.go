package jwk

import (
	"crypto"
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"

	tdd "github.com/stretchr/testify/assert"
)

// RFC 7638 test vector.
type testVector struct {
	Description         string `json:"description"`
	JWK                 Record `json:"jwk"`
	HashAlgorithm       string `json:"hash_algorithm"`
	ThumbprintBase64URL string `json:"thumbprint_base64url"`
	CanonicalJWK        string `json:"canonical_jwk"`
	ExpectedKID         string `json:"expected_kid,omitempty"`
}

func loadTestVectors(t *testing.T) []testVector {
	data, err := os.ReadFile("testdata/rfc7638.json")
	if err != nil {
		t.Fatalf("failed to read test vectors: %v", err)
	}

	var vectors []testVector
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatalf("failed to parse test vectors: %v", err)
	}

	return vectors
}

func TestRecord_Thumbprint(t *testing.T) {
	assert := tdd.New(t)
	vectors := loadTestVectors(t)

	for _, tv := range vectors {
		t.Run(tv.Description, func(t *testing.T) {
			var hash crypto.Hash
			switch tv.HashAlgorithm {
			case "SHA-256":
				hash = crypto.SHA256
			case "SHA-384":
				hash = crypto.SHA384
			case "SHA-512":
				hash = crypto.SHA512
			default:
				t.Fatalf("unsupported hash algorithm: %s", tv.HashAlgorithm)
			}

			thumbprint, err := tv.JWK.Thumbprint(hash)
			assert.Nil(err, "thumbprint computation should not fail")
			assert.Equal(tv.ThumbprintBase64URL, thumbprint, "thumbprint should match expected value")
		})
	}

	t.Run("CanonicalJSON", func(t *testing.T) {
		for _, tv := range vectors {
			t.Run(tv.Description+"_canonical", func(t *testing.T) {
				fields := thumbprintFields[tv.JWK.KeyType]
				canonical, err := tv.JWK.buildCanonicalJSON(fields)
				assert.Nil(err)
				assert.Equal(tv.CanonicalJWK, canonical, "canonical JSON should match expected value")
			})
		}
	})

	t.Run("HashAlgorithms", func(t *testing.T) {
		rec := Record{
			KeyType: keyTypeRSA,
			N:       "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
			E:       "AQAB",
		}

		// Different hash algorithms should produce different thumbprints
		tp256, err := rec.Thumbprint(crypto.SHA256)
		assert.Nil(err)

		tp384, err := rec.Thumbprint(crypto.SHA384)
		assert.Nil(err)

		tp512, err := rec.Thumbprint(crypto.SHA512)
		assert.Nil(err)

		assert.NotEqual(tp256, tp384, "SHA-256 and SHA-384 should produce different thumbprints")
		assert.NotEqual(tp256, tp512, "SHA-256 and SHA-512 should produce different thumbprints")
		assert.NotEqual(tp384, tp512, "SHA-384 and SHA-512 should produce different thumbprints")
	})

	t.Run("PrivateKeyEqualsPublicKey", func(t *testing.T) {
		t.Run("RS256", func(t *testing.T) {
			// Generate a key
			key, err := New("RS256")
			assert.Nil(err)

			// Export full key (private)
			privateRec := key.Export(false)

			// Export public key only
			publicRec := key.Export(true)

			// Both should have the same thumbprint
			privateTP, err := privateRec.Thumbprint(crypto.SHA256)
			assert.Nil(err)

			publicTP, err := publicRec.Thumbprint(crypto.SHA256)
			assert.Nil(err)

			assert.Equal(privateTP, publicTP, "private and public key should have same thumbprint")
		})

		t.Run("ES256", func(t *testing.T) {
			// Generate an EC key
			key, err := New("ES256")
			assert.Nil(err)

			// Export full key (private)
			privateRec := key.Export(false)

			// Export public key only
			publicRec := key.Export(true)

			// Both should have the same thumbprint
			privateTP, err := privateRec.Thumbprint(crypto.SHA256)
			assert.Nil(err)

			publicTP, err := publicRec.Thumbprint(crypto.SHA256)
			assert.Nil(err)

			assert.Equal(privateTP, publicTP, "EC private and public key should have same thumbprint")
		})
	})

	t.Run("UnsupportedKeyType", func(t *testing.T) {
		rec := Record{
			KeyType: "UNKNOWN",
		}

		_, err := rec.Thumbprint(crypto.SHA256)
		assert.NotNil(err)
		assert.Contains(err.Error(), "unsupported key type")
	})

	t.Run("MissingRequiredField", func(t *testing.T) {
		// RSA key missing 'n'
		rec := Record{
			KeyType: keyTypeRSA,
			E:       "AQAB",
		}

		_, err := rec.Thumbprint(crypto.SHA256)
		assert.NotNil(err)
		assert.Contains(err.Error(), "missing required field")
	})
}

func TestThumbprintURI(t *testing.T) {
	assert := tdd.New(t)

	rec := Record{
		KeyType: keyTypeEC,
		Crv:     "P-256",
		X:       "f83OJ3D2xF1Bg8vub9tLe1gHMzV76e8Tus9uPHvRVEU",
		Y:       "x_FEzRu9m36HLN_tue659LNpXW6pCyStikYjKIWI5a0",
	}

	uri, err := rec.ThumbprintURI(crypto.SHA256)
	assert.Nil(err)

	// URI should have correct format
	assert.True(strings.HasPrefix(uri, "urn:ietf:params:oauth:jwk-thumbprint:sha-256:"),
		"URI should have correct prefix")

	// Parse the URI back
	hashAlgo, thumbprint, err := ParseThumbprintURI(uri)
	assert.Nil(err)
	assert.Equal("sha-256", hashAlgo)

	// Thumbprint should match direct computation
	tp, err := rec.Thumbprint(crypto.SHA256)
	assert.Nil(err)
	assert.Equal(tp, thumbprint)
}

func TestParseThumbprintURI(t *testing.T) {
	assert := tdd.New(t)

	tests := []struct {
		name         string
		uri          string
		expectError  bool
		errContains  string
		expectedAlgo string
	}{
		{
			name:         "valid SHA-256 URI",
			uri:          "urn:ietf:params:oauth:jwk-thumbprint:sha-256:NzbLsXh8uDCcd-6MNwXF4W_7noWXFZAfHkxZsRGC9Xs",
			expectError:  false,
			expectedAlgo: "sha-256",
		},
		{
			name:         "valid SHA-384 URI",
			uri:          "urn:ietf:params:oauth:jwk-thumbprint:sha-384:Ci6Z2XHMg_xu8Yh0_n6q8-Zm03m9_2tJQcX5UCcwbzMrQqR0JWWVFNZ",
			expectError:  false,
			expectedAlgo: "sha-384",
		},
		{
			name:        "missing prefix",
			uri:         "invalid:sha-256:test",
			expectError: true,
			errContains: "missing prefix",
		},
		{
			name:        "missing colon separator",
			uri:         "urn:ietf:params:oauth:jwk-thumbprint:sha-256",
			expectError: true,
			errContains: "missing hash algorithm or thumbprint",
		},
		{
			name:        "invalid base64url",
			uri:         "urn:ietf:params:oauth:jwk-thumbprint:sha-256:not@valid!",
			expectError: true,
			errContains: "invalid thumbprint encoding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashAlgo, thumbprint, err := ParseThumbprintURI(tt.uri)
			if tt.expectError {
				assert.NotNil(err)
				if tt.errContains != "" {
					assert.Contains(err.Error(), tt.errContains)
				}
			} else {
				assert.Nil(err)
				assert.Equal(tt.expectedAlgo, hashAlgo)
				assert.NotEmpty(thumbprint)
			}
		})
	}
}

func TestMatchThumbprint(t *testing.T) {
	assert := tdd.New(t)

	rec := Record{
		KeyType: keyTypeOct,
		K:       "GawgguFyGrWKav7AX4VKUg",
	}

	// Get actual thumbprint
	tp, err := rec.Thumbprint(crypto.SHA256)
	assert.Nil(err)

	// Should match
	matches, err := rec.MatchThumbprint(tp, crypto.SHA256)
	assert.Nil(err)
	assert.True(matches)

	// Should not match wrong thumbprint
	matches, err = rec.MatchThumbprint("wrongthumbprint", crypto.SHA256)
	assert.Nil(err)
	assert.False(matches)
}

func TestMatchThumbprintURI(t *testing.T) {
	assert := tdd.New(t)

	rec := Record{
		KeyType: keyTypeOct,
		K:       "GawgguFyGrWKav7AX4VKUg",
	}

	// Create URI
	uri, err := rec.ThumbprintURI(crypto.SHA256)
	assert.Nil(err)

	// Should match
	matches, err := rec.MatchThumbprintURI(uri)
	assert.Nil(err)
	assert.True(matches)

	// Should not match wrong URI
	matches, err = rec.MatchThumbprintURI("urn:ietf:params:oauth:jwk-thumbprint:sha-256:wrongthumbprint")
	assert.Nil(err)
	assert.False(matches)
}

func TestRecord_ThumbprintBytes(t *testing.T) {
	assert := tdd.New(t)

	rec := Record{
		KeyType: keyTypeEC,
		Crv:     "P-256",
		X:       "f83OJ3D2xF1Bg8vub9tLe1gHMzV76e8Tus9uPHvRVEU",
		Y:       "x_FEzRu9m36HLN_tue659LNpXW6pCyStikYjKIWI5a0",
	}

	// Get raw bytes
	tpBytes, err := rec.ThumbprintBytes(crypto.SHA256)
	assert.Nil(err)
	assert.Equal(32, len(tpBytes)) // SHA-256 produces 32 bytes

	// Get base64url string
	tpString, err := rec.Thumbprint(crypto.SHA256)
	assert.Nil(err)

	// Base64url decode should match raw bytes
	decoded, err := base64.RawURLEncoding.DecodeString(tpString)
	assert.Nil(err)
	assert.Equal(tpBytes, decoded)
}

func TestRecordString(t *testing.T) {
	assert := tdd.New(t)

	rec := Record{
		KeyType: keyTypeOct,
		K:       "GawgguFyGrWKav7AX4VKUg",
	}

	str := rec.String()
	assert.NotEmpty(str)

	// Should be valid base64url
	_, err := base64.RawURLEncoding.DecodeString(str)
	assert.Nil(err)

	// Invalid record should show error
	invalidRec := Record{KeyType: "INVALID"}
	str = invalidRec.String()
	assert.Contains(str, "invalid JWK")
}

func BenchmarkThumbprintRSA(b *testing.B) {
	rec := Record{
		KeyType: keyTypeRSA,
		N:       "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
		E:       "AQAB",
	}

	for i := 0; i < b.N; i++ {
		_, _ = rec.Thumbprint(crypto.SHA256)
	}
}

func BenchmarkThumbprintEC(b *testing.B) {
	rec := Record{
		KeyType: keyTypeEC,
		Crv:     "P-256",
		X:       "f83OJ3D2xF1Bg8vub9tLe1gHMzV76e8Tus9uPHvRVEU",
		Y:       "x_FEzRu9m36HLN_tue659LNpXW6pCyStikYjKIWI5a0",
	}

	for i := 0; i < b.N; i++ {
		_, _ = rec.Thumbprint(crypto.SHA256)
	}
}
