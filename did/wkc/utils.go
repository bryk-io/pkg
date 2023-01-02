package wkc

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"

	"go.bryk.io/pkg/did"
	"go.bryk.io/pkg/errors"
	"go.bryk.io/x/jose/jwk"
	"go.bryk.io/x/jose/jwt"
)

// Base64 encoding used consistently by all standard keys.
var b64 = base64.RawURLEncoding

// RandomID returns a short random ID string.
func RandomID() string {
	seed := make([]byte, 6)
	_, _ = rand.Read(seed)
	return fmt.Sprintf("%X-%X", seed[:3], seed[3:])
}

// Converts an existing RSA private key to a valid JWK representation.
func toJWK(id *did.Identifier, vm string) (jwk.Key, error) {
	key := id.VerificationMethod(vm)
	if key == nil {
		return nil, errors.New("invalid verification method name")
	}
	pb, _ := pem.Decode(key.Private)
	priv, err := x509.ParsePKCS1PrivateKey(pb.Bytes)
	if err != nil {
		return nil, errors.New("invalid verification method")
	}
	return jwk.Import(jwk.Record{
		KeyID:   vm,
		KeyType: "PSS",
		Use:     "sig",
		Alg:     "PS256",
		KeyOps:  []string{"verify", "sign"},
		N:       b64.EncodeToString(priv.PublicKey.N.Bytes()),
		E:       b64.EncodeToString(big.NewInt(int64(priv.PublicKey.E)).Bytes()),
		D:       b64.EncodeToString(priv.D.Bytes()),
		P:       b64.EncodeToString(priv.Primes[0].Bytes()),
		Q:       b64.EncodeToString(priv.Primes[1].Bytes()),
		DP:      b64.EncodeToString(priv.Precomputed.Dp.Bytes()),
		DQ:      b64.EncodeToString(priv.Precomputed.Dq.Bytes()),
		Qi:      b64.EncodeToString(priv.Precomputed.Qinv.Bytes()),
	})
}

// IssuerCheck validates the "iss" claim.
func jwtDomainCheck(domain string) jwt.Check {
	return func(token *jwt.Token) error {
		val, err := token.Get("/domain")
		if err != nil {
			return errors.New("invalid 'domain' claim")
		}
		if domain != fmt.Sprintf("%s", val) {
			return errors.New("invalid 'domain' claim")
		}
		return nil
	}
}
