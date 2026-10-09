package tred

import (
	"bytes"
	"crypto/rand"
	"io"
	mr "math/rand"
	"sync"
	"testing"

	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/errors"
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
	assert.True(errors.Is(err, ErrUnsupportedCipher), "invalid error")

	// Invalid protocol version
	conf.Version = 0x11
	conf.Cipher = CHACHA20
	err = conf.Validate()
	assert.NotNil(err, "invalid configuration")
	assert.True(errors.Is(err, ErrUnsupportedVersion), "invalid error")

	// No key provided
	c2 := &Config{
		Version: Version10,
		Cipher:  CHACHA20,
	}
	err = c2.Validate()
	assert.NotNil(err, "invalid configuration")
	assert.True(errors.Is(err, ErrNoKey), "invalid error")

	// Start a valid worker
	conf.Version = Version10
	conf.Cipher = AES
	w, _ := NewWorker(conf)
	defer w.Destroy()

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
	defer w.Destroy()

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
	defer w.Destroy()

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
	defer w.Destroy()

	// Get random original content as a byte array
	originalContent := make([]byte, 1024)
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
	defer w.Destroy()

	// Get pool of random data streams
	pool := mr.Intn(50)
	var stuff [][]byte
	for i := 0; i < pool; i++ {
		stuff = append(stuff, make([]byte, (1024*1024)*mr.Intn(9)+1))
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

func TestNoncePerRun(t *testing.T) {
	assert := tdd.New(t)
	key := [32]byte{}
	rand.Read(key[:])
	conf, _ := DefaultConfig(key[:])
	w, _ := NewWorker(conf)
	defer w.Destroy()

	encrypt := func() []byte {
		out := bytes.NewBuffer([]byte{})
		_, err := w.Encrypt(bytes.NewReader([]byte("hello world")), out)
		assert.Nil(err, "encrypt error")
		return out.Bytes()
	}

	// Every encryption round must use a fresh nonce.
	a := header(encrypt())
	b := header(encrypt())
	assert.NotEqual(a[8:], b[8:], "nonce must differ between encryption rounds")
}

// shortReader limits each Read call to a small number of bytes.
type shortReader struct {
	r io.Reader
	n int
}

func (s *shortReader) Read(p []byte) (int, error) {
	if len(p) > s.n {
		p = p[:s.n]
	}
	return s.r.Read(p)
}

// eofReader delivers data in full-size chunks and returns io.EOF together
// with the remaining bytes on the final read.
type eofReader struct {
	data []byte
	off  int
}

func (r *eofReader) Read(p []byte) (int, error) {
	if r.off >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.off:])
	r.off += n
	if r.off >= len(r.data) {
		return n, io.EOF
	}
	return n, nil
}

func TestReaderEdgeCases(t *testing.T) {
	assert := tdd.New(t)
	content := make([]byte, 1024*1024)
	rand.Read(content)

	key := [32]byte{}
	rand.Read(key[:])
	conf, _ := DefaultConfig(key[:])
	w, _ := NewWorker(conf)
	defer w.Destroy()

	// Encrypt from a reader that returns data together with io.EOF.
	encrypted := bytes.NewBuffer([]byte{})
	_, err := w.Encrypt(&eofReader{data: content}, encrypted)
	assert.Nil(err, "encrypt error")

	// Decrypt through a reader that only delivers short reads.
	decrypted := bytes.NewBuffer([]byte{})
	_, err = w.Decrypt(&shortReader{r: bytes.NewReader(encrypted.Bytes()), n: 137}, decrypted)
	assert.Nil(err, "decrypt error")
	assert.Equal(content, decrypted.Bytes(), "bad decrypt result")
}

func FuzzInput(f *testing.F) {
	// Get random encryption key
	// Invalid key size, will be adjusted when expanding the secret key material
	key := [20]byte{}
	rand.Read(key[:])
	conf, _ := DefaultConfig(key[:])

	testcases := make([][]byte, 10)
	for i := range 10 {
		testcases[i] = make([]byte, 1024*1024)
		rand.Read(testcases[i])
		f.Add(testcases[i])
	}

	f.Fuzz(func(t *testing.T, originalContent []byte) {
		t.Run("CHACHA20", func(t *testing.T) {
			conf.Cipher = CHACHA20
			w, _ := NewWorker(conf)

			// Encrypt
			output := bytes.NewBuffer([]byte{})
			_, err := w.Encrypt(bytes.NewReader(originalContent), output)
			tdd.Nil(t, err, "encrypt error")

			// Decrypt
			decrypted := bytes.NewBuffer([]byte{})
			_, err = w.Decrypt(bytes.NewReader(output.Bytes()), decrypted)
			tdd.Nil(t, err, "decrypt error")
			tdd.Equal(t, originalContent, decrypted.Bytes(), "bad decrypt result")
		})

		t.Run("AES", func(t *testing.T) {
			conf.Cipher = AES
			w, _ := NewWorker(conf)

			// Encrypt
			output := bytes.NewBuffer([]byte{})
			_, err := w.Encrypt(bytes.NewReader(originalContent), output)
			tdd.Nil(t, err, "encrypt error")

			// Decrypt
			decrypted := bytes.NewBuffer([]byte{})
			_, err = w.Decrypt(bytes.NewReader(output.Bytes()), decrypted)
			tdd.Nil(t, err, "decrypt error")
			tdd.Equal(t, originalContent, decrypted.Bytes(), "bad decrypt result")
		})
	})
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

	output := bytes.NewBuffer([]byte{})
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _ = w.Encrypt(bytes.NewReader(content), output)
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

	output := bytes.NewBuffer([]byte{})
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _ = w.Encrypt(bytes.NewReader(content), output)
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
		_, _ = w.Decrypt(bytes.NewReader(ciphertext.Bytes()), output)
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
		_, _ = w.Decrypt(bytes.NewReader(ciphertext.Bytes()), output)
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
