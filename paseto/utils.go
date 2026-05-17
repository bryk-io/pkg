package paseto

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"strings"

	"dario.cat/mergo"
	"go.bryk.io/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/chacha20"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

// Base64 encoding used consistently by all standard keys.
// https://tools.ietf.org/html/draft-paragon-paseto-rfc-00#section-2.1
var b64 = base64.RawURLEncoding

// Is "v" part of the provided list?
func in(v string, list []string) bool {
	for _, k := range list {
		if k == v {
			return true
		}
	}
	return false
}

// Flatten the provided items into a single map structure.
func merge(item ...interface{}) (map[string]interface{}, error) {
	dst := make(map[string]interface{})
	for _, el := range item {
		// Ignore "nil" elements
		if el == nil {
			continue
		}
		// Re-encode element into a map structure to match the same "dst" type
		b, err := json.Marshal(el)
		if err != nil {
			return nil, errors.New("failed to encode item")
		}
		m := make(map[string]interface{})
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, errors.New("failed to re-encode item")
		}
		if err := mergo.Merge(&dst, m, mergo.WithOverride); err != nil {
			return nil, err
		}
	}
	return dst, nil
}

// Encrypt/decrypt message "m", with key "k" and nonce "n" using "AES-256-CTR".
func aesCTR(m, k, n []byte) ([]byte, error) {
	block, err := aes.NewCipher(k)
	if err != nil {
		return nil, errors.Errorf("failed to create aes cipher: %w", err)
	}
	ct := make([]byte, len(m))
	ctr := cipher.NewCTR(block, n)
	ctr.XORKeyStream(ct, m)
	return ct, nil
}

// Encrypt/decrypt message "m", with nonce "n", key "k", and additional data "ad"
// using XChaCha20-Poly1305.
func xChaChaPoly(m, n, k, ad []byte, enc bool) ([]byte, error) {
	// Get cipher
	aead, err := chacha20poly1305.NewX(k)
	if err != nil {
		return nil, err
	}

	// Encrypt
	if enc {
		c := aead.Seal(m[:0], n, m, ad)
		return c, nil
	}

	// Decrypt
	return aead.Open(m[:0], n, m, ad)
}

// Encrypt/decrypt message "m", with nonce "n", key "k" using the ChaCha20 stream
// cipher. If a nonce of 24 bytes is provided, the XChaCha20 construction will be used.
func xChaCha20(m, n, k []byte) ([]byte, error) {
	// Get cipher
	ci, err := chacha20.NewUnauthenticatedCipher(k, n)
	if err != nil {
		return nil, errors.Errorf("failed to initialize cipher: %w", err)
	}
	res := make([]byte, len(m))
	ci.XORKeyStream(res, m)
	return res, nil
}

// HMAC-SHA-384 authenticated hash of message "m" using key "k".
func ah(m, k []byte) ([]byte, error) {
	hh := hmac.New(sha512.New384, k)
	if _, err := hh.Write(m); err != nil {
		return nil, err
	}
	return hh.Sum(nil), nil
}

// BLAKE2b-MAC of size "s", for message "m", using key "k".
func bh(s int, k, m []byte) ([]byte, error) {
	hh, err := blake2b.New(s, k)
	if err != nil {
		return nil, errors.Errorf("failed to initialize MAC: %w", err)
	}
	if _, err := hh.Write(m); err != nil {
		return nil, err
	}
	return hh.Sum(nil), nil
}

// https://tools.ietf.org/html/draft-paragon-paseto-rfc-00#section-2.2.1
func pae(pieces ...[]byte) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, int64(len(pieces)))
	for _, p := range pieces {
		_ = binary.Write(buf, binary.LittleEndian, int64(len(p)))
		buf.Write(p)
	}
	return buf.Bytes()
}

// https://tools.ietf.org/html/draft-paragon-paseto-rfc-00#section-4.3.2
func splitKey(key, salt []byte) (ek, ak []byte, err error) {
	er := hkdf.New(sha512.New384, key, salt, []byte("paseto-encryption-key"))
	ar := hkdf.New(sha512.New384, key, salt, []byte("paseto-auth-key-for-aead"))
	ek = make([]byte, 32)
	ak = make([]byte, 32)
	if _, err = io.ReadFull(er, ek); err != nil {
		return nil, nil, err
	}
	if _, err = io.ReadFull(ar, ak); err != nil {
		return nil, nil, err
	}
	return ek, ak, nil
}

