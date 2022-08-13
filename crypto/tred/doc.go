/*
Package tred provides a reference implementation for the Tamper Resistant Encrypted Data protocol.

Proper secure data management must support scenarios for in-transit and at-rest use cases. For
network communications (i.e, in-transit) there are already well established and secure protocols
available like TLS. The protocols available for secure data persistence (i.e., at-rest) on the
other hand are not as reliable and/or robust for several reasons.

TRED introduces a secure, fast and easy-to-use mechanism to handle secure data persistence. The
protocol uses a simple data structure that introduces very small overhead, prevent cipher text
manipulation and greatly simplifies data integrity validation.

The original data is split into individual packets for optimum security and performance. Each
packet is properly tagged and processed using an authentication encryption cipher. The final
output prevent manipulation (tamper attempts) of the produced cipher text.

	// output:
	packet[...]

	// packet:
	// tag is calculated and validated by the AEAD cipher
	header (16) | payload (1 byte - 64 KB) | tag (16)

	// header:
	version (1) | cipher (1) | payload length (2) | seq (4) | nonce (8)

# Usage

To facilitate the integration of the protocol with higher level components and primitives this
package introduces a 'Worker' component with a simple interface.

	content := make([]byte, 1024*256)
	rand.Read(content)

	// Create a worker instance using the ChaCha20 cipher
	conf, _ := DefaultConfig([]byte("super-secret-key"))
	conf.Cipher = CHACHA20
	w, _ := NewWorker(conf)

	// Encrypt data
	secure := bytes.NewBuffer([]byte{})
	_, _ = w.Encrypt(bytes.NewReader(content), secure)
	if bytes.Equal(content, secure.Bytes()) {
		panic("failed to encrypt original content")
	}

	// Decrypt data
	verification := bytes.NewBuffer([]byte{})
	_, _ = w.Decrypt(secure, verification)
	if !bytes.Equal(content, verification.Bytes()) {
		panic("failed to decrypt data")
	}
*/
package tred
