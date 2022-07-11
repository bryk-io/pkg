package sentryhttp

import (
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
			op.(*internal.Operation).Scope.SetRequest(r) // nolint: forcetypeassert

			// request processing
			next.ServeHTTP(w, r.Clone(op.Context()))
		}
		return http.HandlerFunc(fn)
	}
}