// https://github.com/paseto-standard/paseto-spec/blob/master/docs/01-Protocol-Versions/Version3.md
func v3SplitKey(k, n []byte) (ek, ak, n2 []byte, err error) {
	er := hkdf.New(sha512.New384, k, nil, append([]byte("paseto-encryption-key"), n...))
	kd1 := make([]byte, 48)
	if _, err = io.ReadFull(er, kd1); err != nil {
		return nil, nil, nil, err
	}
	ek = kd1[:32]         // leftmost 32 bytes of the 1st key derivation
	n2 = kd1[32:]         // remaining 16 bytes of the 1st key derivation
	ak = make([]byte, 48) // authentication key obtained on the 2nd key derivation
	ar := hkdf.New(sha512.New384, k, nil, append([]byte("paseto-auth-key-for-aead"), n...))
	if _, err = io.ReadFull(ar, ak); err != nil {
		return nil, nil, nil, err
	}
	return ek, ak, n2, nil
}

// https://github.com/paseto-standard/paseto-spec/blob/master/docs/01-Protocol-Versions/Version4.md
func v4SplitKey(k, n []byte) (ek, ak, n2 []byte, err error) {
	// Prepared keyed hash for step 1
	kd1, err := bh(56, k, append([]byte("paseto-encryption-key"), n...))
	if err != nil {
		return nil, nil, nil, err
	}

	// Prepared keyed hash for step 2
	kd2, err := bh(32, k, append([]byte("paseto-auth-key-for-aead"), n...))
	if err != nil {
		return nil, nil, nil, err
	}

	ek = kd1[:32] // leftmost 32 bytes of the 1st key derivation
	n2 = kd1[32:] // remaining 24 bytes of the 1st key derivation
	ak = kd2      // 32 bytes authentication key obtained from 2nd key derivation
	return
}

// https://tools.ietf.org/html/draft-paragon-paseto-rfc-00#section-4.3.1
func v1GetNonce(m, n []byte) ([]byte, error) {
	ah, err := ah(m, n)
	if err != nil {
		return nil, err
	}
	return ah[:32], nil
}

// https://tools.ietf.org/html/draft-paragon-paseto-rfc-00#section-5.3.1
func v2GetNonce(m, n []byte) ([]byte, error) {
	// BLAKE2b of the message "m" with using "n" as the key, with an output
	// length of 24
	ah, err := blake2b.New(24, n)
	if err != nil {
		return nil, err
	}
	if _, err = ah.Write(m); err != nil {
		return nil, err
	}
	return ah.Sum(nil), nil
}

// Encrypts the token's payload `pld`
//
//	tt  = token type
//	k   = key to encrypt/sign the generated token
//	pld = payload contents
//	ftr = footer contents, optional
//	ia  = implicit assertions, optional
func encrypt(tt string, k EncryptionKey, pld, ftr, ia []byte) ([]byte, error) {
	ek, err := k.Secret()
	if err != nil {
		return nil, err
	}

	// Use proper protocol method based on the token type
	switch ProtocolVersion(tt) {
	case V1L:
		return v1Encrypt(pld, ftr, ek)
	case V2L:
		return v2Encrypt(pld, ftr, ek)
	case V3L:
		return v3Encrypt(pld, ftr, ia, ek)
	case V4L:
		return v4Encrypt(pld, ftr, ia, ek)
	default:
		return nil, errors.New("invalid token type")
	}
}

// Decrypt an existing token.
//
//	t  = token
//	k  = cryptographic key
//	ia = implicit assertions, optional
func decrypt(t *Token, k EncryptionKey, ia []byte) ([]byte, error) {
	dk, err := k.Secret()
	if err != nil {
		return nil, err
	}

	switch ProtocolVersion(t.Header()) {
	case V1L:
		return v1Decrypt(t.String(), t.ftr, dk)
	case V2L:
		return v2Decrypt(t.String(), t.ftr, dk)
	case V3L:
		return v3Decrypt(t.String(), t.ftr, ia, dk)
	case V4L:
		return v4Decrypt(t.String(), t.ftr, ia, dk)
	default:
		return nil, errors.New("invalid token type")
	}
}

