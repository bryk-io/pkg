package tred

import (
	"bytes"
	"math/rand"
	"strings"
	"sync"
	"testing"

	tdd "github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestConfig(t *testing.T) {
	assert := tdd.New(t)
	key := [32]byte{}
	rand.Read(key[:])
	conf, _ := DefaultConfig(key[:])

	// Unsupported ciphers
	conf.Cipher = 0x03
	err := conf.Validate()
	assert.NotNil(err, "invalid configuration")
	assert.True(strings.Contains(err.Error(), ErrUnsupportedCipher), "invalid error")

	// Invalid protocol version
	conf.Version = 0x11
	conf.Cipher = CHACHA20
	err = conf.Validate()
	assert.NotNil(err, "invalid configuration")
	assert.True(strings.Contains(err.Error(), ErrUnsupportedVersion), "invalid error")

	// No key provided
	c2 := &Config{
		Version: Version10,
		Cipher:  CHACHA20,
	}
	err = c2.Validate()
	assert.NotNil(err, "invalid configuration")
	assert.True(strings.Contains(err.Error(), ErrNoKey), "invalid error")

	// Start a valid worker
	conf.Version = Version10
	conf.Cipher = AES
	w, _ := NewWorker(conf)

	// Key expand
	assert.NotEqual(key, w.conf.Key, "failed to expand key material")

	t.Run("header", func(t *testing.T) {
		input := bytes.NewReader([]byte("original content"))
		output := bytes.NewBuffer([]byte{})
		_, err = w.Encrypt(input, output)
		assert.Nil(err, "encrypt error")

		h := header(output.Bytes())
		assert.Equal(conf.Version, h.Version(), "invalid version code")
		assert.Equal(conf.Cipher, h.Cipher(), "invalid cipher code")
	})
}

func TestManifest(t *testing.T) {
	assert := tdd.New(t)
	key := [32]byte{}
	rand.Read(key[:])
	conf, _ := DefaultConfig(key[:])
	w, _ := NewWorker(conf)

	fakeDigest := [32]byte{}
	rand.Read(fakeDigest[:])
	w.seq = 100
	m := w.buildManifest(fakeDigest[:])
	m2 := manifest(m)

	assert.Equal(w.conf.Version, m.Version(), "invalid version code")
	assert.Equal(w.conf.Cipher, m.Cipher(), "invalid cipher code")
	assert.Equal(100, m.Len(), "invalid length")
	assert.Equal(fakeDigest[:], m.Checksum(), "invalid checksum")
	assert.Equal(m, m2, "restore error")
}

func TestChaCha(t *testing.T) {
	assert := tdd.New(t)
	// Get random encryption key
	// Invalid key size, will be adjusted when expanding the secret key material
	key := [20]byte{}
	rand.Read(key[:])
	conf, _ := DefaultConfig(key[:])
	conf.Cipher = CHACHA20
	w, _ := NewWorker(conf)

	// Get random original content as a byte array
	originalContent := make([]byte, 1024*1024)
	rand.Read(originalContent)

	// Encrypt
	output := bytes.NewBuffer([]byte{})
	_, err := w.Encrypt(bytes.NewReader(originalContent), output)
	assert.Nil(err, "encrypt error")
	assert.NotEqual(originalContent, output.Bytes(), "bad encrypt result")

	// Decrypt
	decrypted := bytes.NewBuffer([]byte{})
	_, err = w.Decrypt(bytes.NewReader(output.Bytes()), decrypted)
	assert.Nil(err, "decrypt error")
	assert.Equal(originalContent, decrypted.Bytes(), "bad decrypt result")
}

