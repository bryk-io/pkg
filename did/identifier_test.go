package did

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/crypto/ed25519"
)

type sampleExtensionData struct {
	UUID  string `json:"uuid"`
	Stamp int64  `json:"stamp"`
	Agent string `json:"agent"`
}

func encode(id *Identifier) []byte {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	_ = enc.Encode(id.Document(false))
	return buf.Bytes()
}

func decode(data []byte) (*Identifier, error) {
	doc := &Document{}
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(doc); err != nil {
		return nil, err
	}
	return FromDocument(doc)
}

func TestRegisterContext(t *testing.T) {
	assert := tdd.New(t)
	id, err := NewIdentifierWithMode("bryk", "sample-network", ModeUUID)
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	// Add a couple of keys
	assert.Nil(id.AddNewVerificationMethod("key-1", KeyTypeEd), "add key error")
	id.AddService(&ServiceEndpoint{
		ID:       "my-service",
		Type:     "acme-service",
		Endpoint: "https://acme.com/my-service",
		Extensions: []Extension{
			{
				ID:      "custom.extension",
				Version: "0.1.0",
				Data: map[string]string{
					"address": "Q4HSY6GM7AJGVSZGWPI5NZW2TJ4SIFHPSBXG4MCL72B5DAJL3PCCXIE3HI",
					"asset":   "ALGO",
					"network": "testnet",
				},
			},
		},
	})

	// Register custom context
	// extContext := make(map[string]interface{})
	// _ = json.Unmarshal([]byte(extV1), &extContext)
	id.RegisterContext(extV1Context)

	// JSON encode/decode
	doc := id.Document(true)
	js, err := json.MarshalIndent(doc, "", "  ")
	assert.Nil(err)
	assert.NotZero(len(js))
	// _ = ioutil.WriteFile("testdata/sample.json", js, 0644)

	// Restore id from document
	id2, err := FromDocument(doc)
	assert.Nil(err, "restore from document failed")
	doc2 := id2.Document(true)
	assert.Equal(doc, doc2, "invalid document contents")

	_, err = doc.NormalizedLD()
	assert.Nil(err, "normalized doc")
	_, err = doc.ExpandedLD()
	assert.Nil(err, "expanded doc")
}

func TestIdentifier(t *testing.T) {
	assert := tdd.New(t)

	t.Run("Parse", func(t *testing.T) {
		// Test parser with a fully loaded DID
		// https://w3c.github.io/did-core/#example-a-resource-external-to-a-did-document
		example := "did:example:123/custom/path?service=agent&relativeRef=/credentials#degree"
		id, err := Parse(example)
		assert.Nil(err, "failed to parse identifier")

		// Verify path contents
		assert.Equal("/custom/path", id.Path())
		assert.Equal(2, len(id.data.PathSegments))
		assert.True(id.IsURL())

		// Verify fragment value
		assert.Equal("#degree", id.Fragment())

		// Verify query parameters
		q, err := id.Query()
		assert.Nil(err, "failed to parse query")
		assert.Equal("service=agent&relativeRef=/credentials", id.RawQuery())
		assert.Equal("agent", q.Get("service"))
		assert.Equal("/credentials", q.Get("relativeRef"))

		// Verify string representation
		assert.Equal(example, id.String())
	})

	t.Run("Verify", func(t *testing.T) {
		seed := uuid.New()
		d, _ := NewIdentifier("bryk", seed.String())

		// Check the id string is a valid UUID
		customVerifier := func(s string) error {
			_, e := uuid.Parse(s)
			return e
		}
		assert.Nil(d.Verify(customVerifier), "verify error")
	})

	t.Run("Accessors", func(t *testing.T) {
		d, err := Parse("did:bryk:foo/rick/sanchez?variable=value&sample=test#c137")
		assert.Nil(err, "parse error")
		assert.Equal("bryk", d.Method(), "invalid method")
		assert.Equal("did:bryk:foo", d.DID(), "invalid DID")
		assert.Equal("/rick/sanchez", d.Path(), "invalid path")
		assert.Equal("#c137", d.Fragment(), "invalid fragment")
		assert.Equal("variable=value&sample=test", d.RawQuery(), "invalid raw query")

		q, err := d.Query()
		assert.Nil(err, "failed to retrieve query")
		assert.Equal("value", q.Get("variable"), "invalid query variable")
		assert.Equal("test", q.Get("sample"), "invalid query variable")
	})

	t.Run("New", func(*testing.T) {
		idString := uuid.New()
		_, err := NewIdentifier("", idString.String())
		assert.NotNil(err, "failed to catch missing method")

		for i := 0; i <= 100; i++ {
			d, _ := NewIdentifier("bryk", idString.String())
			_, err = Parse(d.String())
			assert.Nil(err, "invalid identifier produced")
		}
	})

	t.Run("Serialization", func(t *testing.T) {
		// Create a new identifier instance from a string
		id, err := Parse("did:example:q7ckgxeq1lxmra0r")
		assert.Nil(err, "parse error")

		// Add a new key
		assert.Nil(id.AddNewVerificationMethod("key-1", KeyTypeEd), "add key error")

		// Encode
		bin := encode(id)

		// Restore
		id2, err := decode(bin)
		assert.Nil(err, "decode error")
		assert.Equal(id.data.VerificationMethods[0].Private, id2.data.VerificationMethods[0].Private, "invalid data restored")
	})
}