// Sign the token's payload `pld`
//
//	tt  = token type
//	k   = key to encrypt/sign the generated token
//	pld = payload contents
//	ftr = footer contents, optional
//	ia  = implicit assertions, optional
func sign(tt string, k SigningKey, pld, ftr, ia []byte) ([]byte, error) {
	// Use proper protocol method based on the token type
	switch ProtocolVersion(tt) {
	case V1P:
		return v1Sign(pld, ftr, k)
	case V2P:
		return v2Sign(pld, ftr, k)
	case V3P:
		return v3Sign(pld, ftr, ia, k)
	case V4P:
		return v4Sign(pld, ftr, ia, k)
	default:
		return nil, errors.New("invalid token type")
	}
}

// Verifies the payload of an existing token.
//
//	t  = token
//	k  = cryptographic key
//	ia = implicit assertions, optional
func verify(t *Token, k SigningKey, ia []byte) ([]byte, error) {
	// Use proper protocol method based on the token type
	switch ProtocolVersion(t.Header()) {
	case V1P:
		return v1Verify(t.String(), t.ftr, k)
	case V2P:
		return v2Verify(t.String(), t.ftr, k)
	case V3P:
		return v3Verify(t.String(), t.ftr, ia, k)
	case V4P:
		return v4Verify(t.String(), t.ftr, ia, k)
	default:
		return nil, errors.New("invalid token type")
	}
}

// https://tools.ietf.org/html/draft-paragon-paseto-rfc-00#section-4.3.2
func v1Encrypt(m, f, k []byte) ([]byte, error) {
	// Set header value
	h := "v1.local."

	// Footer defaults to an empty string
	if f == nil {
		f = []byte("")
	}

	// Generate 32 random bytes
	rndB := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, rndB); err != nil {
		return nil, errors.New("failed to read random bytes")
	}

	// Calculate nonce
	n, err := v1GetNonce(m, rndB)
	if err != nil {
		return nil, errors.New("failed to calculate message nonce")
	}

	// Split key into an encryption key ("ek") and an authentication key ("ak")
	// using the leftmost 16 bytes of "n" as the HKDF salt.
	ek, ak, err := splitKey(k, n[:16])
	if err != nil {
		return nil, errors.New("failed to generate encryption keys")
	}

	// Encrypt the message "m" using "AES-256-CTR", with "ek" as the key and the
	// rightmost 16 bytes of "n" as the nonce.
	c, err := aesCTR(m, ek, n[16:])
	if err != nil {
		return nil, err
	}

	// Pack "h", "n", "c", and "f" together (in that order) using PAE
	preAuth := pae([]byte(h), n, c, f)

	// Calculate authenticated version of preAuth using "ak"
	t, err := ah(preAuth, ak)
	if err != nil {
		return nil, errors.New("failed to calculate authenticated token")
	}

	// Return body value: n || c || t
	body := make([]byte, 0, len(n)+len(c)+len(t))
	body = append(body, n...)
	body = append(body, c...)
	body = append(body, t...)
	return body, nil
}

