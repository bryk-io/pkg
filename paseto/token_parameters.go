package paseto

import (
	"encoding/json"
	"time"

	"go.bryk.io/pkg/errors"
	cryptoutils "go.bryk.io/pkg/internal/crypto"
)

const (
	// ISO8601 compliant time format.
	timeFormat = "2006-01-02T15:04:05-07:00"

	// Default expiration interval value.
	defaultExpiration = "720h"

	// Default "not before" interval value.
	defaultNotBefore = "0s"
)

// TokenParameters define the settings used to create and validate tokens.
type TokenParameters struct {
	// String that represents the current version of the protocol. The version
	// used implicitly specifies the cipher-suites utilized.
	// Accepted values: v1, v2, v3, v4.
	Version string `json:"version"`

	// Short string describing the purpose of the token.
	// Accepted values: local, public
	//   local: shared-key authenticated encryption
	//   public: public-key digital signatures; not encrypted
	Purpose string `json:"purpose"`

	// Additional public and private claims to be added on the token's payload,
	// optional. If any key in the custom claims conflicts with an existing registered
	// claim, the latter will take precedence and override the custom value.
	// https://tools.ietf.org/html/draft-paragon-paseto-rfc-00#section-6
	CustomClaims interface{} `json:"payload,omitempty"`

	// Footer content, optional. If provided this claims will be JSON encoded an included
	// in the token's footer segment.
	Footer interface{} `json:"footer,omitempty"`

	// The principal that is the subject of the token, required.
	Subject string `json:"subject"`

	// Recipients that the token is intended for, required.
	Audience []string `json:"audience,omitempty"`

	// Set an expiration value for the token.
	// A duration string is a signed sequence of decimal numbers, each with optional
	// fraction and a unit suffix, such as "300ms", "1.5h" or "2h45m". Valid time units
	// are: "ns", "us" (or "µs"), "ms", "s", "m", "h"
	// Optional when generating a new token, defaults to "720h".
	Expiration string `json:"expiration,omitempty"`

	// The time before which the token MUST NOT be accepted for processing.
	// A duration string is a signed sequence of decimal numbers, each with optional
	// fraction and a unit suffix, such as "300ms", "1.5h" or "2h45m". Valid time units
	// are: "ns", "us" (or "µs"), "ms", "s", "m", "h"
	// Optional when generating a new token, defaults to "0s".
	NotBefore string `json:"not_before,omitempty"`

	// Identifier for the token instance, MUST be unique.
	// Optional when generating a new token, defaults to a random UUID v4.
	UniqueIdentifier string `json:"id,omitempty"`

	// Authentication Method Reference as described by the RFC-8176 specification.
	// https://tools.ietf.org/html/rfc8176
	AuthenticationMethod []string `json:"amr,omitempty"`

	// Set location to UTC for all date values included in the cluster
	UseUTC bool `json:"utc"`

	// PASETO v3 and v4 tokens support optional additional authenticated data that
	// IS NOT stored in the token, but IS USED to calculate the authentication tag
	// (local) or signature (public). These can be any application-specific data that
	// must be provided when validating tokens, but isn't appropriate to store in the
	// token itself (e.g. sensitive internal values).
	//
	// One example where implicit assertions might be desirable is ensuring that a
	// PASETO is only used by a specific user in a multi-tenant system. Simply providing
	// the user's account ID when minting and consuming PASETO tokens will bind the
	// token to the desired context.
	ImplicitAssertions string `json:"assertions,omitempty"`
}

// Required footer claims.
type footerClaims struct {
	// https://tools.ietf.org/html/draft-paragon-paseto-rfc-00#section-6.1.1
	KeyID string `json:"kid"`
}

// GetChecks return proper validation checks based on the parameters' setup.
func (tp *TokenParameters) GetChecks() []Check {
	now := time.Now()
	list := []Check{
		IssuedAtCheck(now),
		ExpirationTimeCheck(now),
		NotBeforeCheck(now),
	}
	if len(tp.Audience) > 0 {
		list = append(list, AudienceCheck(tp.Audience))
	}
	if tp.UniqueIdentifier != "" {
		list = append(list, IDCheck(tp.UniqueIdentifier))
	}
	if tp.Subject != "" {
		list = append(list, SubjectCheck(tp.Subject))
	}
	for _, amr := range tp.AuthenticationMethod {
		list = append(list, AuthenticationMethodCheck(amr))
	}
	return list
}

// Return a token kind identifier based on its version and purpose.
func (tp *TokenParameters) tokenType() string {
	return tp.Version + "." + tp.Purpose
}

// Merge the user's provided footer claims with the required fields
// defined by the specification.
func (tp *TokenParameters) getFooter(key string) ([]byte, error) {
	ftr, err := merge(tp.Footer, footerClaims{KeyID: key})
	if err != nil {
		return nil, errors.Errorf("failed to encode footer claims: %w", err)
	}
	return json.Marshal(ftr)
}

// Merge the user's provided custom claims with the required payload
// fields defined by the specification.
func (tp *TokenParameters) getPayload(issuer string) ([]byte, error) {
	rc, err := tp.getRegisteredClaims(issuer)
	if err != nil {
		return nil, err
	}
	pld, err := merge(tp.CustomClaims, rc)
	if err != nil {
		return nil, err
	}
	return json.Marshal(pld)
}

// Return the registered claims to include on the token.
func (tp *TokenParameters) getRegisteredClaims(issuer string) (*RegisteredClaims, error) {
	// Validate parameters
	if tp.UniqueIdentifier == "" {
		tp.UniqueIdentifier = cryptoutils.RandomID()
	}
	if tp.NotBefore == "" {
		tp.NotBefore = defaultNotBefore
	}
	if tp.Expiration == "" {
		tp.Expiration = defaultExpiration
	}

	// Parse not before and expiration intervals
	nbf, err := time.ParseDuration(tp.NotBefore)
	if err != nil {
		return nil, errors.Errorf("invalid 'NotBefore' value: %s", tp.NotBefore)
	}
	exp, err := time.ParseDuration(tp.Expiration)
	if err != nil {
		return nil, errors.Errorf("invalid 'Expiration' value: %s", tp.Expiration)
	}

	// Current date
	now := time.Now()
	if tp.UseUTC {
		now = now.UTC()
	}

	return &RegisteredClaims{
		JTI:                  tp.UniqueIdentifier,
		Issuer:               issuer,
		Subject:              tp.Subject,
		Audience:             tp.Audience,
		AuthenticationMethod: tp.AuthenticationMethod,
		IssuedAt:             now.Format(timeFormat),
		NotBefore:            now.Add(nbf).Format(timeFormat),
		ExpirationTime:       now.Add(exp).Format(timeFormat),
	}, nil
}
