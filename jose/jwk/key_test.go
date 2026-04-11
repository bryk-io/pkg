package jwk

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"

	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/jose/jwa"
)

// https://www.rfc-editor.org/rfc/rfc7518.html#section-3.1
var standardMethods = []string{
	string(jwa.NONE),
	string(jwa.HS256),
	string(jwa.HS384),
	string(jwa.HS512),
	string(jwa.RS256),
	string(jwa.RS384),
	string(jwa.RS512),
	string(jwa.ES256),
	string(jwa.ES384),
	string(jwa.ES512),
	string(jwa.PS256),
	string(jwa.PS384),
	string(jwa.PS512),
}

func TestNewKey(t *testing.T) {
	assert := tdd.New(t)

	for _, alg := range standardMethods {
		if alg == string(jwa.NONE) {
			continue
		}
		k, err := New(jwa.Alg(alg))
		assert.Nil(err, "failed to create key")
		k.SetID(sampleID())
		assert.Equal(jwa.Alg(alg), k.Alg(), "wrong 'alg'")
		assert.Nil(k.Validate())

		// Marshal
		_, err = k.MarshalBinary()
		assert.Nil(err, "marshal")

		// Print thumbprint
		tp, err := k.Thumbprint()
		assert.Nil(err)
		t.Logf("thumbprint: %s", tp)

		// Produce signature
		hm, _ := jwa.Alg(alg).HashFunction()
		msg := []byte("original message to sign")
		sig, err := k.Sign(rand.Reader, msg, hm)
		assert.Nil(err, "sign error")

		// Export and import full key
		rec := k.Export(false)
		js, _ := json.Marshal(rec)
		t.Logf("%s\n", js)
		k2, err := Import(rec)
		assert.Nil(err, "import")
		assert.True(k2.Verify(hm, msg, sig), "bad verify result")

		// HMAC keys are symmetric; there's no "pub-only" version available
		if strings.HasPrefix(alg, "HS") {
			continue
		}

		// Import only pub key
		onlyPub, err := Import(k.Export(true))
		assert.Nil(err, "import")
		assert.NotNil(onlyPub.Public(), "retrieve public key")
		assert.True(onlyPub.Verify(hm, msg, sig), "bad verify result")
		_, err = onlyPub.Sign(rand.Reader, msg, hm)
		assert.NotNil(err, "sign should fail")
	}
}

func sampleID() string {
	seed := make([]byte, 4)
	_, _ = rand.Read(seed)
	return fmt.Sprintf("%X-%X", seed[:2], seed[2:])
}

// TestKeyValidateInvalidHMAC tests validation of invalid HMAC keys.
func TestKeyValidateInvalidHMAC(t *testing.T) {
	assert := tdd.New(t)

	// Test HMAC256 with key too short
	k := &hmacKey{
		alg: jwa.HS256,
		key: make([]byte, 16), // Too short (needs 32 bytes minimum)
	}
	err := k.Validate()
	assert.NotNil(err, "should fail with key too short")
	assert.Contains(err.Error(), "less than minimum")

	// Test nil key
	k = &hmacKey{
		alg: jwa.HS256,
		key: nil,
	}
	err = k.Validate()
	assert.NotNil(err, "should fail with nil key")
	assert.Contains(err.Error(), "nil")

	// Test empty algorithm
	k = &hmacKey{
		alg: "",
		key: make([]byte, 64),
	}
	err = k.Validate()
	assert.NotNil(err, "should fail with empty algorithm")
}

// TestKeyValidateInvalidRSA tests validation of invalid RSA keys.
func TestKeyValidateInvalidRSA(t *testing.T) {
	assert := tdd.New(t)

	// Test with empty algorithm
	k := &rsaKey{
		alg: "RS256",
		key: nil,
	}
	err := k.Validate()
	assert.NotNil(err, "should fail with nil key")

	// Test with mismatched PSS flag
	k = &rsaKey{
		alg: jwa.RS256,
		key: &rsa.PrivateKey{
			PublicKey: rsa.PublicKey{
				N: big.NewInt(0).SetUint64(0),
				E: 0,
			},
		},
		pss: true, // Should be false for RS256
	}
	err = k.Validate()
	assert.NotNil(err, "should fail with PSS mismatch")

	// Test with small key
	k = &rsaKey{
		alg: jwa.RS256,
		key: &rsa.PrivateKey{
			PublicKey: rsa.PublicKey{
				N: big.NewInt(12345),
				E: 3,
			},
		},
		pss: false,
	}
	err = k.Validate()
	assert.NotNil(err, "should fail with key too small")
}

