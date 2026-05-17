package paseto

import (
	"time"

	"go.bryk.io/pkg/errors"
)

// RegisteredClaims contains required parameters used to populate the required/internal
// fields defined by the specification.
//
//	iss (issuer)      - identifies the principal that issued the token
//	sub (subject)     - identifies the principal that is the subject of the token
//	aud (audience)    - identifies the recipients that the token is intended for
//	exp (expiration)  - identifies the time on or after which the token MUST NOT be accepted
//	nbf (not before)  - identifies the time before which the token MUST NOT be accepted
//	iat (issued at)   - identifies the time at which the token was issued
//	jti (token ID)    - provides a unique identifier for the token
//	amr (auth method) - identifiers for authentication methods used
//
// https://tools.ietf.org/html/draft-paragon-paseto-rfc-00#section-6.1
type RegisteredClaims struct {
	// The principal that issued the token.
	Issuer string `json:"iss,omitempty"`

	// The principal that is the subject of the token.
	Subject string `json:"sub,omitempty"`

	// The recipients that the token is intended for.
	Audience []string `json:"aud,omitempty"`

	// The expiration time on or after which the token MUST NOT be accepted for processing.
	ExpirationTime string `json:"exp,omitempty"`

	// The time before which the token MUST NOT be accepted for processing.
	NotBefore string `json:"nbf,omitempty"`

	// The time at which the token was issued.
	IssuedAt string `json:"iat,omitempty"`

	// Unique token identifier.
	JTI string `json:"jti,omitempty"`

	// Authentication Method Reference as described by the RFC-8176 specification.
	// https://tools.ietf.org/html/rfc8176
	AuthenticationMethod []string `json:"amr,omitempty"`
}

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
	// ErrAmrValidation is the error for an invalid "amr" claim.
	ErrAmrValidation = "amr claim is invalid"
)

// IssuerCheck validates the "iss" claim.
func IssuerCheck(iss string) Check {
	return func(token *Token) error {
		rc, err := token.RegisteredClaims()
		if err != nil {
			return err
		}
		if rc.Issuer != iss {
			return errors.New(ErrIssValidation)
		}
		return nil
	}
}

// AudienceCheck validates the "aud" claim.
// It checks if at least one of the audiences in the token's payload is listed in aud.
func AudienceCheck(aud []string) Check {
	return func(token *Token) error {
		rc, err := token.RegisteredClaims()
		if err != nil {
			return err
		}
		for _, serverAud := range aud {
			for _, clientAud := range rc.Audience {
				if clientAud == serverAud {
					return nil
				}
			}
		}
		return errors.New(ErrAudValidation)
	}
}

// IDCheck validates the "jti" claim.
func IDCheck(jti string) Check {
	return func(token *Token) error {
		rc, err := token.RegisteredClaims()
		if err != nil {
			return err
		}
		if rc.JTI != jti {
			return errors.New(ErrJtiValidation)
		}
		return nil
	}
}

// SubjectCheck validates the "sub" claim.
func SubjectCheck(sub string) Check {
	return func(token *Token) error {
		rc, err := token.RegisteredClaims()
		if err != nil {
			return err
		}
		if rc.Subject != sub {
			return errors.New(ErrSubValidation)
		}
		return nil
	}
}

// AuthenticationMethodCheck validates the "amr" claim contains the
// provided authentication reference value.
func AuthenticationMethodCheck(amr string) Check {
	return func(token *Token) error {
		rc, err := token.RegisteredClaims()
		if err != nil {
			return err
		}
		for _, el := range rc.AuthenticationMethod {
			if el == amr {
				return nil
			}
		}
		return errors.New(ErrAmrValidation)
	}
}

// ExpirationTimeCheck validates the "exp" claim.
func ExpirationTimeCheck(now time.Time) Check {
	return func(token *Token) error {
		rc, err := token.RegisteredClaims()
		if err != nil {
			return err
		}
		if rc.ExpirationTime == "" {
			return errors.New(ErrExpValidation)
		}
		exp, err := time.Parse(timeFormat, rc.ExpirationTime)
		if err != nil {
			return errors.New(ErrExpValidation)
		}
		if now.After(exp) {
			return errors.New(ErrExpValidation)
		}
		return nil
	}
}

// IssuedAtCheck validates the "iat" claim.
func IssuedAtCheck(now time.Time) Check {
	return func(token *Token) error {
		rc, err := token.RegisteredClaims()
		if err != nil {
			return err
		}
		if rc.IssuedAt == "" {
			return errors.New(ErrIatValidation)
		}
		iat, err := time.Parse(timeFormat, rc.IssuedAt)
		if err != nil {
			return errors.New(ErrIatValidation)
		}
		if now.Before(iat) {
			return errors.New(ErrIatValidation)
		}
		return nil
	}
}

// NotBeforeCheck validates the "nbf" claim.
func NotBeforeCheck(now time.Time) Check {
	return func(token *Token) error {
		rc, err := token.RegisteredClaims()
		if err != nil {
			return err
		}
		if rc.NotBefore == "" {
			return errors.New(ErrNbfValidation)
		}
		nbf, err := time.Parse(timeFormat, rc.NotBefore)
		if err != nil {
			return errors.New(ErrNbfValidation)
		}
		if now.Before(nbf) {
			return errors.New(ErrNbfValidation)
		}
		return nil
	}
}
