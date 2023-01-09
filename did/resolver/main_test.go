package resolver

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"

	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/did"
	"go.bryk.io/pkg/errors"
)

type sampleEncoder struct{}

func (js *sampleEncoder) Encode(doc *did.Document) ([]byte, error) {
	return json.MarshalIndent(doc, "", "  ")
}

type sampleProvider struct {
	mu  sync.Mutex
	dir map[string]*did.Identifier
}

func (sp *sampleProvider) Read(did string) (*did.Document, *did.DocumentMetadata, error) {
	// simulate an error returned by the provider
	if did == "did:dev:with-internal-error" {
		return nil, nil, errors.New(ErrInternal)
	}

	// simulate "not found" results
	if did == "did:dev:not-found" {
		return nil, nil, errors.New(ErrNotFound)
	}

	sp.mu.Lock()
	defer sp.mu.Unlock()
	id, ok := sp.dir[did]
	if !ok {
		return nil, nil, errors.New(ErrNotFound)
	}
	return id.Document(true), id.GetMetadata(), nil
}

func (sp *sampleProvider) registerNew() string {
	id, _ := did.NewIdentifierWithMode("dev", "", did.ModeUUID)
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.dir[id.String()] = id
	return id.String()
}

func TestResolve(t *testing.T) {
	assert := tdd.New(t)

	// Create sample provider an initial DID record
	prov := new(sampleProvider)
	prov.dir = make(map[string]*did.Identifier)
	activeID := prov.registerNew()

	rr, err := New(
		WithProvider("dev", prov),                            // register "dev" provider
		WithEncoder(ContentTypeDocument, new(sampleEncoder)), // register "json" encoder
	)
	assert.Nil(err, "failed to create new resolver instance")

	t.Run(ErrInvalidDID, func(t *testing.T) {
		_, err := rr.Resolve("this-is-not-a-did", nil)
		assert.Equal(ErrInvalidDID, err.Error())
	})

	t.Run(ErrMethodNotSupported, func(t *testing.T) {
		_, err := rr.Resolve("did:local:12345-67890", nil)
		assert.Equal(ErrMethodNotSupported, err.Error())
	})

	t.Run(ErrInternal, func(t *testing.T) {
		_, err := rr.Resolve("did:dev:with-internal-error", nil)
		assert.Equal(ErrInternal, err.Error())
	})

	t.Run(ErrNotFound, func(t *testing.T) {
		_, err := rr.Resolve("did:dev:not-found", nil)
		assert.Equal(ErrNotFound, err.Error())
	})

	// Valid resolve attempt
	res, err := rr.Resolve(activeID, nil)
	assert.Nil(err)
	js, _ := json.MarshalIndent(res, "", "  ")
	t.Logf("%s\n", js)
}

func TestResolveRepresentation(t *testing.T) {
	assert := tdd.New(t)

	// Create sample provider an initial DID record
	prov := new(sampleProvider)
	prov.dir = make(map[string]*did.Identifier)
	activeID := prov.registerNew()

	rr, err := New(
		WithProvider("dev", prov),                            // register "dev" provider
		WithEncoder(ContentTypeDocument, new(sampleEncoder)), // register "json" encoder
	)
	assert.Nil(err, "failed to create new resolver instance")

	t.Run(ErrInvalidDID, func(t *testing.T) {
		_, err := rr.ResolveRepresentation("this-is-not-a-did", nil)
		assert.Equal(ErrInvalidDID, err.Error())
	})

	t.Run(ErrMethodNotSupported, func(t *testing.T) {
		_, err := rr.ResolveRepresentation("did:local:12345-67890", nil)
		assert.Equal(ErrMethodNotSupported, err.Error())
	})

	t.Run(ErrRepresentationNotSupported, func(t *testing.T) {
		_, err := rr.ResolveRepresentation("did:dev:12345-67890", &ResolutionOptions{Accept: "x-messagepack"})
		assert.Equal(ErrRepresentationNotSupported, err.Error())
	})

	t.Run(ErrInternal, func(t *testing.T) {
		_, err := rr.ResolveRepresentation("did:dev:with-internal-error", nil)
		assert.Equal(ErrInternal, err.Error())
	})

	t.Run(ErrNotFound, func(t *testing.T) {
		_, err := rr.ResolveRepresentation("did:dev:not-found", nil)
		assert.Equal(ErrNotFound, err.Error())
	})

	// Valid attempt
	res, err := rr.ResolveRepresentation(activeID, nil)
	assert.Nil(err)
	t.Logf("%s", res.Representation)
}

func TestResolutionHandler(t *testing.T) {
	assert := tdd.New(t)

	// Create sample provider an initial DID record
	prov := new(sampleProvider)
	prov.dir = make(map[string]*did.Identifier)
	activeID := prov.registerNew()

	rr, err := New(
		WithProvider("dev", prov),                            // register "dev" provider
		WithEncoder(ContentTypeDocument, new(sampleEncoder)), // register "json" encoder
	)
	assert.Nil(err, "failed to create new resolver instance")

	// Resolver server
	mux := http.NewServeMux()
	mux.HandleFunc("/identifiers/", rr.ResolutionHandler)
	go func() {
		_ = http.ListenAndServe(":3000", mux)
	}()

	endpoint := "http://localhost:3000/identifiers/" + activeID
	req, _ := http.NewRequest(http.MethodGet, endpoint, nil)
	req.Header.Set("Accept", ContentTypeDocument)
	res, err := http.DefaultClient.Do(req)
	assert.Nil(err)
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	t.Logf("%s", body)
}