// https://tools.ietf.org/html/draft-paragon-paseto-rfc-00#section-4.3.3
func v1Decrypt(m string, f, k []byte) ([]byte, error) {
	// Get token segments
	mS := strings.Split(m, ".")
	if len(mS) < 3 {
		return nil, errors.New("invalid token")
	}

	// Verify expected header value
	h := "v1.local."
	if !strings.HasPrefix(m, h) {
		return nil, errors.New("invalid header")
	}

	// Compare footer with expected string in constant time
	if len(mS) == 4 && f != nil {
		if subtle.ConstantTimeCompare([]byte(b64.EncodeToString(f)), []byte(mS[3])) != 1 {
			return nil, errors.New("invalid footer")
		}
	}

	// Decode payload
	pld, err := b64.DecodeString(mS[2])
	if err != nil {
		return nil, errors.New("invalid token payload")
	}

	// Split token payload
	n := pld[:32]              // leftmost 32 bytes
	t := pld[len(pld)-48:]     // rightmost 48 bytes
	c := pld[32 : len(pld)-48] // middle remainder of the payload, excluding "n" and "t"

	// Split key into an Encryption key ("ek") and an Authentication key ("ak"), using the
	// leftmost 16 bytes of "n" as the HKDF salt.
	ek, ak, err := splitKey(k, n[:16])
	if err != nil {
		return nil, errors.New("failed to generate encryption keys")
	}

	// Pack "h", "n", "c", and "f" together (in that order) using PAE
	preAuth := pae([]byte(h), n, c, f)

	// Recalculate HMAC-SHA-384 of "preAuth" using "ak" as the key
	t2, err := ah(preAuth, ak)
	if err != nil {
		return nil, errors.New("failed to calculate pre-auth token")
	}

	// Compare "t" with "t2" in constant-time
	if subtle.ConstantTimeCompare(t, t2) != 1 {
		return nil, errors.New("invalid pre-auth token")
	}

	// Decrypt "c" using "AES-256-CTR", using "ek" as the key and the
	// rightmost 16 bytes of "n" as the nonce. Return result.
	return aesCTR(c, ek, n[16:])
}

// https://tools.ietf.org/html/draft-paragon-paseto-rfc-00#section-4.3.4
func v1Sign(m, f []byte, sk crypto.Signer) ([]byte, error) {
	// Set header value
	h := "v1.public."

	// Footer defaults to an empty string
	if f == nil {
		f = []byte("")
	}

	// Pack "h", "m", and "f" together (in that order) using PAE
	m2 := pae([]byte(h), m, f)

	// Sign "m2" using RSA with the private key "sk"
	sig, err := sk.Sign(rand.Reader, m2, nil)
	if err != nil {
		return nil, err
	}

	// Return body value: m || sig
	body := make([]byte, 0, len(m)+len(sig))
	body = append(body, m...)
	body = append(body, sig...)
	return body, nil
}

// https://tools.ietf.org/html/draft-paragon-paseto-rfc-00#section-4.3.5
func v1Verify(sm string, f []byte, pk SigningKey) ([]byte, error) {
	// Get token segments
	smS := strings.Split(sm, ".")
	if len(smS) < 3 {
		return nil, errors.New("invalid token")
	}

	// Verify expected header value
	h := "v1.public."
	if !strings.HasPrefix(sm, h) {
		return nil, errors.New("invalid header")
	}

	// Compare footer with expected string in constant time
	if len(smS) == 4 && f != nil {
		if subtle.ConstantTimeCompare([]byte(b64.EncodeToString(f)), []byte(smS[3])) != 1 {
			return nil, errors.New("invalid footer")
		}
	}

	// Decode payload
	pld, err := b64.DecodeString(smS[2])
	if err != nil {
		return nil, errors.New("invalid token payload")
	}
	s := pld[len(pld)-256:]    // rightmost 256 bytes
	m := pld[:len(pld)-len(s)] // leftmost remainder of the payload, excluding "s"

	// Pack "h", "m", and "f" together (in that order) using PAE
	m2 := pae([]byte(h), m, f)

	// Verify signature
	if !pk.Verify(m2, s) {
		return nil, errors.New("invalid signature")
	}
	return m, nil
}

// https://tools.ietf.org/html/draft-paragon-paseto-rfc-00#section-5.3.1
func v2Encrypt(m, f, k []byte) ([]byte, error) {
	// Set header value
	h := "v2.local."

	// Footer defaults to an empty string
	if f == nil {
		f = []byte("")
	}

	// Generate 24 random bytes
	rndB := make([]byte, 24)
	if _, err := io.ReadFull(rand.Reader, rndB); err != nil {
		return nil, errors.New("failed to read random bytes")
	}

	// Calculate nonce
	n, err := v2GetNonce(m, rndB)
	if err != nil {
		return nil, errors.New("failed to calculate message nonce")
	}

	// Pack "h", "n", and "f" together (in that order) using PAE
	preAuth := pae([]byte(h), n, f)

	// Encrypt the message using XChaCha20-Poly1305
	c, err := xChaChaPoly(m, n, k, preAuth, true)
	if err != nil {
		return nil, err
	}

	// Return body value: n || c
	body := make([]byte, 0, len(n)+len(c))
	body = append(body, n...)
	body = append(body, c...)
	return body, nil
}

