package jwt

import (
	"strings"
	"testing"
	"time"

	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/jose/jwa"
	"go.bryk.io/pkg/jose/jwk"
)

func TestNewGenerator(t *testing.T) {
	assert := tdd.New(t)

	// New generator instance with support for all standard methods
	tg, err := NewGenerator("acme.com", WithSupportForNone())
	assert.Nil(err, "new generator")
	for _, m := range standardMethods() {
		if m == jwa.NONE {
			continue
		}
		k, _ := jwk.New(m)
		k.SetID(string(m)) // use the `alg` identifier as keu ID
		assert.Nil(tg.AddKey(k), "add key", m)
	}

	t.Run("IsSupported", func(t *testing.T) {
		for _, m := range standardMethods() {
			assert.True(tg.IsSupported(m), "supported method")
		}
		assert.False(tg.IsSupported("dummy"), "invalid method")
	})

	t.Run("Issue", func(t *testing.T) {
		params := TokenParameters{
			Subject:     "Rick Sanchez",
			Audience:    []string{"https://bryk.io"},
			NotBefore:   "0ms",
			ContentType: "sample/token",
		}

		for _, m := range standardMethods() {
			mtd := string(m)
			t.Run(mtd, func(t *testing.T) {
				// Set method
				params.Method = mtd
				assert.Nil(params.verify(), "invalid parameters")

				// Generate token
				token, err := tg.Issue(mtd, &params)
				assert.Nil(err, "new token", m)
				t.Log(token)

				// Retrieve single token claim
				issuerValue, err := token.Get("/iss")
				assert.Nil(err, "get issuer")
				assert.Equal("acme.com", issuerValue)

				t.Run("Validate", func(t *testing.T) {
					// Validate token
					assert.Nil(tg.Validate(token.String(), params.GetChecks()...), "validate")
					assert.Equal(2, strings.Count(token.String(), "."), "invalid string structure")
					assert.True(len(token.Bytes()) > 0, "invalid binary contents")
				})
			})
		}
	})

	t.Run("Validate", func(t *testing.T) {
		params := TokenParameters{
			Subject:          "Rick Sanchez",
			Audience:         []string{"https://bryk.io"},
			NotBefore:        "0ms",
			Method:           string(jwa.HS256),
			Expiration:       "1h",
			ContentType:      "custom-credential/0.1",
			UniqueIdentifier: "foo-bar",
		}
		token, err := tg.Issue(string(jwa.HS256), &params)
		assert.Nil(err, "new token error")

		// sub
		assert.NotNil(token.Validate(SubjectCheck("morty")), "invalid subject")

		// iss
		assert.NotNil(token.Validate(IssuerCheck("invalid")), "invalid issuer")

		// aud
		assert.NotNil(token.Validate(AudienceCheck([]string{"foo.bar"})), "invalid audience")

		// jti
		assert.NotNil(token.Validate(IDCheck("another")), "invalid unique identifier")

		// iat
		assert.NotNil(token.Validate(IssuedAtCheck(time.Now().Add(-1*time.Hour))), "invalid issued date")

		// exp
		assert.NotNil(token.Validate(ExpirationTimeCheck(time.Now().Add(2*time.Hour), true)),
			"invalid expiration date")

		// nbf
		assert.NotNil(token.Validate(NotBeforeCheck(time.Now().Add(-1*time.Hour))), "invalid not before date")
	})

	t.Run("CustomClaims", func(t *testing.T) {
		params := TokenParameters{
			Subject:     "Rick Sanchez",
			Audience:    []string{"https://bryk.io"},
			NotBefore:   "0ms",
			ContentType: "sample/token",
			CustomClaims: &customData{
				Username: "rick",
				Email:    "rick@c137.mv",
				Metadata: nestedValue{
					Name:  "foo",
					Value: 7,
				},
			},
		}

		for _, m := range standardMethods() {
			mtd := string(m)
			t.Run(mtd, func(t *testing.T) {
				// Set method
				params.Method = mtd
				assert.Nil(params.verify(), "invalid parameters")

				// Generate token
				token, err := tg.Issue(mtd, &params)
				assert.Nil(err, "new token error", m)

				// Validate token
				assert.Nil(tg.Validate(token.String(), params.GetChecks()...), "validate")
				assert.Equal(2, strings.Count(token.String(), "."), "invalid string structure")
				assert.True(len(token.Bytes()) > 0, "invalid binary contents")

				// Retrieve (nested) token claims
				metadataName, err := token.Get("/metadata/name")
				assert.Nil(err, "get nested claim")
				assert.Equal("foo", metadataName)

				// Decode custom data
				data := &customData{}
				assert.Nil(token.Decode(data), "decode error")
				assert.Equal(params.CustomClaims, data, "custom data")
				t.Logf("%+v", data)
			})
		}
	})
}

func standardMethods() []jwa.Alg {
	return []jwa.Alg{
		jwa.NONE,
		jwa.HS256,
		jwa.HS384,
		jwa.HS512,
		jwa.ES256,
		jwa.ES384,
		jwa.ES512,
		jwa.RS256,
		jwa.RS384,
		jwa.RS512,
		jwa.PS256,
		jwa.PS384,
		jwa.PS512,
	}
}

type customData struct {
	Username string      `json:"username,omitempty"`
	Email    string      `json:"email,omitempty"`
	Metadata nestedValue `json:"metadata,omitempty"`
}

type nestedValue struct {
	Name  string `json:"name,omitempty"`
	Value int    `json:"value,omitempty"`
}
