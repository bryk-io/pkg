package jwk

import (
	"encoding/json"

	"go.bryk.io/pkg/errors"
)

// Set is an object that represents a collection of "JSON Web Keys".
// https://www.rfc-editor.org/rfc/rfc7517.html#section-5
type Set struct {
	// The value of the "keys" parameter is an array of JWK values.
	// By default, the order of the JWK values within the array does
	// not imply an order of preference among them.
	Keys []Record `json:"keys" yaml:"keys" mapstructure:"keys"`
}

// ParseSet parses a JWK Set from JSON data.
func ParseSet(data []byte) (*Set, error) {
	var set Set
	if err := json.Unmarshal(data, &set); err != nil {
		return nil, errors.Wrap(err, "failed to parse JWK Set")
	}
	if err := set.Validate(); err != nil {
		return nil, err
	}
	return &set, nil
}

// FindByID returns a key from the set with the matching KeyID (kid).
// If no matching key is found, returns nil and false.
func (s *Set) FindByID(kid string) (*Record, bool) {
	for i := range s.Keys {
		if s.Keys[i].KeyID == kid {
			return &s.Keys[i], true
		}
	}
	return nil, false
}

// Find returns the first key that matches all provided selectors.
// If no matching key is found, returns nil and false.
func (s *Set) Find(selectors ...KeySelector) (*Record, bool) {
	for i := range s.Keys {
		match := true
		for _, selector := range selectors {
			if !selector(s.Keys[i]) {
				match = false
				break
			}
		}
		if match {
			return &s.Keys[i], true
		}
	}
	return nil, false
}

// Filter returns all keys that match all provided selectors.
// If no selectors are provided, returns all keys.
func (s *Set) Filter(selectors ...KeySelector) []Record {
	if len(selectors) == 0 {
		return s.Keys
	}
	var result []Record
	for _, key := range s.Keys {
		match := true
		for _, selector := range selectors {
			if !selector(key) {
				match = false
				break
			}
		}
		if match {
			result = append(result, key)
		}
	}
	return result
}

// Add adds a key to the set. Returns an error if the key is invalid
// or if a key with the same KeyID already exists (when checkDuplicates is true).
func (s *Set) Add(key Record, checkDuplicates bool) error {
	if err := key.Validate(); err != nil {
		return errors.Wrap(err, "invalid key")
	}
	if checkDuplicates && key.KeyID != "" {
		if _, found := s.FindByID(key.KeyID); found {
			return errors.Errorf("key with ID '%s' already exists", key.KeyID)
		}
	}
	s.Keys = append(s.Keys, key)
	return nil
}

// Remove removes the key with the specified KeyID from the set.
// Returns true if a key was removed, false otherwise.
func (s *Set) Remove(kid string) bool {
	for i, key := range s.Keys {
		if key.KeyID == kid {
			s.Keys = append(s.Keys[:i], s.Keys[i+1:]...)
			return true
		}
	}
	return false
}

// Len returns the number of keys in the set.
func (s *Set) Len() int {
	return len(s.Keys)
}

// IsEmpty returns true if the set contains no keys.
func (s *Set) IsEmpty() bool {
	return len(s.Keys) == 0
}

// Clear removes all keys from the set.
func (s *Set) Clear() {
	s.Keys = nil
}

// Merge combines another JWK Set into this one. If checkDuplicates is true,
// keys with duplicate KeyIDs will be skipped with an error returned listing them.
func (s *Set) Merge(other *Set, checkDuplicates bool) error {
	var skipped []string
	for _, key := range other.Keys {
		if err := s.Add(key, checkDuplicates); err != nil {
			if checkDuplicates && key.KeyID != "" {
				skipped = append(skipped, key.KeyID)
				continue
			}
			return err
		}
	}
	if len(skipped) > 0 {
		return errors.Errorf("skipped duplicate keys: %v", skipped)
	}
	return nil
}

// Clone creates a deep copy of the JWK Set.
func (s *Set) Clone() *Set {
	if s == nil {
		return nil
	}
	clone := &Set{
		Keys: make([]Record, len(s.Keys)),
	}
	for i, key := range s.Keys {
		clone.Keys[i] = key.Clone()
	}
	return clone
}

// Validate checks the JWK Set for consistency and compliance with RFC 7517.
// It validates each key and checks for duplicate KeyIDs.
func (s *Set) Validate() error {
	if s == nil {
		return errors.New("JWK Set is nil")
	}
	seenKIDs := make(map[string]int)
	for i, key := range s.Keys {
		if err := key.Validate(); err != nil {
			return errors.Wrapf(err, "key at index %d is invalid", i)
		}
		if key.KeyID != "" {
			if idx, exists := seenKIDs[key.KeyID]; exists {
				return errors.Errorf("duplicate KeyID '%s' at indices %d and %d", key.KeyID, idx, i)
			}
			seenKIDs[key.KeyID] = i
		}
	}
	return nil
}

