package resolver

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/did"
	"go.bryk.io/pkg/errors"
)

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

func (sp *sampleProvider) deactivate(id string) {
	rec, ok := sp.dir[id]
	if !ok {
		return
	}
	md := rec.GetMetadata()
	md.Deactivated = true
	md.Updated = time.Now().UTC().Format(time.RFC3339)
	_ = rec.AddMetadata(md)
	sp.mu.Lock()
	sp.dir[id] = rec
	sp.mu.Unlock()
}

func (sp *sampleProvider) activate(id string) {
	rec, ok := sp.dir[id]
	if !ok {
		return
	}
	md := rec.GetMetadata()
	md.Deactivated = false
	md.Updated = time.Now().UTC().Format(time.RFC3339)
	_ = rec.AddMetadata(md)
	sp.mu.Lock()
	sp.dir[id] = rec
	sp.mu.Unlock()
}

func TestResolve(t *testing.T) {
	assert := tdd.New(t)

	// Create sample provider an initial DID record
	prov := new(sampleProvider)
	prov.dir = make(map[string]*did.Identifier)
	activeID := prov.registerNew()

	rr, err := New(WithProvider("dev", prov))
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

	rr, err := New(WithProvider("dev", prov))
	assert.Nil(err, "failed to create new resolver instance")

	t.Run(ErrInvalidDID, func(t *testing.T) {
		_, err := rr.ResolveRepresentation("this-is-not-a-did", nil)
		assert.Equal(ErrInvalidDID, err.Error())
	})

	t.Run(ErrMethodNotSupported, func(t *testing.T) {
		_, err := rr.ResolveRepresentation("did:local:12345-67890", nil)
		assert.Equal(ErrMethodNotSupported, err.Error())
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

	rr, err := New(WithProvider("dev", prov))
	assert.Nil(err, "failed to create new resolver instance")

	// Resolver server
	mux := http.NewServeMux()
	mux.HandleFunc("/1.0/identifiers/", rr.ResolutionHandler)
	go func() {
		_ = http.ListenAndServe(":3000", mux)
	}()

	// must return a resolution result
	t.Run("no-accept-header", func(t *testing.T) {
		endpoint := "http://localhost:3000/1.0/identifiers/" + activeID
		req, _ := http.NewRequest(http.MethodGet, endpoint, nil)
		res, err := http.DefaultClient.Do(req)

		assert.Nil(err)
		assert.Equal(res.Header.Get("content-type"), ContentTypeWithProfile+";charset=utf-8")
		assert.Equal(res.StatusCode, http.StatusOK)

		body, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		val := new(Result)
		assert.Nil(json.Unmarshal(body, val))
	})

	// must return a DID document directly
	t.Run(ContentTypeLD, func(t *testing.T) {
		endpoint := "http://localhost:3000/1.0/identifiers/" + activeID
		req, _ := http.NewRequest(http.MethodGet, endpoint, nil)
		req.Header.Set("Accept", ContentTypeLD)
		res, err := http.DefaultClient.Do(req)

		assert.Nil(err)
		assert.Equal(res.Header.Get("content-type"), ContentTypeDocument+";charset=utf-8")
		assert.Equal(res.StatusCode, http.StatusOK)

		body, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		doc := new(did.Document)
		assert.Nil(json.Unmarshal(body, doc))
	})

	// must return a DID document directly
	t.Run(ContentTypeDocument, func(t *testing.T) {
		endpoint := "http://localhost:3000/1.0/identifiers/" + activeID
		req, _ := http.NewRequest(http.MethodGet, endpoint, nil)
		req.Header.Set("Accept", ContentTypeDocument)
		res, err := http.DefaultClient.Do(req)

		assert.Nil(err)
		assert.Equal(res.Header.Get("content-type"), ContentTypeDocument+";charset=utf-8")
		assert.Equal(res.StatusCode, http.StatusOK)

		body, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		doc := new(did.Document)
		assert.Nil(json.Unmarshal(body, doc))
	})

	// must return a resolution result
	t.Run(ContentTypeWithProfile, func(t *testing.T) {
		endpoint := "http://localhost:3000/1.0/identifiers/" + activeID
		req, _ := http.NewRequest(http.MethodGet, endpoint, nil)
		req.Header.Set("Accept", ContentTypeWithProfile)
		res, err := http.DefaultClient.Do(req)

		assert.Nil(err)
		assert.Equal(res.Header.Get("content-type"), ContentTypeWithProfile+";charset=utf-8")
		assert.Equal(res.StatusCode, http.StatusOK)

		body, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		val := new(Result)
		assert.Nil(json.Unmarshal(body, val))
	})

	// must return a "notFound" error
	t.Run(ErrNotFound, func(t *testing.T) {
		endpoint := "http://localhost:3000/1.0/identifiers/did:dev:not-found"
		req, _ := http.NewRequest(http.MethodGet, endpoint, nil)
		req.Header.Set("Accept", ContentTypeLD)
		res, err := http.DefaultClient.Do(req)

		assert.Nil(err)
		assert.Equal(res.Header.Get("content-type"), ContentTypeWithProfile+";charset=utf-8")
		assert.Equal(res.StatusCode, http.StatusNotFound)

		body, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		val := new(Result)
		assert.Nil(json.Unmarshal(body, val))
		assert.Equal(ErrNotFound, val.ResolutionMetadata.Error)
	})

	// must return a "internalError" error
	t.Run(ErrInternal, func(t *testing.T) {
		endpoint := "http://localhost:3000/1.0/identifiers/did:dev:with-internal-error"
		req, _ := http.NewRequest(http.MethodGet, endpoint, nil)
		req.Header.Set("Accept", ContentTypeLD)
		res, err := http.DefaultClient.Do(req)

		assert.Nil(err)
		assert.Equal(res.Header.Get("content-type"), ContentTypeWithProfile+";charset=utf-8")
		assert.Equal(res.StatusCode, http.StatusInternalServerError)

		body, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		val := new(Result)
		assert.Nil(json.Unmarshal(body, val))
		assert.Equal(ErrInternal, val.ResolutionMetadata.Error)
	})

	// must return a "internalError" error
	t.Run("deactivatedDID", func(t *testing.T) {
		prov.deactivate(activeID)
		defer prov.activate(activeID)

		endpoint := "http://localhost:3000/1.0/identifiers/" + activeID
		req, _ := http.NewRequest(http.MethodGet, endpoint, nil)
		req.Header.Set("Accept", ContentTypeLD)
		res, err := http.DefaultClient.Do(req)

		assert.Nil(err)
		assert.Equal(res.Header.Get("content-type"), ContentTypeWithProfile+";charset=utf-8")
		assert.Equal(res.StatusCode, deactivatedStatus)

		body, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		val := new(Result)
		assert.Nil(json.Unmarshal(body, val))
		assert.True(val.DocumentMetadata.Deactivated)
	})
}
