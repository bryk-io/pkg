package resolver

import (
	"encoding/json"
	"net/http"

	"go.bryk.io/pkg/did"
)

// internal JSON encoder instance.
var jsEnc Encoder

func init() {
	jsEnc = new(jsonEncoder)
}

// minimal default JSON encode.
type jsonEncoder struct{}

func (js *jsonEncoder) Encode(doc *did.Document) ([]byte, error) {
	return json.MarshalIndent(doc, "", "  ")
}

const (
	ldContext = "https://w3id.org/did-resolution/v1"

	deactivatedStatus int = http.StatusGone
)

// Map common error codes to an appropriate HTTP status.
// https://w3c-ccg.github.io/did-resolution/#bindings-https
func errToStatus(code string) int {
	switch code {
	case ErrInvalidDID:
		return http.StatusBadRequest
	case ErrInvalidURL:
		return http.StatusBadRequest
	case ErrNotFound:
		return http.StatusNotFound
	case ErrRepresentationNotSupported:
		return http.StatusNotAcceptable
	case ErrMethodNotSupported:
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}
