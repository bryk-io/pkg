package jwk

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/jose/jwa"
)

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

		// Marshal
		_, err = k.MarshalBinary()
		assert.Nil(err, "marshal")

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
