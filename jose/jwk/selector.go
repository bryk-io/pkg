package jwk

// KeySelector is a function that determines if a key should be selected.
// It returns true if the key matches the selection criteria.
type KeySelector func(Record) bool

// MatchKid returns a KeySelector that matches keys by KeyID (kid).
// This is an alias for ByID for semantic clarity.
func MatchKid(kid string) KeySelector {
	return ByID(kid)
}

// ByUse returns a KeySelector that matches keys by use.
func ByUse(use string) KeySelector {
	return func(r Record) bool {
		return r.Use == use
	}
}

// CanSign returns a KeySelector that matches keys suitable for signing.
func CanSign() KeySelector {
	return func(r Record) bool {
		// Check use parameter - "sig" allows signing
		if r.Use != "" && r.Use != UseSignature {
			return false
		}
		// Check key_ops - must include "sign" or be empty
		if len(r.KeyOps) > 0 {
			hasSign := false
			for _, op := range r.KeyOps {
				if op == KeyOpSign {
					hasSign = true
					break
				}
			}
			if !hasSign {
				return false
			}
		}
		return true
	}
}

// CanVerify returns a KeySelector that matches keys suitable for signature verification.
func CanVerify() KeySelector {
	return func(r Record) bool {
		// Check use parameter - "sig" allows verification
		if r.Use != "" && r.Use != UseSignature {
			return false
		}
		// Check key_ops - must include "verify" or be empty
		if len(r.KeyOps) > 0 {
			hasVerify := false
			for _, op := range r.KeyOps {
				if op == KeyOpVerify {
					hasVerify = true
					break
				}
			}
			if !hasVerify {
				return false
			}
		}
		return true
	}
}

// CanEncrypt returns a KeySelector that matches keys suitable for encryption.
func CanEncrypt() KeySelector {
	return func(r Record) bool {
		// Check use parameter - "enc" allows encryption
		if r.Use != "" && r.Use != UseEncryption {
			return false
		}
		// Check key_ops - must include "encrypt" or "wrapKey" or be empty
		if len(r.KeyOps) > 0 {
			hasEncryptOp := false
			for _, op := range r.KeyOps {
				if op == KeyOpEncrypt || op == KeyOpWrapKey {
					hasEncryptOp = true
					break
				}
			}
			if !hasEncryptOp {
				return false
			}
		}
		return true
	}
}

// CanDecrypt returns a KeySelector that matches keys suitable for decryption.
func CanDecrypt() KeySelector {
	return func(r Record) bool {
		// Check use parameter - "enc" allows decryption
		if r.Use != "" && r.Use != UseEncryption {
			return false
		}
		// Check key_ops - must include "decrypt" or "unwrapKey" or be empty
		if len(r.KeyOps) > 0 {
			hasDecryptOp := false
			for _, op := range r.KeyOps {
				if op == KeyOpDecrypt || op == KeyOpUnwrapKey {
					hasDecryptOp = true
					break
				}
			}
			if !hasDecryptOp {
				return false
			}
		}
		return true
	}
}

// ByID returns a KeySelector that matches keys by KeyID (kid).
func ByID(kid string) KeySelector {
	return func(r Record) bool {
		return r.KeyID == kid
	}
}

// ByAlg returns a KeySelector that matches keys by algorithm (alg).
func ByAlg(alg string) KeySelector {
	return func(r Record) bool {
		return r.Alg == alg
	}
}

// ByKeyType returns a KeySelector that matches keys by key type (kty).
func ByKeyType(kty string) KeySelector {
	return func(r Record) bool {
		return r.KeyType == kty
	}
}
