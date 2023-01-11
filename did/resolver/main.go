package resolver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"go.bryk.io/pkg/did"
	"go.bryk.io/pkg/errors"
)

// Provider instances are method-specific and abstract away the
// details on how to interact with the verifiable data registry
// being used.
type Provider interface {
	// Read the details available on the verifiable data registry
	// for a specific `did` entry. DID document metadata is optional
	// but recommended.
	//  - If an error is returned is must be a valid error code as defined
	//    in the spec.
	//  - If no error AND no DID document are returned, a "notFound" error
	//    will be returned by the resolver.
	// https://w3c-ccg.github.io/did-resolution/#errors
	Read(did string) (*did.Document, *did.DocumentMetadata, error)
}

// Encoder instances can be used to generate different
// representations for a resolved DID document.
type Encoder interface {
	// Encode an existing DID document to a valid representation.
	// If an error is returned is must be a valid error code as
	// defined in the spec.
	Encode(doc *did.Document) ([]byte, error)
}

// Instance elements are the main utility provided by the `resolver`
// package. A resolver instance can be used to provided the low level
// resolve functions as well as exposing it through a compliant HTTP
// endpoint intended for public consumption.
// https://w3c-ccg.github.io/did-resolution/#resolving-algorithm
type Instance struct {
	// DID methods registered.
	providers map[string]Provider

	// Encoders available to obtain DID representations.
	encoders map[string]Encoder
}

// New returns a ready-to-use DID resolver instance.
func New(opts ...Option) (*Instance, error) {
	i := &Instance{
		providers: make(map[string]Provider),
		encoders: map[string]Encoder{
			ContentTypeLD:          jsEnc,
			ContentTypeDocument:    jsEnc,
			ContentTypeWithProfile: jsEnc,
		},
	}
	for _, opt := range opts {
		if err := opt(i); err != nil {
			return nil, err
		}
	}
	return i, nil
}

// Resolve a DID into a DID document by using the "Read" operation of the
// applicable DID method.
// https://www.w3.org/TR/did-core/#did-resolution
func (ri *Instance) Resolve(id string, opts *ResolutionOptions) (*Result, error) {
	// Use default resolution options
	if opts == nil {
		opts = new(ResolutionOptions)
	}
	_ = opts.Validate()

	// prepare result holder
	res := &Result{
		Context: []interface{}{ldContext},
		ResolutionMetadata: &ResolutionMetadata{
			ContentType: opts.Accept,
			Retrieved:   time.Now().UTC().Format(time.RFC3339),
		},
	}

	// is DID valid?
	ID, err := did.Parse(id)
	if err != nil {
		err = errors.New(ErrInvalidDID)
		res.ResolutionMetadata.Error = err.Error()
		return res, err
	}

	// is method supported?
	provider, ok := ri.providers[ID.Method()]
	if !ok {
		err = errors.New(ErrMethodNotSupported)
		res.ResolutionMetadata.Error = err.Error()
		return res, err
	}

	// retrieve DID doc and optional metadata
	res.Document, res.DocumentMetadata, err = provider.Read(id)
	if err != nil {
		res.ResolutionMetadata.Error = err.Error()
		return res, err
	}

	// return not found error if DID doc wasn't retrieved
	if res.Document == nil {
		err = errors.New(ErrNotFound)
		res.ResolutionMetadata.Error = err.Error()
		return res, err
	}

	// resolution was successful
	return res, nil
}

