package otelhttp

import (
	"fmt"
	"net/http"

	contrib "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Monitor provide easy-to-use instrumentation primitives for HTTP clients
// and servers.
type Monitor interface {
	// Client provides an HTTP client interface with automatic instrumentation
	// of requests.
	Client(base http.RoundTripper) http.Client

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
	nf SpanNameFormatter
}

// NewMonitor returns a ready to use monitor instance that can be used to
// easily instrument HTTP clients and servers.
func NewMonitor(opts ...Option) Monitor {
	mon := &httpMonitor{
		nf: spanNameFormatter,
	}
	for _, opt := range opts {
		opt(mon)
	}
	return mon
}

func (e *httpMonitor) settings() []contrib.Option {
	// Propagator, metric provider and trace provider are taking from globals
	// setup during the otel.Operator initialization.
	return []contrib.Option{
		contrib.WithMessageEvents(contrib.ReadEvents, contrib.WriteEvents),
	}
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

// Handler adds instrumentation to the provided HTTP handler using the
// `operation` value provided as the span name.
func (e *httpMonitor) Handler(operation string, handler http.Handler) http.Handler {
	return contrib.NewHandler(handler, operation, e.settings()...)
}

// HandlerFunc adds instrumentation to the provided HTTP handler function
// using the `operation` value provided as the span name.
func (e *httpMonitor) HandlerFunc(operation string, hf http.HandlerFunc) http.Handler {
	return contrib.NewHandler(hf, operation, e.settings()...)
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
		return contrib.NewHandler(handler, "", options...)
	}
}

// Default span name formatter.
func spanNameFormatter(r *http.Request) string {
	return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
}
