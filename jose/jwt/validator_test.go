package jwt

import (
	"testing"

	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/jose/jwa"
	"go.bryk.io/pkg/jose/jwk"
)

func TestValidator(t *testing.T) {
	assert := tdd.New(t)

	// Create (or import) a single master key
	mk, _ := jwk.New(jwa.ES256)
	mk.SetID("master-key")

	// Create a new generator instance
	tg, err := NewGenerator("acme.com")
	assert.Nil(err, "new generator")
	assert.Nil(tg.AddKey(mk), "add key")

	// Generator issue a token
	params := TokenParameters{
		Method:      string(jwa.ES256),
		Subject:     "Rick Sanchez",
		Audience:    []string{"https://bryk.io"},
		NotBefore:   "0ms",
		ContentType: "sample/token",
	}
	token, err := tg.Issue("master-key", &params)
	assert.Nil(err, "new token")
	t.Log(token)

	// Generate publishes its JWK set
	tgSet := tg.ExportKeys(true) // no private keys

	// Validator is an external process with access only to public keys
	val, err := NewValidator(WithValidationKeys(tgSet))
	assert.Nil(err, "new validator")

	// Use the validator to validate a token
	valChecks := params.GetChecks()
	valChecks = append(valChecks, IssuerCheck("acme.com"))
	assert.Nil(val.Validate(token.String(), valChecks...), "validate failed")
}
