package otelhttp

// Option elements provide a functional-style mechanism to adjust the HTTP
// monitor behavior.
type Option func(mon *httpMonitor)

// WithSpanNameFormatter allows to adjust how spans are reported.
func WithSpanNameFormatter(nf SpanNameFormatter) Option {
	return func(mon *httpMonitor) {
		mon.nf = nf
	}
}