// TestKeyValidateInvalidEC tests validation of invalid EC keys.
func TestKeyValidateInvalidEC(t *testing.T) {
	assert := tdd.New(t)

	// Test with empty algorithm
	k := &ecKey{
		alg: "",
		sk:  nil,
	}
	err := k.Validate()
	assert.NotNil(err, "should fail with nil key")

	// Test with unsupported curve
	curve := elliptic.P224() // Not supported for JWS
	k = &ecKey{
		alg: jwa.ES256,
		sk: &ecdsa.PrivateKey{
			PublicKey: ecdsa.PublicKey{
				Curve: curve,
				X:     curve.Params().Gx,
				Y:     curve.Params().Gy,
			},
		},
	}
	err = k.Validate()
	assert.NotNil(err, "should fail with unsupported curve")

	// Test with point not on curve
	curve = elliptic.P256()
	k = &ecKey{
		alg: jwa.ES256,
		sk: &ecdsa.PrivateKey{
			PublicKey: ecdsa.PublicKey{
				Curve: curve,
				X:     big.NewInt(1),
				Y:     big.NewInt(2), // Not on curve
			},
		},
	}
	err = k.Validate()
	assert.NotNil(err, "should fail with point not on curve")
}

// TestImportWithValidation tests that Import validates keys.
func TestImportWithValidation(t *testing.T) {
	assert := tdd.New(t)

	// Test importing a valid key
	k, err := New(jwa.HS256)
	assert.Nil(err)
	rec := k.Export(false)
	k2, err := Import(rec)
	assert.Nil(err, "should import valid key without error")
	assert.NotNil(k2, "imported key should not be nil")

	// Verify the imported key is valid
	err = k2.Validate()
	assert.Nil(err, "imported key should be valid")
}

//! MARK: benchmarks

// BenchmarkKeyCreation measures the performance of creating new keys for
// different algorithm types.
func BenchmarkKeyCreation(b *testing.B) {
	algorithms := []struct {
		name string
		alg  jwa.Alg
	}{
		{"HS256", jwa.HS256},
		{"HS384", jwa.HS384},
		{"HS512", jwa.HS512},
		{"RS256", jwa.RS256},
		{"RS384", jwa.RS384},
		{"RS512", jwa.RS512},
		{"PS256", jwa.PS256},
		{"PS384", jwa.PS384},
		{"PS512", jwa.PS512},
		{"ES256", jwa.ES256},
		{"ES384", jwa.ES384},
		{"ES512", jwa.ES512},
	}

	for _, tc := range algorithms {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := New(tc.alg)
				if err != nil {
					b.Fatalf("failed to create key: %v", err)
				}
			}
		})
	}
}

// BenchmarkKeyImport measures the performance of importing keys from
// their portable JWK Record representation.
func BenchmarkKeyImport(b *testing.B) {
	algorithms := []struct {
		name string
		alg  jwa.Alg
	}{
		{"HS256", jwa.HS256},
		{"HS384", jwa.HS384},
		{"HS512", jwa.HS512},
		{"RS256", jwa.RS256},
		{"RS384", jwa.RS384},
		{"RS512", jwa.RS512},
		{"PS256", jwa.PS256},
		{"PS384", jwa.PS384},
		{"PS512", jwa.PS512},
		{"ES256", jwa.ES256},
		{"ES384", jwa.ES384},
		{"ES512", jwa.ES512},
	}

	for _, tc := range algorithms {
		// Create a key to export for import testing
		key, err := New(tc.alg)
		if err != nil {
			b.Fatalf("failed to create key for import test: %v", err)
		}
		record := key.Export(false)

		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Import(record)
				if err != nil {
					b.Fatalf("failed to import key: %v", err)
				}
			}
		})
	}
}

// BenchmarkKeyImportPublicOnly measures the performance of importing
// only public key components (for asymmetric keys).
func BenchmarkKeyImportPublicOnly(b *testing.B) {
	asymmetricAlgs := []struct {
		name string
		alg  jwa.Alg
	}{
		{"RS256", jwa.RS256},
		{"RS384", jwa.RS384},
		{"RS512", jwa.RS512},
		{"PS256", jwa.PS256},
		{"PS384", jwa.PS384},
		{"PS512", jwa.PS512},
		{"ES256", jwa.ES256},
		{"ES384", jwa.ES384},
		{"ES512", jwa.ES512},
	}

	for _, tc := range asymmetricAlgs {
		// Create a key and export public-only for import testing
		key, err := New(tc.alg)
		if err != nil {
			b.Fatalf("failed to create key for import test: %v", err)
		}
		record := key.Export(true) // Export public-only

		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Import(record)
				if err != nil {
					b.Fatalf("failed to import public key: %v", err)
				}
			}
		})
	}
}

