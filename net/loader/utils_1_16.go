//go:build go1.16
// +build go1.16

package loader

import (
	"crypto/tls"
	"encoding/base64"
	"os"
	"path"
)

func loadPEM(value string) ([]byte, error) {
	// Base64 string
	c, err := base64.StdEncoding.DecodeString(value)
	if err == nil {
		return c, nil
	}

	// Load file
	return os.ReadFile(path.Clean(value))
}

// Validates a certificate/private key pair from it's PEM-encoded byte arrays.
func isKeyPairPEM(cert, key []byte) bool {
	_, err := tls.X509KeyPair(cert, key)
	return err == nil
}
