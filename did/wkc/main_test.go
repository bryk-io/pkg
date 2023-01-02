package wkc

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/did"
)

func TestConfig(t *testing.T) {
	assert := tdd.New(t)

	// Register a 'jwt' key with a new or existing DID
	id, _ := did.NewIdentifier("bryk", uuid.NewString())
	assert.Nil(RegisterKey(id, "did-jwt-wkc"), "register key")

	// Generate as many domain links as required
	dom1, err := GenerateDomainLink(id, "did-jwt-wkc", "acme.com")
	assert.Nil(err, "generate domain link")
	dom2, err := GenerateDomainLink(id, "did-jwt-wkc", "cool-product.com")
	assert.Nil(err, "generate domain link")

	// Generate the "well known configuration" block
	conf := new(Configuration)
	conf.Entries = []*DomainLink{
		dom1,
		dom2,
	}

	// Configuration is commonly exposed as a JSON document
	js, _ := json.MarshalIndent(conf, "", "  ")
	t.Logf("%s\n", js)

	// Domains links can also be validated
	assert.Nil(ValidateDomainLink(id, dom1, "acme.com"), "invalid domain link")
}
