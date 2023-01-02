package wkc

import (
	"go.bryk.io/pkg/did"
	"go.bryk.io/pkg/errors"
	"go.bryk.io/x/jose/jwk"
	"go.bryk.io/x/jose/jwt"
)

// Configuration provides a "Well-Known" resource that includes Domain
// Linkage Assertions. Usually presented in the format of a JSON object.
// The DID Configuration resource MUST exist at the domain root, in the
// IETF 8615 Well-Known Resource directory, as follows:
//
//	/.well-known/did-configuration
type Configuration struct {
	// The resource MUST contain this property, and its value MUST be an array
	// of Domain Linkage Assertion objects.
	Entries []*DomainLink `json:"entries"`
}

// DomainLink entries contain a DID string and cryptographic proof (in the
// form of a JWT signed with the specified DID's keys) that verifies the
// domain controller and the DID controller are the same entity.
type DomainLink struct {
	// MUST be present and MUST match the DID string specified in the `iss`
	// property of the assertion's decoded JWT value.
	DID string `json:"did"`

	// MUST be present, and MUST be a JWT signed by the keys currently
	// associated with the specified DID. The JWT assertion defines the
	// following claims:
	//  - iss: MUST be a DID string that matches the DID string asserted
	//    in the DID Entry Object.
	//  - exp: SHOULD be defined to indicate a time after which the assertion
	//    of domain linkage MUST NOT be deemed valid.
	//  - domain: MUST be present, and MUST match the domain the DID
	//    Configuration resource is located at.
	JWT string `json:"jwt"`
}

// RegisterKey creates a new authentication method that ca be used to
// generate JWT tokens (using the PS256 algorithm) on the provided DID.
// The new key will be registered using the provided `vm` value. If a key
// already exists under that name an error will be returned.
func RegisterKey(id *did.Identifier, vm string) error {
	if k := id.VerificationMethod(vm); k != nil {
		return errors.New("verification name already exists")
	}
	if err := id.AddNewVerificationMethod(vm, did.KeyTypeRSA); err != nil {
		return err
	}
	return id.AddVerificationRelationship(id.GetReference(vm), did.AuthenticationVM)
}

// GenerateDomainLink issues a new JWT using the verification method `vm`
// associated with the provided DID instance.
//   - DID is used as the `issuer`
//   - DID is also used as the `sub` claim
//   - Default expiration is 720hrs
//   - The custom claim `domain` is included in the token
//   - `domain` is included in the `aud` claim
func GenerateDomainLink(id *did.Identifier, vm, domain string) (*DomainLink, error) {
	// Get "token generator key" from DID
	tgk, err := toJWK(id, vm)
	if err != nil {
		return nil, errors.New("invalid verification method")
	}

	// Create a JWT generator instance
	// - DID is used as the `issuer`
	// - `jwt` key on the DID doc is used to sign the tokens
	gen, err := jwt.NewGenerator(id.String())
	if err != nil {
		return nil, err
	}
	if err = gen.AddKey(tgk); err != nil {
		return nil, err
	}

	// Issue the domain link token
	token, err := gen.Issue(tgk.ID(), &jwt.TokenParameters{
		Subject:      id.String(),
		Audience:     []string{domain},
		CustomClaims: map[string]string{"domain": domain},
	})
	if err != nil {
		return nil, err
	}

	// Return entry
	return &DomainLink{
		DID: id.String(),
		JWT: token.String(),
	}, nil
}

// ValidateDomainLink ensures the provided domain link entry was properly
// issued and remains valid.
func ValidateDomainLink(id *did.Identifier, dl *DomainLink, domain string) error {
	// Validate token and DID values
	if id.String() != dl.DID {
		return errors.New("invalid DID value")
	}
	token, err := jwt.Parse(dl.JWT)
	if err != nil {
		return err
	}

	// Get "token generator key" from DID
	tgk, err := toJWK(id, token.Header().KeyID)
	if err != nil {
		return errors.New("invalid verification method")
	}

	// Prepare token validator instance
	ks := jwk.Set{Keys: []jwk.Record{
		tgk.Export(true),
	}}
	val, err := jwt.NewValidator(jwt.WithValidationKeys(ks))
	if err != nil {
		return err
	}

	// Run required token validations
	checks := []jwt.Check{
		jwt.IssuerCheck(id.String()),        // iss
		jwt.SubjectCheck(id.String()),       // sub
		jwt.AudienceCheck([]string{domain}), // aud
		jwtDomainCheck(domain),              // domain
	}
	return val.Validate(dl.JWT, checks...)
}
