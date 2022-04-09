package extras

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// HTTPMonitor provide easy-to-use instrumentation primitives for HTTP clients
// and servers.
type HTTPMonitor interface {
	// Client provides an HTTP client interface with automatic instrumentation of requests.
	Client(base http.RoundTripper, opts ...otelhttp.Option) http.Client

	// Handler adds instrumentation to the provided HTTP handler using the
	// `operation` value provided as the span name.
	Handler(operation string, handler http.Handler) http.Handler

	// HandleFunc adds instrumentation to the provided HTTP handler function
	// using the `operation` value provided as the span name.
	HandleFunc(operation string, fn http.HandlerFunc) http.Handler

	// ServerMiddleware provides a mechanism to easily instrument an HTTP handler and
	// automatically collect observability information for all handled requests. Order is
	// important when using middleware, try to load observability support as early as possible.
	ServerMiddleware(name string) func(http.Handler) http.Handler
}

type httpMonitor struct{}

// NewHTTPMonitor returns a ready to use monitor instance that can be used to
// easily instrument HTTP clients and servers.
func NewHTTPMonitor() HTTPMonitor {
	return &httpMonitor{}
}

func (e *httpMonitor) settings() []otelhttp.Option {
	// Propagator, metric provider and trace provider are taking from globals
	// setup during the otel.Operator initialization.
	return []otelhttp.Option{
		otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents),
	}
}

// Client provides an HTTP client interface with automatic instrumentation of requests.
func (e *httpMonitor) Client(base http.RoundTripper, opts ...otelhttp.Option) http.Client {
	settings := append(e.settings(),
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		}),
	)
	settings = append(settings, opts...)
	if base == nil {
		base = http.DefaultTransport
	}
	return http.Client{
		Transport: otelhttp.NewTransport(base, settings...),
	}
}

// Handler adds instrumentation to the provided HTTP handler using the
// `operation` value provided as the span name.
func (e *httpMonitor) Handler(operation string, handler http.Handler) http.Handler {
	return otelhttp.NewHandler(handler, operation, e.settings()...)
}

// HandleFunc adds instrumentation to the provided HTTP handler function
// using the `operation` value provided as the span name.
func (e *httpMonitor) HandleFunc(operation string, hf http.HandlerFunc) http.Handler {
	return otelhttp.NewHandler(hf, operation, e.settings()...)
}

// ServerMiddleware provides a mechanism to easily instrument an HTTP handler and
// automatically collect observability information for all handled requests. Order is
// important when using middleware, try to load observability support as early as possible.
func (e *httpMonitor) ServerMiddleware(name string) func(http.Handler) http.Handler {
	// Attach a custom span name formatter to differentiate spans based on the
	// HTTP method and path for each operation
	options := e.settings()
	options = append(options, otelhttp.WithSpanNameFormatter(func(op string, r *http.Request) string {
		return fmt.Sprintf("%s %s %s", op, r.Method, r.URL.Path)
	}))
	return func(handler http.Handler) http.Handler {
		return otelhttp.NewHandler(handler, name, options...)
	}
}
