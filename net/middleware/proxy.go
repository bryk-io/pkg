package middleware

import (
	"net/http"

	gmw "github.com/gorilla/handlers"
)

// ProxyHeaders inspects common reverse proxy headers and sets the corresponding
// fields in the HTTP request struct. These are X-Forwarded-For and X-Real-IP
// for the remote (client) IP address, X-Forwarded-Proto or X-Forwarded-Scheme
// for the scheme (http|https), X-Forwarded-Host for the host and the RFC7239
// Forwarded header, which may include both client IPs and schemes.
//
// NOTE: This middleware should only be used when behind a reverse
// proxy like nginx, HAProxy or Apache. Reverse proxies that don't (or are
// configured not to) strip these headers from client requests, or where these
// headers are accepted "as is" from a remote client (e.g. when Go is not behind
// a proxy), can manifest as a vulnerability if your application uses these
// headers for validating the 'trustworthiness' of a request.
func ProxyHeaders() func(http.Handler) http.Handler {
	return gmw.ProxyHeaders
}
