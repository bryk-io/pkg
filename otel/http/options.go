package otelhttp

import "net/http"

// Option elements provide a functional-style mechanism to adjust the HTTP
// monitor behavior.
type Option func(mon *httpMonitor)

// SpanNameFormatter allows to adjust how a given transaction is reported
// when handling an HTTP request on the client or server side.
type SpanNameFormatter func(r *http.Request) string

// WithSpanNameFormatter allows to adjust how spans are reported.
func WithSpanNameFormatter(nf SpanNameFormatter) Option {
	return func(mon *httpMonitor) {
		mon.nf = nf
	}
}
