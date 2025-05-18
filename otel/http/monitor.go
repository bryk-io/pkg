package otelhttp

import (
	"fmt"
	"net/http"

	contrib "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

// Monitor provide easy-to-use instrumentation primitives for HTTP clients
// and servers.
type Monitor interface {
	// Client provides an HTTP client interface with automatic instrumentation
	// of requests.
	Client(base http.RoundTripper) http.Client

	// RoundTripper wraps the provided `http.RoundTripper` with one that starts a span,
	// injects the span context into the outbound request headers, and enriches it
	// with metrics. If the provided `base` http.RoundTripper is nil, `http.DefaultTransport`
	// will be used by default.
	RoundTripper(base http.RoundTripper) http.RoundTripper

	// Handler adds instrumentation to the provided HTTP handler using the
	// `operation` value provided as the span name.
	Handler(operation string, handler http.Handler) http.Handler

	// HandlerFunc adds instrumentation to the provided HTTP handler function
	// using the `operation` value provided as the span name.
	HandlerFunc(operation string, fn http.HandlerFunc) http.Handler

	// ServerMiddleware provides a mechanism to easily instrument an HTTP
	// handler and automatically collect observability information for all
	// handled requests. Order is important when using middleware, try to
	// load observability support as early as possible.
	ServerMiddleware() func(http.Handler) http.Handler
}

type httpMonitor struct {
	nf SpanNameFormatter // span name formatter
	ft []Filter          // operation filters
	rt string            // report trace ID
	ev bool              // report events
}

// NewMonitor returns a ready to use monitor instance that can be used to
// easily instrument HTTP clients and servers.
func NewMonitor(opts ...Option) Monitor {
	mon := &httpMonitor{
		nf: spanNameFormatter,
		ft: []Filter{},
	}
	for _, opt := range opts {
		opt(mon)
	}
	return mon
}

func (e *httpMonitor) settings() []contrib.Option {
	// Propagator, metric provider and trace provider are taking from globals
	// setup during the otel.Operator initialization.
	opts := []contrib.Option{}
	if e.ev {
		opts = append(opts, contrib.WithMessageEvents(contrib.ReadEvents, contrib.WriteEvents))
	}
	for _, ft := range e.ft {
		opts = append(opts, contrib.WithFilter(contrib.Filter(ft)))
	}
	return opts
}

// Client provides an HTTP client interface with automatic instrumentation
// of requests.
func (e *httpMonitor) Client(base http.RoundTripper) http.Client {
	settings := append(e.settings(),
		contrib.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return e.nf(r)
		}),
	)
	if base == nil {
		base = http.DefaultTransport
	}
	return http.Client{
		Transport: contrib.NewTransport(base, settings...),
	}
}

// RoundTripper wraps the provided `http.RoundTripper` with one that starts a span,
// injects the span context into the outbound request headers, and enriches it
// with metrics. If the provided `base` http.RoundTripper is nil, `http.DefaultTransport`
// will be used by default.
func (e *httpMonitor) RoundTripper(base http.RoundTripper) http.RoundTripper {
	settings := append(e.settings(),
		contrib.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return e.nf(r)
		}),
	)
	return contrib.NewTransport(base, settings...)
}

// Handler adds instrumentation to the provided HTTP handler using the
// `operation` value provided as the span name.
func (e *httpMonitor) Handler(operation string, handler http.Handler) http.Handler {
	return contrib.NewHandler(e.getHandler(handler), operation, e.settings()...)
}

// HandlerFunc adds instrumentation to the provided HTTP handler function
// using the `operation` value provided as the span name.
func (e *httpMonitor) HandlerFunc(operation string, hf http.HandlerFunc) http.Handler {
	return contrib.NewHandler(e.getHandler(hf), operation, e.settings()...)
}

// ServerMiddleware provides a mechanism to easily instrument an HTTP
// handler and automatically collect observability information for all
// handled requests. Order is important when using middleware, try to
// load observability support as early as possible.
func (e *httpMonitor) ServerMiddleware() func(http.Handler) http.Handler {
	// Attach a custom span name formatter to differentiate spans based on the
	// HTTP method and path for each operation
	options := e.settings()
	options = append(options, contrib.WithSpanNameFormatter(func(_ string, r *http.Request) string {
		return e.nf(r)
	}))
	return func(handler http.Handler) http.Handler {
		return contrib.NewHandler(e.getHandler(handler), "", options...)
	}
}

// Returns a handler that, if enabled, reports the trace ID in the response.
func (e *httpMonitor) getHandler(next http.Handler) http.Handler {
	if e.rt == "" {
		return next
	}
	return reportTraceID(next, e.rt)
}

// Default span name formatter.
func spanNameFormatter(r *http.Request) string {
	return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
}

// Attach the trace identifier in a specified response header.
func reportTraceID(next http.Handler, h string) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		if span.SpanContext().IsValid() {
			w.Header().Set(h, span.SpanContext().TraceID().String())
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
