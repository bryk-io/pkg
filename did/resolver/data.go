package resolver

import (
	"net/http"
	"strings"

	"go.bryk.io/pkg/did"
)

// Common content-type IANA values.
const (
	// ContentTypeLD instructs the resolution endpoint to return
	// standard JSON LD data.
	ContentTypeLD = "application/ld+json"

	// ContentTypeDocument instructs the resolution endpoint to
	// return the obtained DID document as result.
	ContentTypeDocument = "application/did+ld+json"

	// ContentTypeWithProfile instructs the resolution endpoint to
	// return a complete resolution response structure as result. If
	// no value is provided in the `Accept` header, this will be the
	// default behavior.
	// https://w3c-ccg.github.io/did-resolution/#output-didresolutionresult
	ContentTypeWithProfile = "application/ld+json;profile=\"https://w3id.org/did-resolution\""
)

// Common error codes.
// https://w3c-ccg.github.io/did-resolution/#errors
const (
	// An unexpected error occurs during DID Resolution or DID URL
	// dereferencing.
	// https://w3c-ccg.github.io/did-resolution/#internalerror
	ErrInternal = "internalError"

	// During DID Resolution or DID URL dereferencing a DID or DID URL
	// doesn't exist.
	// https://w3c-ccg.github.io/did-resolution/#notfound
	ErrNotFound = "notFound"

	// An invalid DID is detected during DID Resolution.
	// https://w3c-ccg.github.io/did-resolution/#invaliddid
	ErrInvalidDID = "invalidDid"

	// An invalid DID URL is detected during DID Resolution or DID
	// URL dereferencing.
	// https://w3c-ccg.github.io/did-resolution/#invaliddidurl
	ErrInvalidURL = "invalidDidUrl"

	// Obtained DID document is invalid.
	ErrInvalidDocument = "invalidDidDocument"

	// DID method is not supported during DID Resolution or DID URL
	// dereferencing.
	// https://w3c-ccg.github.io/did-resolution/#methodnotsupported
	ErrMethodNotSupported = "methodNotSupported"

	// DID document representation is not supported during DID Resolution
	// or DID URL dereferencing.
	// https://w3c-ccg.github.io/did-resolution/#representationnotsupported
	ErrRepresentationNotSupported = "representationNotSupported"
)

// Result obtained from a "resolution" process.
// https://w3c-ccg.github.io/did-resolution/#output-didresolutionresult
type Result struct {
	// JSON-LD context statement for the document.
	// https://w3c-ccg.github.io/did-spec/#context
	Context []interface{} `json:"@context" yaml:"-"`

	// Resolved DID document.
	Document *did.Document `json:"didDocument,omitempty"`

	// DID document metadata.
	DocumentMetadata *did.DocumentMetadata `json:"didDocumentMetadata,omitempty"`

	// Resolution process metadata.
	ResolutionMetadata *ResolutionMetadata `json:"didResolutionMetadata,omitempty"`

	// Representation obtained from the DID document during a
	// `resolveRepresentation` operation.
	Representation []byte `json:"-"`
}

// ResolutionMetadata contains information about the DID Resolution process.
// This metadata typically changes between invocations of the DID Resolution
// functions as it represents data about the resolution process itself. The
// source of this metadata is the DID resolver.
type ResolutionMetadata struct {
	// Media type of the returned content.
	ContentType string `json:"contentType"`

	// Date and time of the DID resolution process.
	Retrieved string `json:"retrieved"`

	// Error code, if any.
	Error string `json:"error,omitempty"`
}

// ResolutionOptions provides additional settings available when processing
// a "resolve" request.
type ResolutionOptions struct {
	// The Media Type of the caller's preferred representation of the DID document.
	// The Media Type MUST be expressed as an ASCII string. The DID resolver
	// implementation SHOULD use this value to determine the representation
	// contained in the returned `didDocumentStream` if such a representation
	// is supported and available. This property is OPTIONAL for the
	// `resolveRepresentation` function and MUST NOT be used with the `resolve`
	// function.
	// If not provided, `application/did+ld+json` will be used by default.
	Accept string `json:"accept"`
}

// Validate the resolution options provided and load sensible default
// values.
func (ro *ResolutionOptions) Validate() error {
	if ro.Accept == "" || ro.Accept == "*/*" {
		ro.Accept = ContentTypeWithProfile
	}
	if ro.Accept == "application/json" {
		ro.Accept = ContentTypeLD
	}
	return nil
}

// FromRequest loads resolution options from an incoming HTTP request.
func (ro *ResolutionOptions) FromRequest(req *http.Request) {
	ro.Accept = strings.Split(req.Header.Get("Accept"), ",")[0]
}
