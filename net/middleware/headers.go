package middleware

import (
	"net/http"
)

// Headers middleware can be used to add a set of HTTP headers to all
// responses produced by an HTTP server.
func Headers(headers map[string]string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			for k, v := range headers {
				w.Header().Set(k, v)
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
