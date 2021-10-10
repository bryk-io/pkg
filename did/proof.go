package did

import (
	"crypto/rand"
	"encoding/hex"
)

// ProofLD provides a common format for Linked Data Proofs. Proofs add
// authentication and integrity protection to linked data documents through
// the use of mathematical algorithms.
// https://w3c-ccg.github.io/ld-proofs/
type ProofLD struct {
	// JSON-LD context statement for the document.
	// https://w3c-ccg.github.io/did-spec/#context
	Context []string `json:"@context,omitempty" yaml:"-"`

	// A URI that identifies the digital proof suite that was used to create
	// the proof.
	Type string `json:"type" yaml:"type"`

	// A link to a machine-readable object, such as a DID Document, that contains
	// authorization relations that explicitly permit the use of certain verification
	// methods for specific purposes. For example, a controller object could contain
	// statements that restrict a public key to being used only for signing Verifiable
	// Credentials and no other kinds of documents.
	Controller string `json:"controller,omitempty" yaml:"controller,omitempty"`

	// Creation timestamp in the RFC3339 format.
	Created string `json:"created" yaml:"created"`

	// A string value that specifies the operational domain of a digital proof.
	// This may be an Internet domain name like "example.com", a ad-hoc value such
	// as "corp-level3-access", or a very specific transaction value like "8zF6T$mqP".
	// A signer may include a domain in its digital proof to restrict its use to
	// particular target, identified by the specified domain.
	Domain string `json:"domain,omitempty" yaml:"domain,omitempty"`

	// A random or pseudo-random value used by some authentication protocols to
	// mitigate replay attacks.
	Challenge string `json:"challenge,omitempty" yaml:"challenge,omitempty"`

	// A string value that is included in the digital proof and MUST only be used
	// once for a particular domain and window of time. This value is used to mitigate
	// replay attacks.
	Nonce string `json:"nonce,omitempty" yaml:"nonce,omitempty"`

	// The specific intent for the proof, the reason why an entity created it.
	// Acts as a safeguard to prevent the proof from being misused for a purpose
	// other than the one it was intended for. For example, a proof can be used
	// for purposes of authentication, for asserting control of a Verifiable
	// Credential (assertionMethod), and several others.
	//
	// Common values include: authentication, assertionMethod, keyAgreement,
	// capabilityInvocation, capabilityDelegation.
	// https://w3c-ccg.github.io/ld-proofs/#proof-purpose
	Purpose string `json:"proofPurpose,omitempty" yaml:"proofPurpose,omitempty"`

	// A set of parameters required to independently verify the proof, such as
	// an identifier for a public/private key pair that would be used in the
	// proof.
	VerificationMethod string `json:"verificationMethod,omitempty" yaml:"verificationMethod,omitempty"`

	// Proof value produced.
	Value []byte `json:"proofValue" yaml:"proofValue"`
}

// NormalizedLD produces an RDF dataset on the JSON-LD document, the algorithm used is
// "URDNA2015" and the format "application/n-quads"
// https://json-ld.github.io/normalization/spec
func (p *ProofLD) NormalizedLD() ([]byte, error) {
	r, err := normalize(p)
	return r, err
}

// GetInput returns a valid proof input value as described by the specification.
// https://w3c-ccg.github.io/ld-proofs/#proof-algorithm
func (p *ProofLD) GetInput(data []byte) ([]byte, error) {
	// Add a random nonce value if not already set, as suggested by the specification
	if p.Nonce == "" {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		p.Nonce = hex.EncodeToString(b)
	}

	// Remove any proof value already set
	if p.Value != nil {
		p.Value = nil
	}

	// Get normalized proof document
	doc, err := p.NormalizedLD()
	if err != nil {
		return nil, err
	}

	// Generated input
	// input = hash(normalized_document) | hash(data)
	return append(getHash(doc), getHash(data)...), nil
}
