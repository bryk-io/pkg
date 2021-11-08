package did

import (
	"encoding/json"
	"errors"
)

// Extension provides a flexible mechanism to add contextual parameters to
// the standard elements used on a DID document.
type Extension struct {
	// Unique identifier for the extension.
	ID string `json:"id" yaml:"id"`

	// Semantic versioning value.
	Version string `json:"version" yaml:"version"`

	// Custom extension parameters.
	Data interface{} `json:"data" yaml:"data,omitempty"`
}

// Populate the provided holder element with the extension's data.
func (ext Extension) load(holder interface{}) error {
	js, err := json.Marshal(ext.Data)
	if err != nil {
		return err
	}
	return json.Unmarshal(js, holder)
}

// Document represents a valid DID document instance.
// https://w3c.github.io/did-core/#core-properties
type Document struct {
	// JSON-LD context statement for the document.
	// https://w3c-ccg.github.io/did-spec/#context
	Context []interface{} `json:"@context" yaml:"-"`

	// DID described by the document.
	// https://w3c.github.io/did-core/#did-subject
	Subject string `json:"id" yaml:"id"`

	// A DID controller is an entity that is authorized to make changes to a DID document.
	// https://w3c.github.io/did-core/#did-controller
	Controller string `json:"controller,omitempty" yaml:"controller,omitempty"`

	// A DID subject can have multiple identifiers for different purposes, or at
	// different times. The assertion that two or more DIDs (or other types of URI)
	// refer to the same DID subject can be made using the `alsoKnownAs` property.
	// https://w3c.github.io/did-core/#also-known-as
	AlsoKnownAs []string `json:"alsoKnownAs,omitempty" yaml:"alsoKnownAs,omitempty"`

	// A DID document can express verification methods, such as cryptographic public
	// keys, which can be used to authenticate or authorize interactions with the DID
	// subject or associated parties.
	//
	// https://w3c.github.io/did-core/#verification-methods
	VerificationMethod []PublicKey `json:"verificationMethod,omitempty" yaml:"verificationMethod,omitempty"`

	// The authentication verification relationship is used to specify how the DID subject
	// is expected to be authenticated, for purposes such as logging into a website or
	// engaging in any sort of challenge-response protocol.
	//
	// https://w3c.github.io/did-core/#authentication
	Authentication []string `json:"authentication,omitempty" yaml:"authentication,omitempty"`

	// The assertionMethod verification relationship is used to specify how the DID subject
	// is expected to express claims, such as for the purposes of issuing a Verifiable
	// Credential.
	//
	// This property is useful, for example, during the processing of a verifiable credential
	// by a verifier. During verification, a verifier checks to see if a verifiable credential
	// contains a proof created by the DID subject by checking that the verification method used
	// to assert the proof is associated with the assertionMethod property in the corresponding
	// DID document.
	//
	// https://w3c.github.io/did-core/#assertion
	AssertionMethod []string `json:"assertionMethod,omitempty" yaml:"assertionMethod,omitempty"`

	// The keyAgreement verification relationship is used to specify how an entity can
	// generate encryption material in order to transmit confidential information intended
	// for the DID subject, such as for the purposes of establishing a secure communication
	// channel with the recipient.
	//
	// An example of when this property is useful is when encrypting a message intended for
	// the DID subject. In this case, the counterparty uses the cryptographic public key
	// information in the verification method to wrap a decryption key for the recipient.
	//
	// https://w3c.github.io/did-core/#key-agreement
	KeyAgreement []string `json:"keyAgreement,omitempty" yaml:"keyAgreement,omitempty"`

	// The capabilityInvocation verification relationship is used to specify a verification
	// method that might be used by the DID subject to invoke a cryptographic capability,
	// such as the authorization to update the DID Document.
	//
	// An example of when this property is useful is when a DID subject needs to access a
	// protected HTTP API that requires authorization in order to use it. In order to authorize
	// when using the HTTP API, the DID subject uses a capability that is associated with a
	// particular URL that is exposed via the HTTP API. The invocation of the capability could
	// be expressed in a number of ways, e.g., as a digitally signed message that is placed
	// into the HTTP Headers.
	//
	// The server providing the HTTP API is the verifier of the capability and it would need
	// to verify that the verification method referred to by the invoked capability exists in
	// the capabilityInvocation property of the DID document. The verifier would also check
	// to make sure that the action being performed is valid and the capability is appropriate
	// for the resource being accessed. If the verification is successful, the server has
	// cryptographically determined that the invoker is authorized to access the protected
	// resource.
	//
	// https://w3c.github.io/did-core/#capability-invocation
	CapabilityInvocation []string `json:"capabilityInvocation,omitempty" yaml:"capabilityInvocation,omitempty"`

	// The capabilityDelegation verification relationship is used to specify a mechanism that
	// might be used by the DID subject to delegate a cryptographic capability to another party,
	// such as delegating the authority to access a specific HTTP API to a subordinate.
	//
	// An example of when this property is useful is when a DID controller chooses to delegate
	// their capability to access a protected HTTP API to a party other than themselves. In
	// order to delegate the capability, the DID subject would use a verification method
	// associated with the capabilityDelegation verification relationship to cryptographically
	// sign the capability over to another DID subject.
	//
	// https://w3c.github.io/did-core/#capability-delegation
	CapabilityDelegation []string `json:"capabilityDelegation,omitempty" yaml:"capabilityDelegation,omitempty"`

	// Services are used in DID documents to express ways of communicating with the DID subject
	// or associated entities. A service can be any type of service the DID subject wants to
	// advertise, including decentralized identity management services for further discovery,
	// authentication, authorization, or interaction.
	//
	// https://w3c.github.io/did-core/#services
	Services []ServiceEndpoint `json:"service,omitempty" yaml:"service,omitempty"`
}