// GetKey returns the key at the specified index.
// Returns nil and false if the index is out of bounds.
func (s *Set) GetKey(index int) (*Record, bool) {
	if index < 0 || index >= len(s.Keys) {
		return nil, false
	}
	return &s.Keys[index], true
}

// KeyIDs returns a slice of all KeyIDs in the set.
func (s *Set) KeyIDs() []string {
	ids := make([]string, 0, len(s.Keys))
	for _, key := range s.Keys {
		if key.KeyID != "" {
			ids = append(ids, key.KeyID)
		}
	}
	return ids
}

// First returns the first key in the set.
// Returns nil and false if the set is empty.
func (s *Set) First() (*Record, bool) {
	if len(s.Keys) == 0 {
		return nil, false
	}
	return &s.Keys[0], true
}

// SelectByOperation returns keys suitable for the specified operation.
func (s *Set) SelectByOperation(op string) []Record {
	return s.Filter(func(r Record) bool {
		if len(r.KeyOps) == 0 {
			// If no key_ops specified, infer from use and key type
			switch op {
			case KeyOpSign:
				return r.Use == "" || r.Use == UseSignature
			case KeyOpVerify:
				return r.Use == "" || r.Use == UseSignature
			case KeyOpEncrypt, KeyOpWrapKey:
				return r.Use == "" || r.Use == UseEncryption
			case KeyOpDecrypt, KeyOpUnwrapKey:
				return r.Use == "" || r.Use == UseEncryption
			default:
				return true
			}
		}
		for _, keyOp := range r.KeyOps {
			if keyOp == op {
				return true
			}
		}
		return false
	})
}

// ! MARK: utils

// checkUseAndKeyOpsConsistency checks that use and key_ops are consistent
// according to RFC 7517 Section 4.3.
func checkUseAndKeyOpsConsistency(use string, keyOps []string) error {
	if use == "" || len(keyOps) == 0 {
		return nil
	}
	// RFC 7517: When both are used, the information they convey MUST be consistent
	// "sig" use should only have sign, verify operations
	// "enc" use should only have encrypt, decrypt, wrapKey, unwrapKey, deriveKey, deriveBits operations
	sigOps := map[string]bool{
		KeyOpSign:   true,
		KeyOpVerify: true,
	}
	encOps := map[string]bool{
		KeyOpEncrypt:    true,
		KeyOpDecrypt:    true,
		KeyOpWrapKey:    true,
		KeyOpUnwrapKey:  true,
		KeyOpDeriveKey:  true,
		KeyOpDeriveBits: true,
	}
	switch use {
	case UseSignature:
		for _, op := range keyOps {
			if !sigOps[op] {
				return errors.Errorf("inconsistent use='%s' with key_ops containing '%s'", use, op)
			}
		}
	case UseEncryption:
		for _, op := range keyOps {
			if !encOps[op] {
				return errors.Errorf("inconsistent use='%s' with key_ops containing '%s'", use, op)
			}
		}
	}
	return nil
}

// validateKeyOps validates the key_ops array according to RFC 7517.
func validateKeyOps(ops []string) error {
	validOps := map[string]bool{
		KeyOpSign:       true,
		KeyOpVerify:     true,
		KeyOpEncrypt:    true,
		KeyOpDecrypt:    true,
		KeyOpWrapKey:    true,
		KeyOpUnwrapKey:  true,
		KeyOpDeriveKey:  true,
		KeyOpDeriveBits: true,
	}
	seen := make(map[string]bool)
	for _, op := range ops {
		if !validOps[op] {
			return errors.Errorf("invalid key_ops value: %s", op)
		}
		if seen[op] {
			return errors.Errorf("duplicate key_ops value: %s", op)
		}
		seen[op] = true
	}
	return nil
}

// validateAlgorithm checks if the algorithm is valid for the given key type.
func validateAlgorithm(keyType, alg string) error {
	// Get the algorithm family from the first two characters
	if len(alg) < 2 {
		return errors.Errorf("invalid algorithm: %s", alg)
	}
	algFamily := alg[:2]
	var expectedKeyType string
	switch algFamily {
	case "HS":
		expectedKeyType = keyTypeOct
	case "RS":
		expectedKeyType = keyTypeRSA
	case "PS":
		expectedKeyType = keyTypePSS
	case "ES":
		expectedKeyType = keyTypeEC
	case "Ed":
		expectedKeyType = keyTypeOKP
	default:
		return errors.Errorf("unsupported algorithm: %s", alg)
	}
	if keyType != expectedKeyType {
		return errors.Errorf("algorithm %s requires key type %s, got %s", alg, expectedKeyType, keyType)
	}
	return nil
}
