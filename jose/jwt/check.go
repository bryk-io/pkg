package jwt

import (
	"time"

	"go.bryk.io/pkg/errors"
)

const (
	// ErrAudValidation is the error for an invalid "aud" claim.
	ErrAudValidation = "aud claim is invalid"
	// ErrExpValidation is the error for an invalid "exp" claim.
	ErrExpValidation = "exp claim is invalid"
	// ErrIatValidation is the error for an invalid "iat" claim.
	ErrIatValidation = "iat claim is invalid"
	// ErrIssValidation is the error for an invalid "iss" claim.
	ErrIssValidation = "iss claim is invalid"
	// ErrJtiValidation is the error for an invalid "jti" claim.
	ErrJtiValidation = "jti claim is invalid"
	// ErrNbfValidation is the error for an invalid "nbf" claim.
	ErrNbfValidation = "nbf claim is invalid"
	// ErrSubValidation is the error for an invalid "sub" claim.
	ErrSubValidation = "sub claim is invalid"
	// ErrAlgValidation is the error for an invalid "alg" header.
	ErrAlgValidation = "alg header is invalid"
	// ErrCtyValidation is the error for an invalid "cty" header.
	ErrCtyValidation = "cty header is invalid"
	// ErrAmrValidation is the error for an invalid "amr" claim.
	ErrAmrValidation = "amr claim is invalid"
)

// Check functions allow to execute verifications against a JWT instance.
type Check func(token *Token) error

// AudienceCheck validates the "aud" claim.
// It checks if at least one of the audiences in the JWT payload is listed in aud.
func AudienceCheck(aud []string) Check {
	return func(token *Token) error {
		pl, err := token.RegisteredClaims()
		if err != nil {
			return err
		}
		for _, serverAud := range aud {
			for _, clientAud := range pl.Audience {
				if clientAud == serverAud {
					return nil
				}
			}
		}
		return errors.New(ErrAudValidation)
	}
}

// ExpirationTimeCheck validates the "exp" claim.
func ExpirationTimeCheck(now time.Time, validateZero bool) Check {
	return func(token *Token) error {
		pl, err := token.RegisteredClaims()
		if err != nil {
			return err
		}
		expInt := pl.ExpirationTime
		if !validateZero && expInt == 0 {
			return nil
		}
		if exp := time.Unix(expInt, 0); now.After(exp) {
			return errors.New(ErrExpValidation)
		}
		return nil
	}
}

// IssuedAtCheck validates the "iat" claim.
func IssuedAtCheck(now time.Time) Check {
	return func(token *Token) error {
		pl, err := token.RegisteredClaims()
		if err != nil {
			return err
		}
		if iat := time.Unix(pl.IssuedAt, 0); now.Before(iat) {
			return errors.New(ErrIatValidation)
		}
		return nil
	}
}

// IssuerCheck validates the "iss" claim.
func IssuerCheck(iss string) Check {
	return func(token *Token) error {
		pl, err := token.RegisteredClaims()
		if err != nil {
			return err
		}
		if pl.Issuer != iss {
			return errors.New(ErrIssValidation)
		}
		return nil
	}
}

// IDCheck validates the "jti" claim.
func IDCheck(jti string) Check {
	return func(token *Token) error {
		pl, err := token.RegisteredClaims()
		if err != nil {
			return err
		}
		if pl.JTI != jti {
			return errors.New(ErrJtiValidation)
		}
		return nil
	}
}

// NotBeforeCheck validates the "nbf" claim.
func NotBeforeCheck(now time.Time) Check {
	return func(token *Token) error {
		pl, err := token.RegisteredClaims()
		if err != nil {
			return err
		}
		if nbf := time.Unix(pl.NotBefore, 0); now.Before(nbf) {
			return errors.New(ErrNbfValidation)
		}
		return nil
	}
}

// SubjectCheck validates the "sub" claim.
func SubjectCheck(sub string) Check {
	return func(token *Token) error {
		pl, err := token.RegisteredClaims()
		if err != nil {
			return err
		}
		if pl.Subject != sub {
			return errors.New(ErrSubValidation)
		}
		return nil
	}
}

// AlgorithmCheck validates the "alg" header.
func AlgorithmCheck(alg string) Check {
	return func(token *Token) error {
		if token.Header().Algorithm != alg {
			return errors.New(ErrAlgValidation)
		}
		return nil
	}
}

// ContentTypeCheck validates the "cty" header.
func ContentTypeCheck(cty string) Check {
	return func(token *Token) error {
		if token.Header().ContentType != cty {
			return errors.New(ErrCtyValidation)
		}
		return nil
	}
}
