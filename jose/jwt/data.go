package jwt

// Header is based on the JWT specification from RFC-7519.
//
//	typ (type)         - media type
//	alg (algorithm)    - cryptographic algorithm used to generate the token
//	cty (content type) - used to convey structural information about the token
//	kid (key id)       - identifier for the cryptographic key used to sign the token
type Header struct {
	// Declare the media type of this complete JWT.
	Type string `json:"typ,omitempty"`

	// Specify the cryptographic algorithm used to generate the JWT.
	Algorithm string `json:"alg,omitempty"`

	// Used to convey structural information about the JWT.
	ContentType string `json:"cty,omitempty"`

	// An optional identifier for the cryptographic key used to generate the JWT.
	KeyID string `json:"kid,omitempty"`
}

// RegisteredClaims for the JWT payload section according to the RFC-7519.
//
//	iss (issuer)      - identifies the principal that issued the JWT
//	sub (subject)     - identifies the principal that is the subject of the JWT
//	aud (audience)    - identifies the recipients that the JWT is intended for
//	exp (expiration)  - identifies the time on or after which the JWT MUST NOT be accepted
//	nbf (not before)  - identifies the time before which the JWT MUST NOT be accepted
//	iat (issued at)   - identifies the time at which the JWT was issued
//	jti (JWT ID)      - provides a unique identifier for the JWT
//
// More information: https://tools.ietf.org/html/rfc7519#section-4.1
type RegisteredClaims struct {
	// The principal that issued the JWT.
	Issuer string `json:"iss,omitempty"`

	// The principal that is the subject of the JWT.
	Subject string `json:"sub,omitempty"`

	// The recipients that the JWT is intended for.
	Audience []string `json:"aud,omitempty"`

	// The expiration time on or after which the JWT MUST NOT be accepted for processing.
	ExpirationTime int64 `json:"exp,omitempty"`

	// The time before which the JWT MUST NOT be accepted for processing.
	NotBefore int64 `json:"nbf,omitempty"`

	// The time at which the JWT was issued.
	IssuedAt int64 `json:"iat,omitempty"`

	// Unique identifier for the JWT.
	JTI string `json:"jti,omitempty"`
}