// https://tools.ietf.org/html/draft-paragon-paseto-rfc-00#section-5.3.2
func v2Decrypt(m string, f, k []byte) ([]byte, error) {
	// Get token segments
	mS := strings.Split(m, ".")
	if len(mS) < 3 {
		return nil, errors.New("invalid token")
	}

	// Verify expected header value
	h := "v2.local."
	if !strings.HasPrefix(m, h) {
		return nil, errors.New("invalid header")
	}

	// Compare footer with expected string in constant time
	if len(mS) == 4 && f != nil {
		if subtle.ConstantTimeCompare([]byte(b64.EncodeToString(f)), []byte(mS[3])) != 1 {
			return nil, errors.New("invalid footer")
		}
	}

	// Decode payload
	pld, err := b64.DecodeString(mS[2])
	if err != nil {
		return nil, errors.New("invalid token payload")
	}

	// Split token payload
	n := pld[:24] // leftmost 24 bytes
	c := pld[24:] // rightmost bytes, excluding n

	// Pack "h", "n" and "f" together (in that order) using PAE
	preAuth := pae([]byte(h), n, f)

	// Decrypt "c" using "XChaCha20-Poly1305"
	return xChaChaPoly(c, n, k, preAuth, false)
}

// https://tools.ietf.org/html/draft-paragon-paseto-rfc-00#section-5.3.3
func v2Sign(m, f []byte, sk crypto.Signer) ([]byte, error) {
	// Set header value
	h := "v2.public."

	// Footer defaults to an empty string
	if f == nil {
		f = []byte("")
	}

	// Pack "h", "m", and "f" together (in that order) using PAE
	m2 := pae([]byte(h), m, f)

	// Sign "m2" using Ed25519 with the private key "sk"
	sig, err := sk.Sign(rand.Reader, m2, nil)
	if err != nil {
		return nil, err
	}

	// Return body value: m || sig
	body := make([]byte, 0, len(m)+len(sig))
	body = append(body, m...)
	body = append(body, sig...)
	return body, nil
}

// https://tools.ietf.org/html/draft-paragon-paseto-rfc-00#section-5.3.4
func v2Verify(sm string, f []byte, pk SigningKey) ([]byte, error) {
	// Get token segments
	smS := strings.Split(sm, ".")
	if len(smS) < 3 {
		return nil, errors.New("invalid token")
	}

	// Verify expected header value
	h := "v2.public."
	if !strings.HasPrefix(sm, h) {
		return nil, errors.New("invalid header")
	}

	// Compare footer with expected string in constant time
	if len(smS) == 4 && f != nil {
		if subtle.ConstantTimeCompare([]byte(b64.EncodeToString(f)), []byte(smS[3])) != 1 {
			return nil, errors.New("invalid footer")
		}
	}

	// Decode payload
	pld, err := b64.DecodeString(smS[2])
	if err != nil {
		return nil, errors.New("invalid token payload")
	}
	s := pld[len(pld)-64:]     // rightmost 64 bytes
	m := pld[:len(pld)-len(s)] // leftmost remainder of the payload, excluding "s"

	// Footer defaults to an empty string
	if f == nil {
		f = []byte("")
	}

	// Pack "h", "m", and "f" together (in that order) using PAE
	m2 := pae([]byte(h), m, f)

	// Verify signature
	if !pk.Verify(m2, s) {
		return nil, errors.New("invalid signature")
	}
	return m, nil
}

