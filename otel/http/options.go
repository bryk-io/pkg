package otelhttp

import (
	apiErrors "go.bryk.io/pkg/otel/errors"
)

// Option elements provide a functional-style mechanism to adjust the HTTP
// monitor behavior.
type Option func(mon *httpMonitor)

// WithSpanNameFormatter allows to adjust how spans are reported.
func WithSpanNameFormatter(nf SpanNameFormatter) Option {
	return func(mon *httpMonitor) {
		mon.nf = nf
	}
}

// WithErrorReporter allow to collect telemetry about exceptions occurring on
// the client and server side.
func WithErrorReporter(rep apiErrors.Reporter) Option {
	return func(mon *httpMonitor) {
		mon.rep = rep
	}
}
