package middleware

import (
	"net/http"

	gmw "github.com/gorilla/handlers"
)

// GzipCompression enabled for HTTP responses for clients that support it
// via the 'Accept-Encoding' header.
//
// Compressing TLS traffic may leak the page contents to an attacker if the
// page contains user input: http://security.stackexchange.com/a/102015/12208
func GzipCompression(level int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return gmw.CompressHandlerLevel(next, level)
	}
}