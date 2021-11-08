package did

import (
	"encoding/json"
	"fmt"
)

// KeyType provide a valid values when specifying a cryptographic public key.
type KeyType int

const (
	// KeyTypeEd specify a Ed25519 public key.
	// https://w3c-dvcg.github.io/lds-ed25519-2018/
	KeyTypeEd KeyType = iota

	// KeyTypeRSA specify a RSA public key.
	// https://w3c-dvcg.github.io/lds-rsa2018/
	KeyTypeRSA

	// KeyTypeSecp256k1 specify an ECDSA secp256k1 public key.
	// https://w3c-dvcg.github.io/lds-ecdsa-secp256k1-2019/
	KeyTypeSecp256k1
)

// String returns the value identifier for a given key type value.
func (v KeyType) String() string {
	values := [...]string{
		"Ed25519VerificationKey2020",
		"RsaVerificationKey2018",
		"EcdsaSecp256k1VerificationKey2019",
	}
	if int(v) > len(values) {
		return "unknown key type"
	}
	return values[v]
}

// SignatureType returns the value identifier for the kind of signature generated
// by the key.
func (v KeyType) SignatureType() string {
	values := [...]string{
		"Ed25519Signature2020",
		"RsaSignature2018",
		"EcdsaSecp256k1Signature2019",
	}
	if int(v) > len(values) {
		return "unknown signature type"
	}
	return values[v]
}

// MarshalJSON provides custom encoding implementation.
func (v *KeyType) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.String())
}

// UnmarshalJSON provides custom decoding implementation.
func (v *KeyType) UnmarshalJSON(b []byte) error {
	var s string
	var err error
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	*v, err = keyTypeFromString(s)
	if err != nil {
		return err
	}
	return nil
}

// MarshalYAML provides custom YAML encoding.
func (v KeyType) MarshalYAML() (interface{}, error) {
	return v.String(), nil
}

// UnmarshalYAML provides custom YAML decoding.
func (v *KeyType) UnmarshalYAML(unmarshal func(v interface{}) error) error {
	var s string
	var err error
	if err = unmarshal(&s); err != nil {
		return err
	}
	*v, err = keyTypeFromString(s)
	if err != nil {
		return err
	}
	return nil
}

func keyTypeFromString(val string) (kt KeyType, err error) {
	switch val {
	case KeyTypeEd.String():
		kt = KeyTypeEd
		return
	case KeyTypeRSA.String():
		kt = KeyTypeRSA
		return
	case KeyTypeSecp256k1.String():
		kt = KeyTypeSecp256k1
		return
	default:
		err = fmt.Errorf("unknown key type: %s", val)
		return
	}
}
