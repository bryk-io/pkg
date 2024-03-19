package otelhttp

import (
	"net/http"
)

// Option elements provide a functional-style mechanism to adjust the HTTP
// monitor behavior.
type Option func(mon *httpMonitor)

// Filter is a predicate used to determine whether a given http.request
// should be traced. A Filter must return true if the request should be
// traced.
type Filter func(*http.Request) bool

// SpanNameFormatter allows to adjust how a given transaction is reported
// when handling an HTTP request on the client or server side.
type SpanNameFormatter func(r *http.Request) string

// WithSpanNameFormatter allows to adjust how spans are reported.
func WithSpanNameFormatter(nf SpanNameFormatter) Option {
	return func(mon *httpMonitor) {
		mon.nf = nf
	}
}

// WithNetworkEvents instructs the monitor to collect read and
// write network events. These events are discarded by default.
func WithNetworkEvents() Option {
	return func(mon *httpMonitor) {
		mon.ev = true
	}
}

// WithTraceInHeader allows to set a custom header to report the
// transaction ID. The server will use this header to report the trace
// ID to the client.
func WithTraceInHeader(h string) Option {
	return func(mon *httpMonitor) {
		mon.rt = h
	}
}

// WithFilter adds a filter function to the monitor. If any filter
// indicates to exclude a request then the request will not be traced.
// All filters must allow a request to be traced for a Span to be created.
// If no filters are provided then all requests are traced. Filters will
// be invoked for each processed request, it is advised to make them
// simple and fast.
func WithFilter(f Filter) Option {
	return func(mon *httpMonitor) {
		mon.ft = append(mon.ft, f)
	}
}
