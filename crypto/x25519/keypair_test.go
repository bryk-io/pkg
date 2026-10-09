package x25519

import (
	"bytes"
	"encoding/pem"
	"testing"

	tdd "github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	// The method "github.com/awnumar/memguard/core.NewCoffer" currently
	// leaks a routine used to re-key the global enclave handler.
	// https://github.com/awnumar/memguard/blob/master/core/coffer.go#L36
	goleak.VerifyTestMain(m, goleak.IgnoreTopFunction("github.com/awnumar/memguard/core.NewCoffer.func1"))
}

func TestNew(t *testing.T) {
	assert := tdd.New(t)
	c, err := New()
	assert.Nil(err, "create new key error")
	c.Destroy()
}

func TestDH(t *testing.T) {
	assert := tdd.New(t)
	k1, _ := New()
	k2, _ := New()
	dh1 := k1.DH(k2.PublicKey())
	dh2 := k2.DH(k1.PublicKey())
	assert.Equal(dh1, dh2, "bad diffie-hellman result")
	k1.Destroy()
	k2.Destroy()
}

func TestMarshal(t *testing.T) {
	assert := tdd.New(t)
	k, _ := New()
	b1, _ := k.MarshalBinary()
	b2, _ := k.MarshalBinary()
	assert.Equal(b1, b2, "marshal result should be deterministic")

	pub := k.PublicKey()
	var p1 [32]byte
	copy(p1[:], pub[:])
	k2, err := Unmarshal(b2)
	assert.Nil(err, "unmarshal error")
	pub2 := k2.PublicKey()
	assert.Equal(p1, pub2, "invalid restore")

	k.Destroy()
	k2.Destroy()
}

func TestUnmarshalInvalid(t *testing.T) {
	assert := tdd.New(t)

	// Non-PEM content must return an error instead of panicking.
	kp, err := Unmarshal([]byte("this is not PEM data"))
	assert.Nil(kp, "expected no key pair on invalid data")
	assert.NotNil(err, "expected error on invalid data")

	// PEM content with an unexpected block type must be rejected.
	bl := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: make([]byte, 32),
	}
	kp, err = Unmarshal(pem.EncodeToMemory(bl))
	assert.Nil(kp, "expected no key pair on invalid type")
	assert.NotNil(err, "expected error on invalid type")
}

func TestDHWithKDF(t *testing.T) {
	assert := tdd.New(t)
	k1, _ := New()
	k2, _ := New()
	defer k1.Destroy()
	defer k2.Destroy()

	s1, err := k1.WithKDF(k2.PublicKey(), 32, nil)
	assert.Nil(err, "kdf error")
	s2, err := k2.WithKDF(k1.PublicKey(), 32, nil)
	assert.Nil(err, "kdf error")
	assert.Equal(s1, s2, "bad diffie-hellman result")

	// The raw DH output must agree with the KDF expansion on both sides.
	raw := k1.DH(k2.PublicKey())
	assert.NotNil(raw, "raw dh failed")
	assert.NotEqual(s1, raw, "kdf output must differ from the raw secret")

	// Different context information must derive different keys.
	s3, err := k1.WithKDF(k2.PublicKey(), 32, []byte("other-context"))
	assert.Nil(err, "kdf error")
	assert.NotEqual(s1, s3, "info must change the derived key")
}

func TestDestroy(t *testing.T) {
	// Normal run
	assert := tdd.New(t)
	k, _ := New()
	k.Destroy()
	assert.Nil(k.lb, "failed to destroy locked memory buffer")

	// This time lb is no longer initialized but runs ok
	k.Destroy()
}

func TestFromSeed(t *testing.T) {
	assert := tdd.New(t)
	seed := []byte("super secret seed material")
	k1, err := FromSeed(seed)
	assert.Nil(err, "from seed error")

	// Both keys should match after restore
	cp, _ := FromSeed(seed)
	assert.Equal(k1.public, cp.public, "restore error")
	assert.Equal(k1.lb.Bytes(), cp.lb.Bytes(), "restore error")

	// Run DH with restored key and another randomly generated
	k2, _ := New()
	dh1 := k1.DH(k2.PublicKey())
	dh2 := k2.DH(k1.PublicKey())
	assert.Equal(dh1, dh2, "bad diffie-hellman result")

	k1.Destroy()
	k2.Destroy()
	cp.Destroy()
}

// Generate a shared secret between two key pair instances.
func ExampleNew() {
	// Generate peers
	alice, _ := New()
	bob, _ := New()
	defer alice.Destroy()
	defer bob.Destroy()

	// Generate shared secret on both sides
	s1 := alice.DH(bob.PublicKey())
	s2 := bob.DH(alice.PublicKey())

	// Test secrets
	if !bytes.Equal(s1, s2) {
		panic("failed to generate valid secret")
	}
}