// ResolveRepresentation attempts to resolve a DID into a DID document by using
// the "Read" operation of the applicable DID method and encode a suitable
// representation based on the options provided.
// https://www.w3.org/TR/did-core/#did-resolution
func (ri *Instance) ResolveRepresentation(id string, opts *ResolutionOptions) (*Result, error) {
	// Use default resolution options
	if opts == nil {
		opts = new(ResolutionOptions)
	}
	_ = opts.Validate()

	// prepare result holder
	res := &Result{
		Context: []interface{}{ldContext},
		ResolutionMetadata: &ResolutionMetadata{
			ContentType: opts.Accept,
			Retrieved:   time.Now().UTC().Format(time.RFC3339),
		},
	}

	// is DID valid?
	ID, err := did.Parse(id)
	if err != nil {
		err = errors.New(ErrInvalidDID)
		res.ResolutionMetadata.Error = err.Error()
		return res, err
	}

	// is method supported?
	provider, ok := ri.providers[ID.Method()]
	if !ok {
		err = errors.New(ErrMethodNotSupported)
		res.ResolutionMetadata.Error = err.Error()
		return res, err
	}

	// is encoder supported?
	enc, ok := ri.encoders[opts.Accept]
	if !ok {
		err = errors.New(ErrRepresentationNotSupported)
		res.ResolutionMetadata.Error = err.Error()
		return res, err
	}

	// retrieve DID doc and optional metadata
	res.Document, res.DocumentMetadata, err = provider.Read(id)
	if err != nil {
		res.ResolutionMetadata.Error = err.Error()
		return res, err
	}

	// return not found error if DID doc wasn't retrieved
	if res.Document == nil {
		err = errors.New(ErrNotFound)
		res.ResolutionMetadata.Error = err.Error()
		return res, err
	}

	// get DID document representation
	res.Representation, err = enc.Encode(res.Document)
	if err != nil {
		res.ResolutionMetadata.Error = err.Error()
		return res, err
	}

	// resolution was successful
	return res, nil
}

// ResolutionHandler exposes the `resolve` operations through an HTTP endpoint
// compatible with the DIF specification.
// https://w3c-ccg.github.io/did-resolution/#bindings-https
func (ri *Instance) ResolutionHandler(rw http.ResponseWriter, rq *http.Request) {
	// get requested identifier
	id := strings.TrimPrefix(rq.URL.Path, "/1.0/identifiers/")

	// process resolution request
	opts := new(ResolutionOptions)
	opts.FromRequest(rq)
	_ = opts.Validate()

	var (
		res *Result
		err error
	)
	if strings.Count(opts.Accept, "json") > 0 {
		// standard JSON LD mime types are handled directly by a
		// 'resolve' operation
		res, err = ri.Resolve(id, opts)
	} else {
		// custom mime types are hadled by specialized encoders
		res, err = ri.ResolveRepresentation(id, opts)
	}

	// return error
	if err != nil {
		rw.Header().Set("Content-Type", ContentTypeWithProfile+";charset=utf-8")
		rw.WriteHeader(errToStatus(err.Error()))
		_ = json.NewEncoder(rw).Encode(res)
		return
	}

	// return deactivated doc
	if res.DocumentMetadata != nil && res.DocumentMetadata.Deactivated {
		rw.Header().Set("Content-Type", ContentTypeWithProfile+";charset=utf-8")
		rw.WriteHeader(deactivatedStatus)
		_ = json.NewEncoder(rw).Encode(res)
		return
	}

	// return result
	// https://w3c-ccg.github.io/did-resolution/#did-resolution-result
	switch opts.Accept {
	case ContentTypeLD:
		// return the DID document directly
		rw.Header().Set("Content-Type", ContentTypeDocument+";charset=utf-8")
		_ = json.NewEncoder(rw).Encode(res.Document)
	case ContentTypeDocument:
		// return the DID document directly
		rw.Header().Set("Content-Type", ContentTypeDocument+";charset=utf-8")
		_ = json.NewEncoder(rw).Encode(res.Document)
	case ContentTypeWithProfile:
		// return the complete resolution result
		rw.Header().Set("Content-Type", ContentTypeWithProfile+";charset=utf-8")
		_ = json.NewEncoder(rw).Encode(res)
	default:
		// return custom representation
		rw.Header().Set("Content-Type", opts.Accept)
		_, _ = rw.Write(res.Representation)
	}
}
