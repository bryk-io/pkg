package gzip

import (
	"net/http"

	gmw "github.com/gorilla/handlers"
)

// Handler enables gzip compression of HTTP responses for clients that
// support it via the 'Accept-Encoding' header. The compression level
// should be any integer value between `1` (optimal speed) and `9`
// (optimal compression) inclusive.
//
// Compressing TLS traffic may leak the page contents to an attacker if
// the page contains user input: http://security.stackexchange.com/a/102015/12208
func Handler(level int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return gmw.CompressHandlerLevel(next, level)
	}
}
