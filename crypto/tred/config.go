package tred

import (
	"crypto/rand"
	"errors"
	"io"
)

// Config provides all configuration parameters available when creating a new
// TRED worker agent.
type Config struct {
	// Protocol version
	Version byte

	// Cipher code
	Cipher byte

	// Secure cryptographic key to use
	Key []byte

	// Internals
	rng   io.Reader
	nonce [8]byte
}

// DefaultConfig generates sane default configuration parameters using the provided key value.
func DefaultConfig(k []byte) (*Config, error) {
	c := &Config{
		Version: Version10,
		Cipher:  AES,
		Key:     k,
	}
	return c, c.init()
}

// Validate the configuration instance against common setup errors.
func (c *Config) Validate() error {
	// Cipher
	if _, ok := supportedCiphers[c.Cipher]; !ok {
		return errors.New(ErrUnsupportedCipher)
	}

	// Version
	if c.Version != Version10 {
		return errors.New(ErrUnsupportedVersion)
	}

	// No key if provided
	if len(c.Key) == 0 {
		return errors.New(ErrNoKey)
	}
	return nil
}

// Initialize internal configuration elements.
func (c *Config) init() error {
	c.rng = rand.Reader
	if _, err := c.rng.Read(c.nonce[:]); err != nil {
		return errors.New(ErrRandomNonce)
	}
	return nil
}
