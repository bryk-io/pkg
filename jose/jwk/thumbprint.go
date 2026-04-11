package jwk

import (
	"crypto"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"go.bryk.io/pkg/errors"
)

// thumbprintFields defines the required JWK members for each key type
// that must be included in the thumbprint computation per RFC 7638.
// The members are listed in lexicographic order as required by the spec.
var thumbprintFields = map[string][]string{
	keyTypeRSA: {"e", "kty", "n"},
	keyTypeEC:  {"crv", "kty", "x", "y"},
	keyTypeOct: {"k", "kty"},
	keyTypePSS: {"e", "kty", "n"},   // Same as RSA
	keyTypeOKP: {"crv", "kty", "x"}, // For EdDSA/X25519/X448
}

// ParseThumbprintURI parses a thumbprint URI and extracts the hash algorithm and thumbprint.
// The URI format is: urn:ietf:params:oauth:jwk-thumbprint:<hash-alg>:<base64url-thumbprint>.
func ParseThumbprintURI(uri string) (hashAlgo string, thumbprint string, err error) {
	const prefix = "urn:ietf:params:oauth:jwk-thumbprint:"

	if !strings.HasPrefix(uri, prefix) {
		return "", "", errors.New("invalid thumbprint URI format: missing prefix")
	}

	// Remove prefix
	remaining := strings.TrimPrefix(uri, prefix)

	// Split into algorithm and thumbprint
	parts := strings.SplitN(remaining, ":", 2)
	if len(parts) != 2 {
		return "", "", errors.New("invalid thumbprint URI format: missing hash algorithm or thumbprint")
	}

	hashAlgo = parts[0]
	thumbprint = parts[1]

	// Validate thumbprint is valid base64url
	if _, err := b64.DecodeString(thumbprint); err != nil {
		return "", "", errors.Wrap(err, "invalid thumbprint encoding in URI")
	}

	return hashAlgo, thumbprint, nil
}

// Thumbprint computes the RFC 7638 JWK Thumbprint using the specified hash function.
//
// The thumbprint is computed as follows:
//  1. Extract required members for the key type (e, kty, n for RSA; crv, kty, x, y for EC; etc.)
//  2. Order members lexicographically by name
//  3. Create canonical JSON with no whitespace and no escaping
//  4. Hash the UTF-8 representation of the JSON
//  5. Return base64url-encoded hash
//
// For private keys, the thumbprint is computed on the corresponding public key.
// This ensures that public and private keys of the same key pair have the same thumbprint.
//
// Supported hash functions: SHA-256, SHA-384, SHA-512.
func (r *Record) Thumbprint(hash crypto.Hash) (string, error) {
	if !hash.Available() {
		return "", errors.Errorf("hash function %v is not available", hash)
	}

	// Get required fields for this key type
	fields, ok := thumbprintFields[r.KeyType]
	if !ok {
		return "", errors.Errorf("unsupported key type for thumbprint: %s", r.KeyType)
	}

	// Build canonical JSON with required fields in lexicographic order
	canonicalJSON, err := r.buildCanonicalJSON(fields)
	if err != nil {
		return "", errors.Wrap(err, "failed to build canonical JSON")
	}

	// Compute hash
	h := hash.New()
	if _, err := h.Write([]byte(canonicalJSON)); err != nil {
		return "", errors.Wrap(err, "failed to compute hash")
	}
	digest := h.Sum(nil)

	// Return base64url-encoded digest
	return b64.EncodeToString(digest), nil
}

// ThumbprintURI creates a thumbprint URI according to RFC 7638.
// The format is: urn:ietf:params:oauth:jwk-thumbprint:<hash-alg>:<base64url-thumbprint>
//
// Example: urn:ietf:params:oauth:jwk-thumbprint:sha-256:NzbLsXh8uDCcd-6MNwXF4W_7noWXFZAfHkxZsRGC9Xs.
func (r *Record) ThumbprintURI(hash crypto.Hash) (string, error) {
	thumbprint, err := r.Thumbprint(hash)
	if err != nil {
		return "", err
	}

	hashName := hashAlgorithmName(hash)
	if hashName == "" {
		return "", errors.Errorf("unsupported hash algorithm for URI: %v", hash)
	}

	return fmt.Sprintf("urn:ietf:params:oauth:jwk-thumbprint:%s:%s", hashName, thumbprint), nil
}

// MatchThumbprint checks if this JWK matches the given thumbprint.
func (r *Record) MatchThumbprint(expectedThumbprint string, hash crypto.Hash) (bool, error) {
	actualThumbprint, err := r.Thumbprint(hash)
	if err != nil {
		return false, err
	}

	return actualThumbprint == expectedThumbprint, nil
}

