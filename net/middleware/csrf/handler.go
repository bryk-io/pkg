//go:build go1.25

package csrf

import "net/http"

// Reference:
// https://words.filippo.io/csrf/

// Handler returns a middleware that applies cross-origin checks on incoming
// requests. If a request fails cross-origin checks, the request is rejected
// with a `403 Forbidden` status.
func Handler(options *Options) func(http.Handler) http.Handler {
	ch := new(http.CrossOriginProtection)
	if options != nil {
		for _, origin := range options.TrustedOrigins {
			_ = ch.AddTrustedOrigin(origin)
		}
		if options.BypassPattern != "" {
			ch.AddInsecureBypassPattern(options.BypassPattern)
		}
	}

	return func(next http.Handler) http.Handler {
		return ch.Handler(next)
	}
}
