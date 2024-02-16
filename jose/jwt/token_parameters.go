package jwt

import (
	"strings"
	"time"

	"go.bryk.io/pkg/errors"
	cryptoutils "go.bryk.io/pkg/internal/crypto"
	"go.bryk.io/pkg/jose/jwa"
)

// TokenParameters define the settings used to create and validate tokens.
type TokenParameters struct {
	// The principal that is the subject of the JWT, required.
	Subject string

	// Recipients that the JWT is intended for, required.
	Audience []string

	// Cryptographic method used for the token.
	// Optional when generating a new token, it will be automatically
	// set to the proper value based on the key used to sign the
	// token or to "NONE".
	Method string

	// Specify a custom content identifier in the JWT header.
	// Optional when generating a new token, not set by default.
	ContentType string

	// Set an expiration value for the JWT.
	// A duration string is a signed sequence of decimal numbers, each with optional
	// fraction and a unit suffix, such as "300ms", "1.5h" or "2h45m". Valid time units
	// are: "ns", "us" (or "µs"), "ms", "s", "m", "h"
	// Optional when generating a new token, defaults to "720h".
	Expiration string

	// The time before which the JWT MUST NOT be accepted for processing.
	// A duration string is a signed sequence of decimal numbers, each with optional
	// fraction and a unit suffix, such as "300ms", "1.5h" or "2h45m". Valid time units
	// are: "ns", "us" (or "µs"), "ms", "s", "m", "h"
	// Optional when generating a new token, defaults to "0s".
	NotBefore string

	// Identifier for the token instance, MUST be unique.
	// Optional when generating a new token, defaults to a random UUID v4.
	UniqueIdentifier string

	// Additional public and private claims to be added/expected on the JWT payload,
	// optional. If any key in the custom claims conflicts with an existing registered
	// claim, the latter will take precedence and override the custom value.
	CustomClaims interface{}

	// Produced when parsing 'NotBefore'.
	nbf time.Duration

	// Produced when parsing 'Expiration'.
	exp time.Duration
}

// GetChecks return a collection of standard validations based on the parameters.
func (tp *TokenParameters) GetChecks() []Check {
	// Basic payload validations
	now := time.Now()
	checks := []Check{
		// 'alg' header validation
		AlgorithmCheck(tp.Method),
		// 'iat' validation
		IssuedAtCheck(now),
		// 'nbf' validation
		NotBeforeCheck(now),
		// 'exp' validation
		ExpirationTimeCheck(now, true),
	}
	// 'sub' validation
	if tp.Subject != "" {
		checks = append(checks, SubjectCheck(tp.Subject))
	}
	// 'aud' validation
	if len(tp.Audience) > 0 {
		checks = append(checks, AudienceCheck(tp.Audience))
	}
	// 'jti' validation
	if tp.UniqueIdentifier != "" {
		checks = append(checks, IDCheck(tp.UniqueIdentifier))
	}
	// 'cty' header validation
	if tp.ContentType != "" {
		checks = append(checks, ContentTypeCheck(tp.ContentType))
	}
	return checks
}

// Verify the parameters instance is valid. Set default values as required.
func (tp *TokenParameters) verify() error {
	// Verify subject is provided
	if strings.TrimSpace(tp.Subject) == "" {
		return errors.New("'subject' is a required parameter")
	}

	// Verify audience is provided
	if len(tp.Audience) == 0 {
		return errors.New("'audience' is a required parameter")
	}

	// Default method to "NONE" if not provided
	if strings.TrimSpace(tp.Method) == "" {
		tp.Method = string(jwa.NONE)
	}

	// Unique identifier
	if strings.TrimSpace(tp.UniqueIdentifier) == "" {
		tp.UniqueIdentifier = cryptoutils.RandomID()
	}

	// Expiration time
	if strings.TrimSpace(tp.Expiration) == "" {
		tp.Expiration = "720h"
	}
	exp, err := time.ParseDuration(tp.Expiration)
	if err != nil {
		return errors.New("invalid 'Expiration' value")
	}
	tp.exp = exp

	// Not before time
	if strings.TrimSpace(tp.NotBefore) == "" {
		tp.NotBefore = "0s"
	}
	nbf, err := time.ParseDuration(tp.NotBefore)
	if err != nil {
		return errors.New("invalid 'NotBefore' value")
	}
	tp.nbf = nbf
	return nil
}