// MatchThumbprintURI checks if this JWK matches the given thumbprint URI.
func (r *Record) MatchThumbprintURI(uri string) (bool, error) {
	hashAlgo, expectedThumbprint, err := ParseThumbprintURI(uri)
	if err != nil {
		return false, err
	}

	hash, err := hashFromAlgorithmName(hashAlgo)
	if err != nil {
		return false, err
	}

	return r.MatchThumbprint(expectedThumbprint, hash)
}

// buildCanonicalJSON creates the canonical JSON representation for thumbprint computation.
// Per RFC 7638:
//   - Only required members are included
//   - Members are ordered lexicographically by name
//   - No whitespace or line breaks
//   - No escaping of characters (ASCII only)
//   - Integer values are represented without fraction or exponent parts
func (r *Record) buildCanonicalJSON(fields []string) (string, error) {
	// Create a map with only the required fields
	params := make(map[string]string)

	for _, field := range fields {
		var value string
		switch field {
		case "kty":
			value = r.KeyType
		case "n":
			value = r.N
		case "e":
			value = r.E
		case "k":
			value = r.K
		case "crv":
			value = r.Crv
		case "x":
			value = r.X
		case "y":
			value = r.Y
		default:
			return "", errors.Errorf("unknown thumbprint field: %s", field)
		}

		if value == "" {
			return "", errors.Errorf("missing required field for thumbprint: %s", field)
		}

		params[field] = value
	}

	// Build canonical JSON manually to ensure exact format
	// RFC 7638 requires: no whitespace, lexicographic order, no escaping
	var sb strings.Builder
	sb.WriteByte('{')

	// Sort field names to ensure lexicographic order
	sortedFields := make([]string, len(fields))
	copy(sortedFields, fields)
	sort.Strings(sortedFields)

	for i, field := range sortedFields {
		if i > 0 {
			sb.WriteByte(',')
		}
		// Write field name (always ASCII)
		sb.WriteByte('"')
		sb.WriteString(field)
		sb.WriteByte('"')
		sb.WriteByte(':')
		// Write value (already base64url, so ASCII-only)
		sb.WriteByte('"')
		sb.WriteString(params[field])
		sb.WriteByte('"')
	}

	sb.WriteByte('}')

	return sb.String(), nil
}

// ThumbprintBytes computes the RFC 7638 JWK Thumbprint and returns the raw bytes.
// This is useful when you need the raw digest rather than the base64url-encoded string.
func (r *Record) ThumbprintBytes(hash crypto.Hash) ([]byte, error) {
	if !hash.Available() {
		return nil, errors.Errorf("hash function %v is not available", hash)
	}

	// Get required fields for this key type
	fields, ok := thumbprintFields[r.KeyType]
	if !ok {
		return nil, errors.Errorf("unsupported key type for thumbprint: %s", r.KeyType)
	}

	// Build canonical JSON
	canonicalJSON, err := r.buildCanonicalJSON(fields)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build canonical JSON")
	}

	// Compute and return hash
	h := hash.New()
	if _, err := h.Write([]byte(canonicalJSON)); err != nil {
		return nil, errors.Wrap(err, "failed to compute hash")
	}

	return h.Sum(nil), nil
}

// String returns a string representation of the JWK thumbprint (SHA-256).
// This implements the fmt.Stringer interface.
func (r *Record) String() string {
	thumbprint, err := r.Thumbprint(crypto.SHA256)
	if err != nil {
		return fmt.Sprintf("<invalid JWK: %v>", err)
	}
	return thumbprint
}

// MarshalJSON implements custom JSON marshaling that includes the thumbprint as "kid"
// if no KeyID is already set.
func (r *Record) MarshalJSON() ([]byte, error) {
	// Use a type alias to avoid infinite recursion
	type RecordAlias Record
	alias := (*RecordAlias)(r)

	// If no KeyID is set, try to compute one from thumbprint
	if r.KeyID == "" {
		if thumbprint, err := r.Thumbprint(crypto.SHA256); err == nil {
			r.KeyID = thumbprint
		}
	}

	return json.Marshal(alias)
}

// hashAlgorithmName returns the algorithm name for use in URIs.
func hashAlgorithmName(hash crypto.Hash) string {
	switch hash {
	case crypto.SHA256:
		return "sha-256"
	case crypto.SHA384:
		return "sha-384"
	case crypto.SHA512:
		return "sha-512"
	default:
		return ""
	}
}

// hashFromAlgorithmName returns the crypto.Hash from an algorithm name.
func hashFromAlgorithmName(name string) (crypto.Hash, error) {
	switch name {
	case "sha-256":
		return crypto.SHA256, nil
	case "sha-384":
		return crypto.SHA384, nil
	case "sha-512":
		return crypto.SHA512, nil
	default:
		return 0, errors.Errorf("unsupported hash algorithm: %s", name)
	}
}
