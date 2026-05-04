package jwk

const fieldKTY = "kty"

// Constants for "use" (public key use) parameter values as defined in RFC 7517.
const (
	// UseSignature indicates a key is used for digital signature or MAC.
	UseSignature = "sig"
	// UseEncryption indicates a key is used for encryption.
	UseEncryption = "enc"
)

// Constants for "key_ops" parameter values as defined in RFC 7517.
const (
	// KeyOpSign is used to compute digital signature or MAC.
	KeyOpSign = "sign"
	// KeyOpVerify is used to verify digital signature or MAC.
	KeyOpVerify = "verify"
	// KeyOpEncrypt is used to encrypt content.
	KeyOpEncrypt = "encrypt"
	// KeyOpDecrypt is used to decrypt content.
	KeyOpDecrypt = "decrypt"
	// KeyOpWrapKey is used to encrypt a key.
	KeyOpWrapKey = "wrapKey"
	// KeyOpUnwrapKey is used to decrypt a key.
	KeyOpUnwrapKey = "unwrapKey"
	// KeyOpDeriveKey is used to derive a key.
	KeyOpDeriveKey = "deriveKey"
	// KeyOpDeriveBits is used to derive bits not to be used as a key.
	KeyOpDeriveBits = "deriveBits"
)

// Constants for "kty" (key type) parameter values as defined in RFC 7517 and JWA.
const (
	// keyTypeRSA indicates an RSA key.
	keyTypeRSA = "RSA"
	// keyTypeEC indicates an Elliptic Curve key.
	keyTypeEC = "EC"
	// keyTypeOct indicates a symmetric (octet sequence) key.
	keyTypeOct = "oct"
	// keyTypeOKP indicates an Octet Key Pair (for EdDSA, X25519, X448).
	keyTypeOKP = "OKP"
	// keyTypePSS indicates an RSA-PSS key (non-standard but used internally).
	keyTypePSS = "PSS"
)
