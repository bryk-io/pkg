package tred

import (
	"crypto/aes"
	"crypto/cipher"
	"io"
	"sync"
	"time"

	"go.bryk.io/pkg/errors"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/sha3"
)

const (
	// Version10 provides the protocol version tag for 1.0.
	Version10 = 0x10

	// AES in GCM mode cipher code.
	AES = 0x00

	// CHACHA20 cipher code.
	CHACHA20 = 0x01

	// Encryption keys must be 32 bytes long to properly use ciphers in 256 bits mode.
	keySize = 32

	// Maximum payload size is 64kb.
	payloadSize = 64 * 1024

	// Package header is a 16 long byte array.
	// 	version (1) | cipher (1) | payload length (2) | seq (4) | nonce (8)
	// 	seq is packages counter that prevents rearrange
	// 	nonce mitigates problems of encryption key reuse
	headerSize = 16

	// Tag is a checksum validation code included in each package.
	tagSize = 16

	// Encrypted packet size.
	packetSize = headerSize + payloadSize + tagSize
)

// Common error values.
var (
	ErrInvalidSequenceNumber = errors.New("out of order packet")
	ErrInvalidPacketTag      = errors.New("invalid packet tag")
	ErrInvalidPayloadLen     = errors.New("invalid payload size")
	ErrUnsupportedCipher     = errors.New("unsupported cipher suite")
	ErrUnsupportedVersion    = errors.New("unsupported version code")
	ErrNoKey                 = errors.New("value for key is required")
	ErrRandomNonce           = errors.New("failed to read random nonce")
)

// Supported cipher suites.
var supportedCiphers = map[byte]func(key []byte) (cipher.AEAD, error){
	CHACHA20: chacha20poly1305.New,
	AES: func(key []byte) (cipher.AEAD, error) {
		aes256, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		return cipher.NewGCM(aes256)
	},
}

// Result defines the output of a successful encrypt or decrypt operation.
type Result struct {
	// Number of packets produced
	Packets uint32

	// Execution time
	Duration time.Duration
}

// Worker provides a protocol agent.
type Worker struct {
	conf  *Config
	seq   uint32
	nonce [8]byte
	mutex sync.Mutex
}

// NewWorker returns a usable protocol worker instance.
func NewWorker(c *Config) (*Worker, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if err := c.init(); err != nil {
		return nil, err
	}
	w := &Worker{
		conf: c,
		seq:  0,
	}
	if err := w.expandKey(nil); err != nil {
		return nil, err
	}
	return w, nil
}

// Destroy will securely remove the key material held by the worker. Note that
// the worker shares its configuration with the caller, so a configuration
// instance should not be shared across workers when calling this method.
func (w *Worker) Destroy() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	wipeBytes(w.conf.Key)
	w.conf.Key = nil
}

// Encrypt will secure the 'input' content and send it to 'output'.
func (w *Worker) Encrypt(input io.Reader, output io.Writer) (*Result, error) {
	// Get cipher
	c, ok := supportedCiphers[w.conf.Cipher]
	if !ok {
		return nil, ErrUnsupportedCipher
	}
	aead, err := c(w.conf.Key[:])
	if err != nil {
		return nil, err
	}

	// Lock internal state
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Reset worker
	payload := make([]byte, payloadSize)
	w.seq = 0
	if err := w.newNonce(); err != nil {
		return nil, err
	}
	start := time.Now()

	// Process input
	for {
		// Avoid leaking plaintext bytes from a previous packet into the
		// current encryption round.
		clear(payload)
		n, err := input.Read(payload)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		if n > 0 {
			// Encrypt payload
			// Use 'seq | nonce' as operation nonce
			// Use 'version | cipher | payload length' as additional data
			h := w.buildHeader(n)
			ciphertext := aead.Seal(nil, h[4:headerSize], payload, h[:4])

			// Build package
			packet := make([]byte, headerSize+len(ciphertext))
			copy(packet[:headerSize], h)
			copy(packet[headerSize:], ciphertext)
			if err := writeFull(output, packet); err != nil {
				return nil, err
			}
			w.seq++
		}
		if errors.Is(err, io.EOF) || n == 0 {
			break
		}
	}
	wipeBytes(payload)

	// Return final result
	return &Result{
		Packets:  w.seq,
		Duration: time.Since(start),
	}, nil
}

// Decrypt will open the secure 'input' content and send it to 'output'.
func (w *Worker) Decrypt(input io.Reader, output io.Writer) (*Result, error) {
	// Get cipher
	c, ok := supportedCiphers[w.conf.Cipher]
	if !ok {
		return nil, ErrUnsupportedCipher
	}
	aead, err := c(w.conf.Key[:])
	if err != nil {
		return nil, err
	}

	// Lock internal state
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Reset worker
	packet := make([]byte, packetSize)
	w.seq = 0
	start := time.Now()

	// Process input
	for {
		if _, err := io.ReadFull(input, packet); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, errors.Wrap(err, ErrInvalidPacketTag.Error())
		}
		// Validate packet sequence
		h := header(packet)
		if h.SequenceNumber() != w.seq {
			return nil, ErrInvalidSequenceNumber
		}
		if h.Len() > payloadSize {
			return nil, ErrInvalidPayloadLen
		}

		// Decrypt and validate packet ciphertext
		ciphertext := packet[headerSize:]
		payload, err := aead.Open(nil, h[4:headerSize], ciphertext, h[:4])
		if err != nil {
			return nil, errors.Wrap(err, ErrInvalidPacketTag.Error())
		}

		// Add output
		if err := writeFull(output, payload[:h.Len()]); err != nil {
			return nil, err
		}
		w.seq++
	}

	// Return final result
	return &Result{
		Packets:  w.seq,
		Duration: time.Since(start),
	}, nil
}

// Generate a fresh random nonce for a new encryption round.
func (w *Worker) newNonce() error {
	if _, err := w.conf.rng.Read(w.nonce[:]); err != nil {
		return ErrRandomNonce
	}
	return nil
}

// Build a valid packet header block.
func (w *Worker) buildHeader(packetLength int) headerBlock {
	h := headerBlock(make([]byte, headerSize))
	h.SetVersion(w.conf.Version)
	h.SetCipher(w.conf.Cipher)
	h.SetLen(packetLength)
	h.SetSequenceNumber(w.seq)
	h.SetNonce(w.nonce)
	return h
}

// Build a valid output manifest block.
func (w *Worker) buildManifest(digest []byte) manifestBlock {
	m := manifestBlock(make([]byte, manifestSize))
	m.SetVersion(w.conf.Version)
	m.SetCipher(w.conf.Cipher)
	m.SetLen(int(w.seq))
	m.SetChecksum(digest)
	return m
}

// Securely expand secret key material.
func (w *Worker) expandKey(info []byte) error {
	h := hkdf.New(sha3.New256, w.conf.Key, make([]byte, keySize), info)
	buf := make([]byte, keySize)
	if _, err := io.ReadFull(h, buf); err != nil {
		return errors.Wrap(err, "failed to read HKDF key")
	}
	old := w.conf.Key
	w.conf.Key = buf
	wipeBytes(old)
	return nil
}

// Securely removes the contents of the provided buffer.
func wipeBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// Writes the complete buffer to the provided writer, handling short writes.
func writeFull(w io.Writer, b []byte) error {
	for len(b) > 0 {
		n, err := w.Write(b)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		b = b[n:]
	}
	return nil
}
