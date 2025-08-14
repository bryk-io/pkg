package ed25519

import (
	"fmt"
	"testing"

	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/crypto/x25519"
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
	ed, err := New()
	assert.Nil(err, "failed to create new key")

	edb, err := ed.MarshalBinary()
	assert.Nil(err, "marshal error")
	assert.NotNil(edb, "marshal error")
	ed.Destroy()
}

func TestSignatureVerification(t *testing.T) {
	assert := tdd.New(t)
	ed, err := New()
	assert.Nil(err, "failed to create new key")
	defer ed.Destroy()

	msg := []byte("message content")
	s := ed.Sign(msg)
	assert.True(ed.Verify(msg, s), "verify error")
	assert.False(ed.Verify([]byte("invalid message"), s), "verify error")
	assert.False(ed.Verify(msg, append(s, s...)), "verify error")
}

func TestEncodeDecode(t *testing.T) {
	assert := tdd.New(t)
	k, _ := New()
	b1, _ := k.MarshalBinary()
	b2, _ := k.MarshalBinary()
	assert.Equal(b1, b2, "non deterministic marshal result")

	pub := k.PublicKey()
	var p1 [32]byte
	copy(p1[:], pub[:])
	k.Destroy()

	k2, err := Unmarshal(b2)
	assert.Nil(err, "unmarshal error")
	assert.NotNil(k2, "unmarshal error")
	assert.Equal(p1, k2.pub, "invalid key restore")
	k2.Destroy()
}

func TestDestroy(t *testing.T) {
	assert := tdd.New(t)
	ed, _ := New()
	ed.Destroy()
	assert.Empty(ed.priv, "failed to destroy locked memory buffer")

	// This time lb is no longer initialized but runs ok
	ed.Destroy()
}

func TestRestore(t *testing.T) {
	assert := tdd.New(t)
	seed := []byte("super-secret-value")
	k, err := FromSeed(seed)
	assert.Nil(err, "from seed error")

	// Test message
	msg := []byte("message to sign")
	s := k.Sign(msg)
	assert.True(k.Verify(msg, s), "verify error")

	// Attempt verification with a random key
	k2, _ := New()
	assert.False(k2.Verify(msg, s), "verification with another key should fail")
	k2.Destroy()

	// Destroy first keypair
	priv := make([]byte, 64)
	copy(priv, k.PrivateKey())
	k.Destroy()

	// Create a new keypair using the same seed value
	k3, err := FromSeed(seed)
	assert.Nil(err, "from seed error")
	assert.True(k3.Verify(msg, s), "verify error")
	k3.Destroy()

	// Create a new keypair using the same private key
	k4, err := FromPrivateKey(priv)
	assert.Nil(err, "from private key error")
	assert.True(k4.Verify(msg, s), "verify error")
	k4.Destroy()
}

func TestDH(t *testing.T) {
	assert := tdd.New(t)

	// Key agreements between ed25519 instances
	alice, _ := New()
	bob, _ := New()
	s1 := alice.DH(bob.ToX25519())
	s2 := bob.DH(alice.ToX25519())
	assert.Equal(s1, s2, "failed to generate shared secret")

	// Key agreement between ed25519 and x25519 instances
	charlie, _ := x25519.New()
	s3 := charlie.DH(alice.ToX25519())
	s4 := alice.DH(charlie.PublicKey())
	assert.Equal(s3, s4, "failed to generate shared secret")
}

func ExampleUnmarshal() {
	// Restore key from a previously PEM-encoded private key
	kp, err := Unmarshal([]byte("pem-encoded-private-key"))
	if err != nil {
		panic(err)
	}
	defer kp.Destroy()

	// Use the key to produce a signature
	signature := kp.Sign([]byte("message-to-sign"))
	fmt.Printf("signature produced: %x", signature)
}

func ExampleKeyPair_Verify() {
	msg := []byte("message-to-sign")
	kp, _ := New()
	signature := kp.Sign(msg)
	fmt.Printf("verification result: %v", kp.Verify(msg, signature))
}
