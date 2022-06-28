package x25519

import (
	"bytes"
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
