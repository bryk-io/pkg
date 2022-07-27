package sentryhttp

import (
	"context"
	"io"
	"net/http"

	"go.bryk.io/pkg/errors"
	apiErrors "go.bryk.io/pkg/otel/errors"
	"go.bryk.io/pkg/otel/sentry/internal"
	"go.opentelemetry.io/otel/propagation"
)

// Client instruments the provided `base` transport to automatically capture
// performance data on all HTTP requests.
func Client(base http.RoundTripper, rep apiErrors.Reporter, nf TransactionNameFormatter) http.RoundTripper {
	sr, ok := rep.(*internal.Reporter)
	if !ok {
		return base
	}
	if base == nil {
		base = http.DefaultTransport
	}
	if nf == nil {
		nf = txNameFormatter
	}
	return &customTransport{
		nf:   nf,
		rep:  sr,
		next: base,
	}
}

type customTransport struct {
	rep  *internal.Reporter
	next http.RoundTripper
	nf   TransactionNameFormatter
}

func (ct *customTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	op := ct.rep.FromContext(r.Context())
	if op == nil {
		// Continue normal processing
		return ct.next.RoundTrip(r)
	}

	// Create operation for the HTTP request
	opts := []apiErrors.OperationOption{
		internal.AsTransaction(ct.nf(r)),
		internal.ToContinue(op.TraceID()),
	}
	ro := ct.rep.Start(context.Background(), "http.client", opts...)
	defer ro.Finish()
	ro.(*internal.Operation).Scope.SetRequest(r) // nolint: forcetypeassert

	// Propagate operation details to the server
	ro.Inject(propagation.HeaderCarrier(r.Header))

	// Execute request and report errors
	res, err := ct.next.RoundTrip(r)
	if err != nil && !errors.Is(err, io.EOF) {
		ro.Status(getStatus(res.StatusCode))
		ro.Report(err)
	}
	return res, err
}