func TestAES(t *testing.T) {
	assert := tdd.New(t)
	// Get random encryption key
	key := [32]byte{}
	rand.Read(key[:])
	conf, _ := DefaultConfig(key[:])
	w, _ := NewWorker(conf)

	// Get random original content as a byte array
	originalContent := make([]byte, 1024*1024)
	rand.Read(originalContent)

	// Encrypt
	output := bytes.NewBuffer([]byte{})
	_, err := w.Encrypt(bytes.NewReader(originalContent), output)
	assert.Nil(err, "encrypt error")
	assert.NotEqual(originalContent, output.Bytes(), "bad encrypt result")

	// Decrypt
	decrypted := bytes.NewBuffer([]byte{})
	_, err = w.Decrypt(bytes.NewReader(output.Bytes()), decrypted)
	assert.Nil(err, "decrypt error")
	assert.Equal(originalContent, decrypted.Bytes(), "bad decrypt result")
}

func TestConcurrency(t *testing.T) {
	assert := tdd.New(t)
	key := [32]byte{}
	rand.Read(key[:])
	conf, _ := DefaultConfig(key[:])
	w, _ := NewWorker(conf)

	// Get pool of random data streams
	pool := rand.Intn(50)
	var stuff [][]byte
	for i := 0; i < pool; i++ {
		stuff = append(stuff, make([]byte, (1024*1024)*rand.Intn(9)+1))
		rand.Read(stuff[i])
	}
	wg := sync.WaitGroup{}
	wg.Add(pool)

	// Concurrently process all data streams
	for _, v := range stuff {
		go func(v []byte) {
			_, err := w.Encrypt(bytes.NewReader(v), bytes.NewBuffer([]byte{}))
			assert.Nil(err, "encrypt error")
			wg.Done()
		}(v)
	}
	wg.Wait()
}

func BenchmarkWorker_EncryptChaCha(b *testing.B) {
	// Get worker
	key := [20]byte{}
	rand.Read(key[:])
	conf, _ := DefaultConfig(key[:])
	conf.Cipher = CHACHA20
	w, _ := NewWorker(conf)

	// Get random content to encrypt
	content := make([]byte, 1024*1024)
	rand.Read(content)

	input := bytes.NewReader(content)
	output := bytes.NewBuffer([]byte{})
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _ = w.Encrypt(input, output)
	}
}

func BenchmarkWorker_EncryptAES(b *testing.B) {
	// Get worker
	key := [20]byte{}
	rand.Read(key[:])
	conf, _ := DefaultConfig(key[:])
	conf.Cipher = AES
	w, _ := NewWorker(conf)

	// Get random content to encrypt
	content := make([]byte, 1024*1024)
	rand.Read(content)

	input := bytes.NewReader(content)
	output := bytes.NewBuffer([]byte{})
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _ = w.Encrypt(input, output)
	}
}

func BenchmarkWorker_DecryptChaCha(b *testing.B) {
	// Get worker
	key := [20]byte{}
	rand.Read(key[:])
	conf, _ := DefaultConfig(key[:])
	conf.Cipher = CHACHA20
	w, _ := NewWorker(conf)

	// Get random content to encrypt
	content := make([]byte, 1024*1024)
	rand.Read(content)

	// Encrypt content
	input := bytes.NewReader(content)
	ciphertext := bytes.NewBuffer([]byte{})
	_, _ = w.Encrypt(input, ciphertext)
	output := bytes.NewBuffer([]byte{})

	// Measure decrypt time
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _ = w.Decrypt(ciphertext, output)
	}
}

func BenchmarkWorker_DecryptAES(b *testing.B) {
	// Get worker
	key := [20]byte{}
	rand.Read(key[:])
	conf, _ := DefaultConfig(key[:])
	conf.Cipher = AES
	w, _ := NewWorker(conf)

	// Get random content to encrypt
	content := make([]byte, 1024*1024)
	rand.Read(content)

	// Encrypt content
	input := bytes.NewReader(content)
	ciphertext := bytes.NewBuffer([]byte{})
	_, _ = w.Encrypt(input, ciphertext)
	output := bytes.NewBuffer([]byte{})

	// Measure decrypt time
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _ = w.Decrypt(ciphertext, output)
	}
}

// Complete protocol usage.
// Error handling omitted for brevity.
func ExampleNewWorker() {
	// Get random content to encrypt
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
}
