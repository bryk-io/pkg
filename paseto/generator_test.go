package paseto

import (
	"crypto/elliptic"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"testing"

	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/crypto/ed25519"
	"go.bryk.io/pkg/errors"
	cryptoutils "go.bryk.io/pkg/internal/crypto"
)

type customFooter struct {
	Dimension string `json:"dimension,omitempty"`
	KeyID     string `json:"kid,omitempty"`
}

type customPayload struct {
	Foo     string `json:"foo,omitempty"`
	Subject string `json:"sub,omitempty"`
}

// Official test vector file format
// https://github.com/paseto-standard/test-vectors
type testFile struct {
	Name  string       `json:"name"`
	Tests []testVector `json:"tests"`
}

type testVector struct {
	Name         string `json:"name"`
	ShouldFail   bool   `json:"expect-fail"`
	Nonce        string `json:"nonce"`
	Key          string `json:"key"`
	KeySeed      string `json:"secret-key-seed"`
	PublicKey    string `json:"public-key"`
	SecretKey    string `json:"secret-key"`
	SecretKeyPEM string `json:"secret-key-pem"`
	Token        string `json:"token"`
	Payload      string `json:"payload"`
	Footer       string `json:"footer"`
	Assertion    string `json:"implicit-assertion"`
}

func edKeyRestore(seed string) (*ed25519.KeyPair, error) {
	skB, err := hex.DecodeString(seed)
	if err != nil {
		return nil, err
	}
	return ed25519.FromPrivateKey(skB)
}

func loadTestFile(name string) (*testFile, error) {
	// Open test file
	file, err := os.ReadFile("testdata/" + name)
	if err != nil {
		return nil, err
	}

	// Decode test file
	tf := new(testFile)
	err = json.Unmarshal(file, &tf)
	return tf, err
}

func runTestFile(t *testing.T, tf *testFile, gen *Generator) {
	t.Run(tf.Name, func(t *testing.T) {
		for _, vt := range tf.Tests {
			t.Run(vt.Name, func(t *testing.T) {
				assert := tdd.New(t)

				// Parse token
				token, err := Parse(vt.Token)
				assert.Nil(err, "parse token")

				// No more tests required
				if vt.ShouldFail {
					return
				}

				// Decode key used
				var sk Key
				switch ProtocolVersion(token.Header()) {
				case V1P:
					key := new(rsaKey)
					key.SetID("test-key")
					err = key.UnmarshalBinary([]byte(vt.SecretKey))
					assert.Nil(err, "unmarshal RSA key")
					sk = key
				case V2P:
					key := new(edKey)
					key.kp, err = edKeyRestore(vt.SecretKey)
					assert.Nil(err, "unmarshal ED25519 key")
					sk = key
				case V3P:
					block, _ := pem.Decode([]byte(vt.SecretKeyPEM))
					priv, err := x509.ParseECPrivateKey(block.Bytes)
					assert.Nil(err, "invalid private key")
					key := new(ecdsaKey)
					key.sk = priv
					key.crv = elliptic.P384()
					sk = key
				case V4P:
					key := new(edKey)
					key.kp, err = edKeyRestore(vt.SecretKey)
					assert.Nil(err, "unmarshal ED25519 key")
					sk = key
				default:
					sec, err := hex.DecodeString(vt.Key)
					assert.Nil(err, "decode key")
					key := new(hmacKey)
					key.sk = sec
					sk = key
				}

				// manually unseal token
				if sk != nil {
					ia := []byte(vt.Assertion)
					token.pld, err = gen.unseal(token, sk, ia)
					assert.Nil(err)
				}

				// Decode payload
				body := map[string]interface{}{}
				err = token.DecodePayload(&body)
				assert.Nil(err, "decode payload")
				assert.NotEmpty(body["data"])

				// Get registered claims
				_, err = token.RegisteredClaims()
				assert.Nil(err, "registered claims")

				// Decode footer
				if strings.Contains(vt.Footer, "{") {
					footer := map[string]interface{}{}
					err = token.DecodeFooter(&footer)
					assert.Nil(err, "decode footer")
					assert.NotEmpty(footer["kid"])
				}
			})
		}
	})
}