// https://github.com/paseto-standard/paseto-spec/blob/master/docs/01-Protocol-Versions/Version3.md#encrypt
func v3Encrypt(m, f, i, k []byte) ([]byte, error) {
	// Ensure key size is 256 bits
	if len(k) != 32 {
		return nil, errors.New("invalid key size")
	}

	// Set header value
	h := "v3.local."

	// Footer defaults to an empty string
	if f == nil {
		f = []byte("")
	}

	// Implicit assertion "i" defaults to an empty string
	if i == nil {
		i = []byte("")
	}

	// Generate 32 random bytes nonce
	n := make([]byte, 32)
	if _, err := rand.Read(n); err != nil {
		return nil, errors.Errorf("failed to read random nonce: %w", err)
	}

	// Split key
	ek, ak, n2, err := v3SplitKey(k, n)
	if err != nil {
		return nil, errors.New("failed to generate encryption keys")
	}

	// Encrypt the message "m" using "AES-256-CTR", with "ek" as the key and
	// "n2" as the nonce.
	c, err := aesCTR(m, ek, n2)
	if err != nil {
		return nil, err
	}

	// Pack "h", "n", "c", "f" and "i" together (in that order) using PAE
	preAuth := pae([]byte(h), n, c, f, i)

	// Calculate authenticated version of preAuth using "ak"
	t, err := ah(preAuth, ak)
	if err != nil {
		return nil, errors.New("failed to calculate authenticated token")
	}

	// Return body value: n || c || t
	body := make([]byte, 0, len(n)+len(c)+len(t))
	body = append(body, n...)
	body = append(body, c...)
	body = append(body, t...)
	return body, nil
}

// https://github.com/paseto-standard/paseto-spec/blob/master/docs/01-Protocol-Versions/Version3.md#decrypt
func v3Decrypt(m string, f, i, k []byte) ([]byte, error) {
	// Ensure key size is 256 bits
	if len(k) != 32 {
		return nil, errors.New("invalid key size")
	}

	// Get token segments
	mS := strings.Split(m, ".")
	if len(mS) < 3 {
		return nil, errors.New("invalid token")
	}

	// Verify expected header value
	h := "v3.local."
	if !strings.HasPrefix(m, h) {
		return nil, errors.New("invalid header")
	}

	// Compare footer with expected string in constant time
	if len(mS) == 4 && f != nil {
		if subtle.ConstantTimeCompare([]byte(b64.EncodeToString(f)), []byte(mS[3])) != 1 {
			return nil, errors.New("invalid footer")
		}
	}

	// Footer defaults to an empty string
	if f == nil {
		f = []byte("")
	}

	// Implicit assertion "i" defaults to an empty string
	if i == nil {
		i = []byte("")
	}

	// Decode payload
	pld, err := b64.DecodeString(mS[2])
	if err != nil {
		return nil, errors.New("invalid token payload")
	}

	// Split token payload
	n := pld[:32]              // leftmost 32 bytes
	t := pld[len(pld)-48:]     // rightmost 48 bytes
	c := pld[32 : len(pld)-48] // middle remainder of the payload, excluding `n` and `t`

	// Split key
	ek, ak, n2, err := v3SplitKey(k, n)
	if err != nil {
		return nil, errors.New("failed to generate encryption keys")
	}

	// Pack "h", "n", "c", "f" and "i" together (in that order) using PAE
	preAuth := pae([]byte(h), n, c, f, i)

	// Recalculate authenticated version of preAuth using "ak"
	t2, err := ah(preAuth, ak)
	if err != nil {
		return nil, errors.New("failed to calculate authenticated token")
	}

	// Compare "t" with "t2" in constant-time
	if subtle.ConstantTimeCompare(t, t2) != 1 {
		return nil, errors.New("invalid pre-auth token")
	}

	// Decrypt "c" using "AES-256-CTR", using "ek" as the key and "n2" as the nonce.
	// Return result.
	return aesCTR(c, ek, n2)
}

// https://github.com/paseto-standard/paseto-spec/blob/master/docs/01-Protocol-Versions/Version3.md#sign
func v3Sign(m, f, i []byte, sk crypto.Signer) ([]byte, error) {
	// Set header value
	h := "v3.public."

	// Get compressed public key
	ek, ok := sk.(*ecdsaKey)
	if !ok {
		return nil, errors.New("invalid cryptographic key")
	}
	pk := ek.pubBytes()

	// Footer, defaults to an empty string
	if f == nil {
		f = []byte("")
	}

	// Implicit assertions, default to an empty string
	if i == nil {
		i = []byte("")
	}

	// Pack "pk", "h", "m", "f" and "i" together (in that order) using PAE
	m2 := pae(pk, []byte(h), m, f, i)

	// Sign "m2" using ECDSA with the private key "sk"
	sig, err := sk.Sign(rand.Reader, m2, nil)
	if err != nil {
		return nil, err
	}

	// Return body value: m || sig
	body := make([]byte, 0, len(m)+len(sig))
	body = append(body, m...)
	body = append(body, sig...)
	return body, nil
}