// DocumentMetadata provides information pertaining to the DID document itself,
// rather than the DID subject.
// https://www.w3.org/TR/did-core/#metadata-structure
//
// Additional details:
//   https://github.com/w3c/did-core/issues/65
//   https://github.com/w3c/did-core/issues/203
type DocumentMetadata struct {
	// Timestamp of the original creation, normalized to UTC 00:00.
	// https://w3c-ccg.github.io/did-spec/#created-optional
	Created string `json:"created,omitempty" yaml:"created,omitempty"`

	// Timestamp of the latest updated registered for the document, normalized to
	// UTC 00:00.
	// https://w3c-ccg.github.io/did-spec/#updated-optional
	Updated string `json:"updated,omitempty" yaml:"updated,omitempty"`

	// Whether the DID should be considered active or not.
	// https://www.w3.org/TR/did-spec-registries/#deactivated
	Deactivated bool `json:"deactivated" yaml:"deactivated"`
}

// ServiceEndpoint represents any type of service the entity wishes to advertise,
// including decentralized identity management services for further discovery,
// authentication, authorization, or interaction.
// https://w3c.github.io/did-core/#service-endpoints
type ServiceEndpoint struct {
	// Unique identifier for the service entry.
	ID string `json:"id" yaml:"id"`

	// Cryptographic suite identifier.
	Type string `json:"type" yaml:"type"`

	// Main URL for interactions.
	Endpoint string `json:"serviceEndpoint" yaml:"serviceEndpoint"`

	// Extensions used on the service endpoint instance.
	Extensions []Extension `json:"extensions,omitempty" yaml:"extensions,omitempty"`
}

// AddExtension can be used to register additional contextual information in the service
// instance. If another extension with the same id and version information, the data will
// be updated.
func (se *ServiceEndpoint) AddExtension(ext Extension) {
	for i, ee := range se.Extensions {
		if ee.ID == ext.ID && ee.Version == ext.Version {
			se.Extensions[i] = ext
			return
		}
	}
	se.Extensions = append(se.Extensions, ext)
}

// GetExtension retrieves the information available for a given extension and decode it into
// the  provided holder instance (usually a pointer to a structure type). If no information is
// available or a decoding problems occurs an error will be returned.
func (se *ServiceEndpoint) GetExtension(id string, version string, holder interface{}) error {
	for _, ee := range se.Extensions {
		if ee.ID == id && ee.Version == version {
			return ee.load(holder)
		}
	}
	return errors.New("no extension")
}

// RegisterContext adds a new context entry to the document. Useful when
// adding new data entries.
// https://w3c.github.io/json-ld-syntax/#the-context
func (d *Document) RegisterContext(el interface{}) {
	for _, v := range d.Context {
		if el == v {
			return
		}
	}
	d.Context = append(d.Context, el)
}

// ExpandedLD returns an expanded JSON-LD document.
// http://www.w3.org/TR/json-ld-api/#expansion-algorithm
func (d *Document) ExpandedLD() ([]byte, error) {
	return expand(d)
}

// NormalizedLD produces an RDF dataset on the JSON-LD document,
// the algorithm used is "URDNA2015" and the format "application/n-quads".
// https://json-ld.github.io/normalization/spec
func (d *Document) NormalizedLD() ([]byte, error) {
	return normalize(d)
}