func sampleGenerator(name string) *Generator {
	g := NewGenerator(name)

	r1, _ := NewKey("v1.public", V1P)
	_ = g.AddKey(r1)

	h1, _ := NewKey("v1.local", V1L)
	_ = g.AddKey(h1)

	e1, _ := NewKey("v2.public", V2P)
	_ = g.AddKey(e1)

	h2, _ := NewKey("v2.local", V2L)
	_ = g.AddKey(h2)

	p1, _ := NewKey("v3.public", V3P)
	_ = g.AddKey(p1)

	h3, _ := NewKey("v3.local", V3L)
	_ = g.AddKey(h3)

	e2, _ := NewKey("v4.public", V4P)
	_ = g.AddKey(e2)

	h4, _ := NewKey("v4.local", V4L)
	_ = g.AddKey(h4)

	return g
}

func BenchmarkGenerator_NewToken(b *testing.B) {
	// Generator
	gen := sampleGenerator("acme.com")

	// Base parameters
	params := TokenParameters{
		CustomClaims: map[string]string{"foo": "bar"},
		Footer:       customFooter{Dimension: "c137"},
	}

	// Benchmarks table
	list := []struct {
		ver string
		pps string
		key string
	}{
		{
			ver: "v1",
			pps: pLocal,
			key: "sample-hmac",
		},
		{
			ver: "v1",
			pps: pPublic,
			key: "sample-rsa",
		},
		{
			ver: "v2",
			pps: pLocal,
			key: "sample-hmac",
		},
		{
			ver: "v2",
			pps: pPublic,
			key: "sample-ed",
		},
	}
	b.ResetTimer()
	for _, bb := range list {
		b.Run(fmt.Sprintf("%s.%s", bb.ver, bb.pps), func(b *testing.B) {
			params.Version = bb.ver
			params.Purpose = bb.pps
			b.StartTimer()
			for n := 0; n < b.N; n++ {
				_, _ = gen.Issue(bb.key, &params)
			}
			b.StopTimer()
		})
	}
}

func BenchmarkGenerator_Validate(b *testing.B) {
	// Generator
	gen := sampleGenerator("acme.com")

	// Base parameters
	params := TokenParameters{
		CustomClaims: map[string]string{"foo": "bar"},
		Footer:       customFooter{Dimension: "c137"},
	}

	// Benchmarks table
	list := []struct {
		ver string
		pps string
		key string
		tkn string
	}{
		{
			ver: "v1",
			pps: pLocal,
			key: "sample-hmac",
		},
		{
			ver: "v1",
			pps: pPublic,
			key: "sample-rsa",
		},
		{
			ver: "v2",
			pps: pLocal,
			key: "sample-hmac",
		},
		{
			ver: "v2",
			pps: pPublic,
			key: "sample-ed",
		},
	}
	b.ResetTimer()
	for _, bb := range list {
		b.Run(fmt.Sprintf("%s.%s", bb.ver, bb.pps), func(b *testing.B) {
			params.Version = bb.ver
			params.Purpose = bb.pps
			tkn, err := gen.Issue(bb.key, &params)
			if err != nil {
				b.Error(err)
			}
			bb.tkn = tkn.String()
			b.StartTimer()
			for n := 0; n < b.N; n++ {
				_ = gen.Validate(bb.tkn)
			}
			b.StopTimer()
		})
	}
}

func TestParse(t *testing.T) {
	assert := tdd.New(t)
	sample := "v1.local.PkNkqbzLzdx-7GMo-X2_dBNYHgUU1einrnxb8bgqmweTuuk5Utkt91TQ-sH_tPybEn17PIOiFdjvr9bAFXYEdQvHhDUNmSblMsS9Bzd17iR_jUXoIU1KG_LjH218.c29tZQ"
	tk, err := Parse(sample)
	assert.Nil(err, "failed to parse valid token")
	assert.Equal("v1", tk.Version(), "version code")
	assert.Equal(pLocal, tk.Purpose(), "purpose code")
	assert.Equal("c29tZQ", tk.Footer(), "token footer")
	assert.Equal([]byte("some"), tk.ftr, "footer contents")
}