func TestVerificationMethods(t *testing.T) {
	assert := tdd.New(t)

	// New DID with master key
	id, err := NewIdentifierWithMode("bryk", "sample-network", ModeUUID)
	assert.Nil(err, "new identifier")
	assert.Nil(id.AddNewVerificationMethod("master", KeyTypeEd), "add key")

	// Add verification methods
	key := id.GetReference("master")
	assert.Nil(id.AddVerificationRelationship(key, AuthenticationVM), "authentication")
	assert.Nil(id.AddVerificationRelationship(key, AssertionVM), "assertion")
	assert.Nil(id.AddVerificationRelationship(key, KeyAgreementVM), "key agreement")
	assert.Nil(id.AddVerificationRelationship(key, CapabilityInvocationVM), "invocation")
	assert.Nil(id.AddVerificationRelationship(key, CapabilityDelegationVM), "delegation")

	// Retrieve verification methods
	assert.Equal(1, len(id.GetVerificationRelationship(AuthenticationVM)), "authentication")
	assert.Equal(1, len(id.GetVerificationRelationship(AssertionVM)), "assertion")
	assert.Equal(1, len(id.GetVerificationRelationship(KeyAgreementVM)), "key agreement")
	assert.Equal(1, len(id.GetVerificationRelationship(CapabilityInvocationVM)), "invocation")
	assert.Equal(1, len(id.GetVerificationRelationship(CapabilityDelegationVM)), "delegation")

	// Verify proof
	mk := id.VerificationMethod("master")
	data, err := id.Document(true).NormalizedLD()
	assert.Nil(err, "normalized DID document")
	proof, err := id.GetProof(mk.ID, "did.bryk.io")
	assert.Nil(err, "get proof")
	assert.True(mk.VerifyProof(data, proof), "verify proof")
}

