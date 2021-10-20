package otel

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func defaultHTTPSettings(op *Operator) []otelhttp.Option {
	opts := []otelhttp.Option{
		otelhttp.WithPropagators(op.propagator),
		otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents),
	}
	if op.metricProvider != nil {
		opts = append(opts, otelhttp.WithMeterProvider(op.metricProvider))
	}
	return opts
}

// HTTPClient provides the interface of the regular HTTP client
// but with automatic instrumentation of requests.
func (op *Operator) HTTPClient(base http.RoundTripper, opts ...otelhttp.Option) http.Client {
	settings := append(defaultHTTPSettings(op),
		otelhttp.WithSpanNameFormatter(func(op string, r *http.Request) string {
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

// HTTPHandler adds instrumentation to the provided HTTP handler using the
// `operation` value provided as the span name.
func (op *Operator) HTTPHandler(operation string, handler http.Handler) http.Handler {
	options := defaultHTTPSettings(op)
	return otelhttp.NewHandler(handler, operation, options...)
}

// HTTPHandleFunc adds instrumentation to the provided HTTP handler function
// using the `operation` value provided as the span name.
func (op *Operator) HTTPHandleFunc(operation string, hf func(http.ResponseWriter, *http.Request)) http.Handler {
	options := defaultHTTPSettings(op)
	return otelhttp.NewHandler(http.HandlerFunc(hf), operation, options...)
}

// HTTPServerMiddleware provides a mechanism to easily instrument an HTTP
// handler and automatically collect observability information for all
// handled requests. Order is important when using middleware, try to load
// observability support as early as possible.
func (op *Operator) HTTPServerMiddleware(name string) func(http.Handler) http.Handler {
	options := defaultHTTPSettings(op)
	return func(handler http.Handler) http.Handler {
		return otelhttp.NewHandler(handler, name, options...)
	}
}