func TestGenerator(t *testing.T) {
	assert := tdd.New(t)

	// Generator
	gen := sampleGenerator("my-cool-service.com")

	// Base parameters
	params := TokenParameters{
		Subject:              "my-user-id",               // user/client/customer id
		Audience:             []string{"my-service.com"}, // token realms
		Expiration:           "24h",                      // expires in 1 day
		NotBefore:            "0s",                       // instantly available
		AuthenticationMethod: []string{"pin", "face"},    // valid AMR values
		CustomClaims: customPayload{
			Foo:     "bar",
			Subject: "this-value-will-be-overwritten",
		},
		Footer: customFooter{
			Dimension: "c137",
			KeyID:     "this-value-will-be-overwritten",
		},
	}

	// Custom token validator.
	myValidator := func(t *Token) error {
		// Validate a field in the payload
		cp := customPayload{}
		if err := t.DecodePayload(&cp); err != nil {
			return err
		}
		if cp.Foo != "bar" {
			return errors.New("invalid 'foo' value")
		}

		// Validate a field in the footer
		cf := customFooter{}
		if err := t.DecodeFooter(&cf); err != nil {
			return err
		}
		if cf.Dimension != "c137" {
			return errors.New("invalid 'dimension' value")
		}
		return nil
	}

	// Tests table
	list := []struct {
		ver string
		pps string
	}{
		{
			ver: "v1",
			pps: pLocal,
		},
		{
			ver: "v1",
			pps: pPublic,
		},
		{
			ver: "v2",
			pps: pLocal,
		},
		{
			ver: "v2",
			pps: pPublic,
		},
		{
			ver: "v3",
			pps: pLocal,
		},
		{
			ver: "v3",
			pps: pPublic,
		},
		{
			ver: "v4",
			pps: pLocal,
		},
		{
			ver: "v4",
			pps: pPublic,
		},
	}
	for _, tt := range list {
		name := fmt.Sprintf("%s.%s", tt.ver, tt.pps)
		t.Run(name, func(t *testing.T) {
			// Retrieve key used for the operation
			keyID := fmt.Sprintf("%s.%s", tt.ver, tt.pps)
			ck, err := gen.GetKey(keyID)
			assert.Nil(err, "failed to retrieve key")

			// Create token
			params.Version = tt.ver
			params.Purpose = tt.pps
			token, err := gen.Issue(keyID, &params)
			assert.Nil(err, name)
			assert.Nil(gen.Validate(token.String()), "validate generated token")
			assert.Equal(tt.ver, token.Version())
			assert.Equal(tt.pps, token.Purpose())

			// Additional payload validations
			checks := []Check{myValidator}
			checks = append(checks, params.GetChecks()...)
			assert.Nil(gen.Validate(token.String(), checks...), "payload error")

			// Decode footer
			ftr := customFooter{}
			assert.Nil(token.DecodeFooter(&ftr), "decode footer")
			assert.Equal("c137", ftr.Dimension, "failed to decode private claim")
			assert.Equal(ck.ID(), ftr.KeyID, "failed to overwrite registered claim")

			// Decode registered claims
			assert.Nil(gen.Unseal(token), "unseal encrypted token")
			rc, err := token.RegisteredClaims()
			assert.Nil(err, "failed to access registered claims")
			assert.Equal(params.UniqueIdentifier, rc.JTI, "invalid JTI")
			assert.Equal("my-cool-service.com", rc.Issuer, "invalid issuer")

			// Decode payload
			cp := customPayload{}
			assert.Nil(token.DecodePayload(&cp), "decode payload")
			assert.Equal(params.Subject, cp.Subject)
			assert.Equal("bar", cp.Foo)
		})
	}
}

func TestExport(t *testing.T) {
	assert := tdd.New(t)

	list := []struct {
		proto ProtocolVersion
		equal bool
	}{
		{
			proto: V1L,
			equal: true,
		},
		{
			proto: V1P,
			equal: true,
		},
		{
			proto: V2L,
			equal: true,
		},
		{
			proto: V2P,
			equal: false,
		},
		{
			proto: V3L,
			equal: true,
		},
		{
			proto: V3P,
			equal: true,
		},
		{
			proto: V4L,
			equal: true,
		},
		{
			proto: V4P,
			equal: false,
		},
	}

	for _, tt := range list {
		t.Run(string(tt.proto), func(t *testing.T) {
			k1, _ := NewKey(cryptoutils.RandomID(), tt.proto)
			rec, err := k1.Export()
			assert.Nil(err, "export")
			js, _ := json.MarshalIndent(rec, "", "  ")
			t.Logf("\n%s\n", js)

			if tt.equal {
				k2, err := ImportKey(rec)
				assert.Nil(err)
				assert.Equal(k1, k2)
			} else {
				_, err = ImportKey(rec)
				assert.Nil(err)
			}
		})
	}
}