func TestDocument(t *testing.T) {
	assert := tdd.New(t)

	// Document instance from existing identifier
	id, err := NewIdentifierWithMode("bryk", "sample-network", ModeUUID)
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	// Add a couple of keys
	assert.Nil(id.AddNewVerificationMethod("key-1", KeyTypeEd), "add key error")
	assert.Nil(id.AddNewVerificationMethod("key-2", KeyTypeRSA), "add key error")
	assert.Nil(id.AddNewVerificationMethod("koblitz", KeyTypeSecp256k1), "add key error")
	assert.Nil(id.AddNewVerificationMethod("new-encoding", KeyTypeEd), "add key error")
	assert.NotNil(id.AddNewVerificationMethod("key-1", KeyTypeEd), "duplicated key id")

	t.Run("KeyExtensions", func(t *testing.T) {
		// Sample extension
		ext := Extension{
			ID:      "org.sample.extension",
			Version: "0.1.0",
			Data: sampleExtensionData{
				UUID:  uuid.New().String(),
				Agent: "https://something.com/foo-123",
				Stamp: time.Now().Unix(),
			},
		}

		// New key with extension data
		_ = id.AddNewVerificationMethod("managed", KeyTypeEd)
		mk := id.VerificationMethod("managed")
		mk.AddExtension(ext)

		// Validations
		d2 := sampleExtensionData{}
		err = mk.GetExtension(ext.ID, "0.2.0", &d2)
		assert.NotNil(err, "failed to catch missing extension")
		err = mk.GetExtension(ext.ID, ext.Version, &d2)
		assert.Nil(err, "failed to decode extension data")
		assert.Equal(ext.Data, d2, "wrong extension data")

		// Clean up
		_ = id.RemoveVerificationMethod("managed")
	})

	t.Run("AuthenticationKeys", func(t *testing.T) {
		// Authentication keys
		assert.Nil(id.AddVerificationRelationship(id.GetReference("key-1"), AuthenticationVM), "add valid key")
		assert.Nil(id.AddVerificationRelationship(id.GetReference("key-2"), AuthenticationVM), "add valid key")
		assert.NotNil(id.AddVerificationRelationship(id.GetReference("key-1"), AuthenticationVM), "duplicate key")
		assert.NotNil(id.RemoveVerificationRelationship(id.GetReference("key-3"), AuthenticationVM), "invalid key")
		assert.Nil(id.RemoveVerificationRelationship(id.GetReference("key-2"), AuthenticationVM), "remove key")
	})

	t.Run("ServiceEndpoints", func(t *testing.T) {
		// Sample extension
		ext := Extension{
			ID:      "org.sample.extension",
			Version: "0.1.0",
			Data: sampleExtensionData{
				UUID:  uuid.New().String(),
				Agent: "https://something.com/foo-123",
				Stamp: time.Now().Unix(),
			},
		}

		// Service endpoints
		se := &ServiceEndpoint{
			ID:       "cherami",
			Type:     "SecureMessaging_1.0",
			Endpoint: "https://cherami.io/rick/sanchez",
		}
		se.AddExtension(ext)

		assert.Nil(id.AddService(se), "add service error")
		assert.NotNil(id.AddService(se), "duplicated service")
		assert.Nil(id.Service("invalid-name"), "failed to catch invalid service id")
		assert.NotNil(id.Service("cherami"), "failed to retrieve valid service id")

		// Validate extensions
		data := sampleExtensionData{}
		svc := id.Service("cherami")
		err = svc.GetExtension(ext.ID, "0.2.0", &data)
		assert.NotNil(err, "failed to catch missing extension")
		err = svc.GetExtension(ext.ID, ext.Version, &data)
		assert.Nil(err, "failed to decode extension data")
		assert.Equal(ext.Data, data, "wrong extension data")

		// Cleanup
		assert.NotNil(id.RemoveService("invalid-name"), "not registered service")
		assert.Nil(id.RemoveService("cherami"), "remove service error")
		assert.Equal(0, len(id.data.Services), "invalid service count")
	})

	t.Run("JSON", func(t *testing.T) {
		// JSON encode/decode
		d1 := id.Document(false)
		js, err := json.MarshalIndent(d1, "", "  ")
		assert.Nil(err, "json encode error")

		d2 := &Document{}
		assert.Nil(json.Unmarshal(js, d2), "json decode error")

		// Verify id value after restore
		assert.Equal(d1.Subject, d2.Subject, "invalid identifier value")
		// log.Printf("%s", js)
	})

	t.Run("LD", func(t *testing.T) {
		// Produce expanded LD representation
		_, err = id.Document(true).ExpandedLD()
		assert.Nil(err, "expanded LD error")

		// Signatures, use normalized DID document as data
		_, err = id.Document(true).NormalizedLD()
		assert.Nil(err, "normalized LD error")
	})

	t.Run("Signatures", func(t *testing.T) {
		// Signatures use normalized DID document as data
		data, _ := id.Document(true).NormalizedLD()

		t.Run("Ed25519", func(t *testing.T) {
			// Sign and verify using a Ed25519 key
			s1, err := id.VerificationMethod("key-1").Sign(data)
			assert.Nil(err, "sign error")
			assert.True(id.VerificationMethod("key-1").Verify(data, s1), "verify error")
		})

		t.Run("RSA", func(t *testing.T) {
			// Sign and verify using an RSA key
			s2, err := id.VerificationMethod("key-2").Sign(data)
			assert.Nil(err, "sign error")
			assert.True(id.VerificationMethod("key-2").Verify(data, s2), "verify error")
		})

		t.Run("secp256k1", func(t *testing.T) {
			s3, err := id.VerificationMethod("koblitz").Sign(data)
			assert.Nil(err, "sign error")
			assert.True(id.VerificationMethod("koblitz").Verify(data, s3), "verify error")
		})
	})

	t.Run("SignaturesLD", func(t *testing.T) {
		// Signatures use normalized DID document as data
		data, _ := id.Document(true).NormalizedLD()

		t.Run("Ed25519", func(t *testing.T) {
			// Sign and verify using a Ed25519 key
			s1, err := id.VerificationMethod("key-1").ProduceSignatureLD(data, "test-domain-value")
			assert.Nil(err, "sign error")
			assert.True(id.VerificationMethod("key-1").VerifySignatureLD(data, s1), "verify error")
		})

		t.Run("RSA", func(t *testing.T) {
			// Sign and verify using an RSA key
			s2, err := id.VerificationMethod("key-2").ProduceSignatureLD(data, "test-domain-value")
			assert.Nil(err, "sign error")
			assert.True(id.VerificationMethod("key-2").VerifySignatureLD(data, s2), "verify error")
		})

		t.Run("secp256k1", func(t *testing.T) {
			// Sign and verify using a secp256k1 key
			s2, err := id.VerificationMethod("koblitz").ProduceSignatureLD(data, "test-domain-value")
			assert.Nil(err)
			assert.True(id.VerificationMethod("koblitz").VerifySignatureLD(data, s2), "verify error")
		})
	})

	t.Run("ProofLD", func(t *testing.T) {
		// Proofs will use normalized DID document as data
		data, _ := id.Document(true).NormalizedLD()

		// Retrieve invalid key
		assert.Nil(id.VerificationMethod("invalid-key"), "fetch invalid key id")

		// Produce and verify proof
		pk := id.VerificationMethod("key-1")
		assert.NotNil(pk, "failed to retrieve key")
		p1, err := pk.ProduceProof(data, "authentication", "test-domain-value")
		assert.Nil(err, "produce proof error")
		assert.True(pk.VerifyProof(data, p1), "verify proof error")
	})

	t.Run("Serialization", func(t *testing.T) {
		bin := encode(id)
		id2, err := decode(bin)
		assert.Nil(err, "decode error")
		b2 := encode(id2)
		assert.Equal(bin, b2, "unexpected re-encoding value")
	})

	t.Run("AddVerificationMethod", func(t *testing.T) {
		k1, _ := ed25519.New()
		assert.NotNil(id.AddVerificationMethod("existing-ed-key", []byte("not a private key"), KeyTypeEd),
			"failed to catch invalid key value")
		assert.NotNil(id.AddVerificationMethod("existing-ed-key", k1.PrivateKey(), KeyTypeRSA),
			"failed to catch invalid key type")
		assert.Nil(id.AddVerificationMethod("existing-ed-key", k1.PrivateKey(), KeyTypeEd),
			"failed to add valid new key")

		_, k2, _ := newRSAKey()
		assert.NotNil(id.AddVerificationMethod("existing-rsa-key", []byte("not a private key"), KeyTypeRSA),
			"failed to catch invalid key value")
		assert.NotNil(id.AddVerificationMethod("existing-rsa-key", k2, KeyTypeEd),
			"failed to catch invalid key type")
		assert.Nil(id.AddVerificationMethod("existing-rsa-key", k2, KeyTypeRSA),
			"failed to add valid new key")
	})

	t.Run("RemoveVerificationMethod", func(t *testing.T) {
		alice, _ := NewIdentifierWithMode("bryk", "sample-network", ModeUUID)
		assert.Nil(alice.AddNewVerificationMethod("master", KeyTypeEd), "add key error")
		assert.Nil(alice.AddVerificationRelationship(alice.GetReference("master"), AuthenticationVM), "add auth key error")
		assert.NotNil(alice.RemoveVerificationMethod("invalid-name"), "invalid key id")
		assert.NotNil(alice.RemoveVerificationMethod("master"), "should not be able to remove only authentication key")
		assert.Nil(alice.AddNewVerificationMethod("master-replacement", KeyTypeEd), "add key error")
		assert.Nil(alice.AddVerificationRelationship(alice.GetReference("master-replacement"), AuthenticationVM), "add auth key error")
		assert.Nil(alice.RemoveVerificationMethod("master"), "remove key error")
		assert.Equal(1, len(alice.VerificationMethods()), "invalid keys count")
	})

	t.Run("DateAccessors", func(t *testing.T) {
		_, err = id.Created()
		assert.Nil(err, "failed to get created date")
		_, err = id.Updated()
		assert.Nil(err, "failed to get updated date")
	})
}

