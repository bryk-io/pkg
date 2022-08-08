package sentryhttp

import (
	"net"
	"net/http"

	apiErrors "go.bryk.io/pkg/otel/errors"
	"go.bryk.io/pkg/otel/sentry"
	"go.bryk.io/pkg/otel/sentry/internal"
)

// Server provides a middleware to instrument HTTP handlers to automatically
// capture performance data on all HTTP requests served.
func Server(rep apiErrors.Reporter, nf TransactionNameFormatter) func(http.Handler) http.Handler {
	// regular processing
	if _, ok := rep.(*internal.Reporter); !ok {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}
	}

	// transaction formatter function
	if nf == nil {
		nf = txNameFormatter
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// create operation for HTTP request
			opts := []apiErrors.OperationOption{
				sentry.AsTransaction(nf(r)),
				sentry.ToContinue(r.Header.Get("sentry-trace")),
			}
			op := rep.Start(r.Context(), "http.server", opts...)
			defer op.Finish()
			setOperationDetails(op, r)

			// request processing
			next.ServeHTTP(w, r.Clone(op.Context()))
		}
		return http.HandlerFunc(fn)
	}
}

func setOperationDetails(op apiErrors.Operation, r *http.Request) {
	io, ok := op.(*internal.Operation)
	if !ok {
		return
	}

	// Load HTTP request
	io.Scope.SetRequest(r)

	// Get user IP
	addr := ""
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if host != "" && err == nil {
		addr = host
	}
	if ip := r.Header.Get("X-Real-Ip"); ip != "" && ip != "127.0.0.1" {
		io.User(apiErrors.User{IPAddress: ip})
		return
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" && ip != "127.0.0.1" {
		io.User(apiErrors.User{IPAddress: ip})
		return
	}
	if addr != "" && addr != "127.0.0.1" {
		io.User(apiErrors.User{IPAddress: addr})
	}
}
