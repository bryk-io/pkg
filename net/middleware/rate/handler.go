package rate

import (
	"net/http"

	lib "golang.org/x/time/rate"
)

// Handler provides a rate limiter middleware based on a "token bucket"
// implementation. A rate limiter controls how frequently HTTP requests
// are allowed to happen. The "token bucket" is of size `limit`, initially
// full and refilled at rate `burst` tokens per second.
//
// The rate limiter is applied to all incoming requests and will reject, with
// status 429, those that exceed the configured limit.
//
// More information: https://www.rfc-editor.org/rfc/rfc6585.html#section-4
func Handler(limit, burst uint) func(http.Handler) http.Handler {
	rl := lib.NewLimiter(lib.Limit(limit), int(burst))
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if err := rl.Wait(r.Context()); err != nil {
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}
			// Call the next handler in the chain.
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
