/*
Package x25519 provides a ECDH (Elliptic Curve Diffie-Hellman) wrapper for curve X25519.

The main component in the package is the 'KeyPair' instance. Each key pair needs to be
securely removed from memory by calling the "Destroy" method.

Key Creation

There are 3 mechanisms to create a new key pair.

	// 1. Create a completely random new key
	rk, _ := New()

	// 2. Key using a given seed material
	sk, _ := FromSeed([]byte("material"))

	// 3. Load from PEM-encoded content
	pk, _ := Unmarshal(pemBinData)

However created, the key pair instance always use a locked memory buffer to securely
hold private information. Is mandatory to properly release the memory buffer after
using the key by calling the 'Destroy' method.

	// Securely release in-memory secrets
	kp.Destroy()

Key Usage

Diffie-Hellman is a shared key creation mechanism. The main use of a key pair is to
generate a shared secret with a provided public key.

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
*/
package x25519
