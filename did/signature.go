package did

import (
	"crypto/rand"
	"encoding/hex"
)

// SignatureLD provides a common format for Linked Data Signatures including information
// about the produced signature, parameters required to verify it, and the signature value
// itself.
//
// https://w3c-dvcg.github.io/ld-signatures/#linked-data-signature-overview
//
// NOTICE:
// This specification has been removed =/
// https://github.com/w3c-ccg/ld-signatures
type SignatureLD struct {
	// JSON-LD context statement for the document.
	// https://w3c-ccg.github.io/did-spec/#context
	Context []string `json:"@context,omitempty"`

	// Identifier for the digital signature suite that was used to create the signature.
	Type string `json:"type"`

	// A URI that identifies the public/private key pair associated with the signature. The
	// URI SHOULD be a URL that can be dereferenced to obtain a linked data document that
	// contains a link identifying the entity that owns the key pair. Dereferencing the entity
	// link SHOULD result in a Linked Data document that contains a link back to the URL
	// identifier for the public/private key pair, thereby proving ownership.
	Creator string `json:"creator"`

	// Creation timestamp in the RFC3339 format.
	Created string `json:"created"`

	// A string value that specifies the operational domain of a digital signature. This may be
	// an Internet domain name like "example.com", a ad-hoc value such as "corp-level3-access",
	// or a very specific transaction value like "8zF6T$mqP". A signer may include a domain in
	// its digital signature to restrict its use to particular target, identified by the specified
	// domain.
	Domain string `json:"domain,omitempty"`

	// A string value that is included in the digital signature and MUST only be used once for
	// a particular domain and window of time. This value is used to mitigate replay attacks.
	Nonce string `json:"nonce,omitempty"`

	// Signature value produced.
	Value []byte `json:"signatureValue"`
}

// NormalizedLD produces an RDF dataset on the JSON-LD document, the algorithm used is
// "URDNA2015" and the format "application/n-quads".
// https://json-ld.github.io/normalization/spec
func (s *SignatureLD) NormalizedLD() ([]byte, error) {
	r, err := normalize(s)
	return r, err
}

// GetInput returns a valid signature input value as described by the specification.
// https://w3c-dvcg.github.io/ld-signatures/#create-verify-hash-algorithm
func (s *SignatureLD) GetInput(data []byte) ([]byte, error) {
	// Add a random nonce value if not already set, as suggested by the specification
	if s.Nonce == "" {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		s.Nonce = hex.EncodeToString(b)
	}

	// Get normalized LD document
	options, err := s.NormalizedLD()
	if err != nil {
		return nil, err
	}

	// Generated input
	input := getHash(options)
	input = append(input, getHash(data)...)
	return input, nil
}