// https://github.com/paseto-standard/paseto-spec/blob/master/docs/01-Protocol-Versions/Version3.md#verify
func v3Verify(sm string, f, i []byte, pk SigningKey) ([]byte, error) {
	// Get token segments
	smS := strings.Split(sm, ".")
	if len(smS) < 3 {
		return nil, errors.New("invalid token")
	}

	// Verify expected header value
	h := "v3.public."
	if !strings.HasPrefix(sm, h) {
		return nil, errors.New("invalid header")
	}

	// Validate cryptographic key
	ek, ok := pk.(*ecdsaKey)
	if !ok {
		return nil, errors.New("invalid cryptographic key")
	}

	// Footer defaults to an empty string
	if f == nil {
		f = []byte("")
	}

	// Implicit assertion "i" defaults to an empty string
	if i == nil {
		i = []byte("")
	}

	// Compare footer with expected string in constant time
	if len(smS) == 4 && f != nil {
		if subtle.ConstantTimeCompare([]byte(b64.EncodeToString(f)), []byte(smS[3])) != 1 {
			return nil, errors.New("invalid footer")
		}
	}

	// Decode payload
	pld, err := b64.DecodeString(smS[2])
	if err != nil {
		return nil, errors.New("invalid token payload")
	}
	s := pld[len(pld)-96:]     // rightmost 96 bytes
	m := pld[:len(pld)-len(s)] // leftmost remainder of the payload, excluding "s"

	// Pack "pk", "h", "m", "f" and "i" together (in that order) using PAE
	m2 := pae(ek.pubBytes(), []byte(h), m, f, i)

	// Verify signature
	if !pk.Verify(m2, s) {
		return nil, errors.New("invalid signature")
	}
	return m, nil
}

// https://github.com/paseto-standard/paseto-spec/blob/master/docs/01-Protocol-Versions/Version4.md#encrypt
func v4Encrypt(m, f, i, k []byte) ([]byte, error) {
	// Ensure key size is 256 bits
	if len(k) != 32 {
		return nil, errors.New("invalid key size")
	}

	// Set header value
	h := "v4.local."

	// Footer defaults to an empty string
	if f == nil {
		f = []byte("")
	}

	// Implicit assertion "i" defaults to an empty string
	if i == nil {
		i = []byte("")
	}

	// Generate 32 random bytes nonce
	n := make([]byte, 32)
	if _, err := rand.Read(n); err != nil {
		return nil, errors.Errorf("failed to read random nonce: %w", err)
	}

	// Split key
	ek, ak, n2, err := v4SplitKey(k, n)
	if err != nil {
		return nil, errors.New("failed to generate encryption keys")
	}

	// Encrypt the message "m" using XChaCha20, with "ek" as the key and
	// "n2" as the nonce.
	c, err := xChaCha20(m, n2, ek)
	if err != nil {
		return nil, err
	}

	// Pack "h", "n", "c", "f" and "i" together (in that order) using PAE
	preAuth := pae([]byte(h), n, c, f, i)

	// Calculate 32 byte authenticated version of preAuth using "ak"
	t, err := bh(32, ak, preAuth)
	if err != nil {
		return nil, errors.New("failed to calculate authenticated token")
	}

	// Return body value: n || c || t
	body := make([]byte, 0, len(n)+len(c)+len(t))
	body = append(body, n...)
	body = append(body, c...)
	body = append(body, t...)
	return body, nil
}