func TestVectors(t *testing.T) {
	assert := tdd.New(t)

	gen := NewGenerator("test-vectors")

	// V1
	v1f, err := loadTestFile("v1.json")
	assert.Nil(err, "load test file")
	runTestFile(t, v1f, gen)

	// V2
	v2f, err := loadTestFile("v2.json")
	assert.Nil(err, "load test file")
	runTestFile(t, v2f, gen)

	// V3
	v3f, err := loadTestFile("v3.json")
	assert.Nil(err, "load test file")
	runTestFile(t, v3f, gen)

	// V4
	v4f, err := loadTestFile("v4.json")
	assert.Nil(err, "load test file")
	runTestFile(t, v4f, gen)
}

func ExampleNewGenerator() {
	// -> Error checks omitted for brevity

	// Create a new generator and cryptographic keys
	gen := NewGenerator("my-service.com")
	k1, _ := NewKey("key-1", V1L)
	k2, _ := NewKey("key-2", V1P)
	k3, _ := NewKey("key-3", V2P)
	_ = gen.AddKey(k1, k2, k3)

	// Prepare a token request. These requests can, for example, be received in
	// JSON format from a different service or API.
	req := &TokenParameters{
		Version:              "v2",
		Purpose:              pPublic,
		Audience:             []string{"consumer-service.com"},
		Subject:              "my-user",               // Specify any value useful for your context
		UniqueIdentifier:     "",                      // A random UUID will be generated by default
		NotBefore:            "0s",                    // Will be valid immediately after creation
		Expiration:           "48h",                   // Will be valid for 2 days
		UseUTC:               true,                    // Dates will be set as UTC zone
		AuthenticationMethod: []string{"pin", "face"}, // AMR codes from RFC-8176 are supported
		CustomClaims: customPayload{
			Foo: "bar", // Payload can include any additional data required
		},
		Footer: customFooter{
			Dimension: "c137", // Footer can include any additional data required
		},
	}

	// Generate a new token based on the provided. Token instances implement
	// "Stringer" to easily use as text.
	token, _ := gen.Issue(k3.ID(), req)
	fmt.Println(token)

	// Additionally, the token instance API facilitate inspection and common tasks.
	fmt.Printf("version: %s\n", token.Version())
	fmt.Printf("purpose: %s\n", token.Purpose())
	fmt.Printf("footer: %s\n", token.Footer())

	// You can easily decode contents in the payload and footer to any generic or custom type.
	ftr := customFooter{}           // Custom type
	pld := map[string]interface{}{} // Generic map; registered claims will also be included
	_ = token.DecodeFooter(&ftr)
	_ = token.DecodePayload(&pld)
	fmt.Printf("custom footer value: %s\n", ftr.Dimension)
	fmt.Printf("payload: %+v\n", pld)

	// Commonly used validators (aud, jti, sub, amr, exp, nbf, iat) can be generated from a
	// `TokenParameters` instance.
	checks := req.GetChecks()

	// You can also provide you own custom validators.
	checks = append(checks, func(token *Token) error {
		// Decode and validate my custom contents
		pld := customPayload{}
		if err := token.DecodePayload(&pld); err != nil {
			return err
		}
		if pld.Foo != "bar" {
			return errors.New("invalid bar value")
		}
		return nil
	})

	// You can validate a token instance directly.
	if err := token.Validate(checks...); err != nil {
		fmt.Printf("invalid token: %s", err)
	}

	// Or you can validate it as a string using the generator instance. Validating the token using the
	// generator instance will also verify the signature, encryption and "iss" registered claim.
	if err := gen.Validate(token.String(), checks...); err != nil {
		fmt.Printf("invalid token: %s", err)
	}
}