func TestFromDocument(t *testing.T) {
	assert := tdd.New(t)

	// Sample extension
	ext := Extension{
		ID:      "org.sample.extension",
		Version: "0.1.0",
		Data: sampleExtensionData{
			UUID:  uuid.New().String(),
			Agent: "https://something.com/foo-123",
			Stamp: time.Now().Unix(),
		},
	}

	// Generate original DID and its document
	id, _ := NewIdentifierWithMode("bryk", "", ModeUUID)
	_ = id.AddNewVerificationMethod("master", KeyTypeEd)
	_ = id.AddNewVerificationMethod("key-2", KeyTypeEd)
	_ = id.AddVerificationRelationship(fmt.Sprintf("%s#%s", id, "master"), AuthenticationVM)
	_ = id.AddVerificationRelationship(fmt.Sprintf("%s#%s", id, "key-2"), AuthenticationVM)
	id.VerificationMethod("key-2").AddExtension(ext)
	js, _ := json.MarshalIndent(id.Document(true), "", "  ")

	// Get new document instance from JSON contents
	doc := &Document{}
	assert.Nil(json.Unmarshal(js, doc), "json decode error")

	// Restore new DID instance from decoded document
	id2, err := FromDocument(doc)
	assert.Nil(err, "from document error")

	// Retrieve extension data
	data := sampleExtensionData{}
	err = id2.VerificationMethod("key-2").GetExtension(ext.ID, ext.Version, &data)
	assert.Nil(err, "failed to retrieve extension data")
	assert.Equal(ext.Data, data, "failed to retrieve extension data")

	// Verification round
	sig, err := id.VerificationMethod("master").ProduceSignatureLD([]byte("foo"), "")
	assert.Nil(err, "sign error")
	assert.True(id2.VerificationMethod("master").VerifySignatureLD([]byte("foo"), sig), "verify error")
	_, err = id2.VerificationMethod("master").Sign([]byte("something here"))
	assert.NotNil(err, "signing without private key present")
}