// BenchmarkKeyValidation measures the performance of validating keys
// of different types.
func BenchmarkKeyValidation(b *testing.B) {
	algorithms := []struct {
		name string
		alg  jwa.Alg
	}{
		{"HS256", jwa.HS256},
		{"HS384", jwa.HS384},
		{"HS512", jwa.HS512},
		{"RS256", jwa.RS256},
		{"RS384", jwa.RS384},
		{"RS512", jwa.RS512},
		{"PS256", jwa.PS256},
		{"PS384", jwa.PS384},
		{"PS512", jwa.PS512},
		{"ES256", jwa.ES256},
		{"ES384", jwa.ES384},
		{"ES512", jwa.ES512},
	}

	for _, tc := range algorithms {
		// Create a key for validation testing
		key, err := New(tc.alg)
		if err != nil {
			b.Fatalf("failed to create key for validation test: %v", err)
		}

		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := key.Validate()
				if err != nil {
					b.Fatalf("key validation failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkKeyValidationAfterImport measures the combined performance
// of importing and validating keys.
func BenchmarkKeyValidationAfterImport(b *testing.B) {
	algorithms := []struct {
		name string
		alg  jwa.Alg
	}{
		{"HS256", jwa.HS256},
		{"HS384", jwa.HS384},
		{"HS512", jwa.HS512},
		{"RS256", jwa.RS256},
		{"RS384", jwa.RS384},
		{"RS512", jwa.RS512},
		{"PS256", jwa.PS256},
		{"PS384", jwa.PS384},
		{"PS512", jwa.PS512},
		{"ES256", jwa.ES256},
		{"ES384", jwa.ES384},
		{"ES512", jwa.ES512},
	}

	for _, tc := range algorithms {
		// Create a key to export for import testing
		key, err := New(tc.alg)
		if err != nil {
			b.Fatalf("failed to create key for validation test: %v", err)
		}
		record := key.Export(false)

		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				imported, err := Import(record)
				if err != nil {
					b.Fatalf("import failed: %v", err)
				}
				err = imported.Validate()
				if err != nil {
					b.Fatalf("validation failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkHMACKeySizes measures HMAC key creation performance
// across different key sizes.
func BenchmarkHMACKeySizes(b *testing.B) {
	sizes := []struct {
		name string
		alg  jwa.Alg
	}{
		{"HS256", jwa.HS256},
		{"HS384", jwa.HS384},
		{"HS512", jwa.HS512},
	}

	for _, tc := range sizes {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := New(tc.alg)
				if err != nil {
					b.Fatalf("failed to create HMAC key: %v", err)
				}
			}
		})
	}
}

// BenchmarkECKeyCurves measures EC key creation performance
// across different curves.
func BenchmarkECKeyCurves(b *testing.B) {
	curves := []struct {
		name string
		alg  jwa.Alg
	}{
		{"ES256-P256", jwa.ES256},
		{"ES384-P384", jwa.ES384},
		{"ES512-P521", jwa.ES512},
	}

	for _, tc := range curves {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := New(tc.alg)
				if err != nil {
					b.Fatalf("failed to create EC key: %v", err)
				}
			}
		})
	}
}

// BenchmarkRSAKeyCreation measures RSA key creation performance.
func BenchmarkRSAKeyCreation(b *testing.B) {
	// Standard RSA key sizes used in JWK
	b.Run("2048-bit", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := New(jwa.RS256)
			if err != nil {
				b.Fatalf("failed to create RSA key: %v", err)
			}
		}
	})
}

// BenchmarkRecordValidation measures the performance of validating
// JWK Record structures directly.
func BenchmarkRecordValidation(b *testing.B) {
	// Create a valid record for testing
	key, err := New(jwa.ES256)
	if err != nil {
		b.Fatalf("failed to create key: %v", err)
	}
	record := key.Export(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := record.Validate()
		if err != nil {
			b.Fatalf("record validation failed: %v", err)
		}
	}
}
