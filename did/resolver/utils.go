package resolver

import "net/http"

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