// Basic identifier generation.
func ExampleNewIdentifier() {
	id, err := NewIdentifier("sample", "foo-bar")
	if err != nil {
		panic(err)
	}
	fmt.Println(id)
	// Output: did:sample:foo-bar
}

// Basic identifier generation for specific method, tag and mode.
func ExampleNewIdentifierWithMode() {
	id, err := NewIdentifierWithMode("bryk", "c137", ModeUUID)
	if err != nil {
		panic(err)
	}
	fmt.Println(id)
}

// Generate DID document from identifier instance.
func ExampleIdentifier_Document() {
	// Create a new identifier instance
	id, err := NewIdentifierWithMode("bryk", "c137", ModeUUID)
	if err != nil {
		panic(err)
	}

	// Add a new key and enable it as authentication mechanism
	_ = id.AddNewVerificationMethod("master", KeyTypeEd)
	_ = id.AddVerificationRelationship(id.GetReference("master"), AuthenticationVM)

	// Get cryptographic proof for the identifier instance
	key := id.VerificationMethod("master")
	proof, err := id.GetProof(key.ID, "sample.com")
	if err != nil {
		panic(err)
	}

	// The verifier can later check the validity of the proof using the
	// public key and normalized version of the DID document.
	doc, _ := id.Document(true).NormalizedLD()
	if !key.VerifyProof(doc, proof) {
		panic("invalid proof")
	}

	// Print DID document in JSON format
	js, _ := json.MarshalIndent(id.Document(true), "", "  ")
	fmt.Printf("%s", js)
}

// Produce and verify singed messages.
func ExamplePublicKey_Sign() {
	id, _ := NewIdentifierWithMode("bryk", "", ModeUUID)
	_ = id.AddNewVerificationMethod("master", KeyTypeEd)
	_ = id.AddVerificationRelationship(id.GetReference("master"), AuthenticationVM)

	// Get master key
	masterKey := id.VerificationMethod("master")
	msg := []byte("original message to sign")

	// Get binary message signature
	signatureBin, _ := masterKey.Sign(msg)
	if !masterKey.Verify(msg, signatureBin) {
		panic("failed to verify binary signature")
	}

	// Get a JSON-LD message signature
	signatureJSON, _ := masterKey.ProduceSignatureLD(msg, "example.com")
	if !masterKey.VerifySignatureLD(msg, signatureJSON) {
		panic("failed to verify JSON-LD signature")
	}
}