// https://github.com/paseto-standard/paseto-spec/blob/master/docs/01-Protocol-Versions/Version4.md#decrypt
func v4Decrypt(m string, f, i, k []byte) ([]byte, error) {
	// Ensure key size is 256 bits
	if len(k) != 32 {
		return nil, errors.New("invalid key size")
	}

	// Get token segments
	mS := strings.Split(m, ".")
	if len(mS) < 3 {
		return nil, errors.New("invalid token")
	}

	// Verify expected header value
	h := "v4.local."
	if !strings.HasPrefix(m, h) {
		return nil, errors.New("invalid header")
	}

	// Compare footer with expected string in constant time
	if len(mS) == 4 && f != nil {
		if subtle.ConstantTimeCompare([]byte(b64.EncodeToString(f)), []byte(mS[3])) != 1 {
			return nil, errors.New("invalid footer")
		}
	}

	// Footer defaults to an empty string
	if f == nil {
		f = []byte("")
	}

	// Implicit assertion "i" defaults to an empty string
	if i == nil {
		i = []byte("")
	}

	// Decode payload
	pld, err := b64.DecodeString(mS[2])
	if err != nil {
		return nil, errors.New("invalid token payload")
	}

	// Split token payload
	n := pld[:32]              // leftmost 32 bytes
	t := pld[len(pld)-32:]     // rightmost 32 bytes
	c := pld[32 : len(pld)-32] // middle remainder of the payload, excluding `n` and `t`

	// Split key
	ek, ak, n2, err := v4SplitKey(k, n)
	if err != nil {
		return nil, errors.New("failed to generate encryption keys")
	}

	// Pack "h", "n", "c", "f" and "i" together (in that order) using PAE
	preAuth := pae([]byte(h), n, c, f, i)

	// Recalculate authenticated version of preAuth using "ak"
	t2, err := bh(32, ak, preAuth)
	if err != nil {
		return nil, errors.New("failed to calculate authenticated token")
	}

	// Compare "t" with "t2" in constant-time
	if subtle.ConstantTimeCompare(t, t2) != 1 {
		return nil, errors.New("invalid pre-auth token")
	}

	// Decrypt "c" using "XChaCha20", using "ek" as the key and "n2" as the nonce.
	// Return result.
	return xChaCha20(c, n2, ek)
}

// https://github.com/paseto-standard/paseto-spec/blob/master/docs/01-Protocol-Versions/Version4.md#sign
func v4Sign(m, f, i []byte, sk crypto.Signer) ([]byte, error) {
	// Set header value
	h := "v4.public."

	// Footer, defaults to an empty string
	if f == nil {
		f = []byte("")
	}

	// Implicit assertions, default to an empty string
	if i == nil {
		i = []byte("")
	}

	// Pack "h", "m", "f" and "i" together (in that order) using PAE
	m2 := pae([]byte(h), m, f, i)

	// Sign "m2" using Ed25519 with the private key "sk"
	sig, err := sk.Sign(rand.Reader, m2, nil)
	if err != nil {
		return nil, err
	}

	// Return body value: m || sig
	body := make([]byte, 0, len(m)+len(sig))
	body = append(body, m...)
	body = append(body, sig...)
	return body, nil
}

// https://github.com/paseto-standard/paseto-spec/blob/master/docs/01-Protocol-Versions/Version4.md#verify
func v4Verify(sm string, f, i []byte, pk SigningKey) ([]byte, error) {
	// Get token segments
	smS := strings.Split(sm, ".")
	if len(smS) < 3 {
		return nil, errors.New("invalid token")
	}

	// Verify expected header value
	h := "v4.public."
	if !strings.HasPrefix(sm, h) {
		return nil, errors.New("invalid header")
	}

	// Compare footer with expected string in constant time
	if len(smS) == 4 && f != nil {
		if subtle.ConstantTimeCompare([]byte(b64.EncodeToString(f)), []byte(smS[3])) != 1 {
			return nil, errors.New("invalid footer")
		}
	}

	// Footer defaults to an empty string
	if f == nil {
		f = []byte("")
	}

	// Implicit assertion "i" defaults to an empty string
	if i == nil {
		i = []byte("")
	}

	// Decode payload
	pld, err := b64.DecodeString(smS[2])
	if err != nil {
		return nil, errors.New("invalid token payload")
	}
	s := pld[len(pld)-64:]     // rightmost 64 bytes
	m := pld[:len(pld)-len(s)] // leftmost remainder of the payload, excluding "s"

	// Pack "h", "m", "f" and "i" together (in that order) using PAE
	m2 := pae([]byte(h), m, f, i)

	// Verify signature
	if !pk.Verify(m2, s) {
		return nil, errors.New("invalid signature")
	}
	return m, nil
}
