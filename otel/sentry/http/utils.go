package sentryhttp

import (
	"fmt"
	"net/http"
)

// TransactionNameFormatter allows to adjust how a given transaction is reported
// when handling an HTTP request on the client or server side.
type TransactionNameFormatter func(r *http.Request) string

// Default transaction name formatter.
func txNameFormatter(r *http.Request) string {
	return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
}

func getStatus(code int) string {
	switch code {
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusBadRequest:
		return "canceled"
	default:
		return "error"
	}
}
