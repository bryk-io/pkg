/*
Package ed25519 provides a EdDSA (Edwards-curve Digital Signature Algorithm) handler for Curve25519.

The main component in the package is the 'KeyPair' instance. All functionality
to generate and validate digital signatures is available by its methods.

# Key Creation

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

# Key Usage

A key pair can be used to produce and verify digital signatures.

	msg := []byte("message-to-sign")
	kp, _ := New()
	signature := kp.Sign(msg)
	log.Printf("verification result: %v", kp.Verify(msg, signature))
*/
package ed25519
