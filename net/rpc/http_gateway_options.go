package rpc

import (
	"net/http"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
)

// HTTPGatewayOption allows adjusting gateway settings following a functional pattern.
type HTTPGatewayOption func(*HTTPGateway) error

// HTTPGatewayFilter allows to further customize the processing of requests.
// If a filter function returns a non-nil error, any further processing of
// the request will be skipped.
type HTTPGatewayFilter func(http.ResponseWriter, *http.Request) error

// WithGatewayPort adjust the gateway to handle requests on a different port. If not
// set the gateway will use the same port as the RPC server by default. If a custom and
// different port is provided, the gateway will manage its own network interface. If
// the RPC endpoint is a UNIX socket and no port is provided for the gateway, a free
// port number will be randomly assigned.
func WithGatewayPort(port int) HTTPGatewayOption {
	return func(gw *HTTPGateway) error {
		gw.mu.Lock()
		defer gw.mu.Unlock()
		gw.port = port
		return nil
	}
}

// WithGatewayMiddleware allows extending and adjusting the behavior of the
// HTTP gateway with standard middleware providers.
func WithGatewayMiddleware(md func(http.Handler) http.Handler) HTTPGatewayOption {
	return func(gw *HTTPGateway) error {
		gw.mu.Lock()
		defer gw.mu.Unlock()
		gw.middleware = append(gw.middleware, md)
		return nil
	}
}

// WithCustomHandlerFunc add a new handler function for a path on the gateway's
// internal mux.
func WithCustomHandlerFunc(path string, handler http.HandlerFunc) HTTPGatewayOption {
	return func(gw *HTTPGateway) error {
		gw.mu.Lock()
		defer gw.mu.Unlock()
		if gw.customPathsF == nil {
			gw.customPathsF = make(map[string]http.HandlerFunc)
		}
		gw.customPathsF[path] = handler
		return nil
	}
}

// WithCustomHandler add a new handler for a path on the gateway's internal mux.
func WithCustomHandler(path string, handler http.Handler) HTTPGatewayOption {
	return func(gw *HTTPGateway) error {
		gw.mu.Lock()
		defer gw.mu.Unlock()
		if gw.customPathsH == nil {
			gw.customPathsH = make(map[string]http.Handler)
		}
		gw.customPathsH[path] = handler
		return nil
	}
}

// WithClientOptions configuration options for the gateway's internal client connection
// to the upstream RPC server.
func WithClientOptions(options []ClientOption) HTTPGatewayOption {
	return func(gw *HTTPGateway) error {
		gw.mu.Lock()
		defer gw.mu.Unlock()
		gw.clientOptions = options
		return nil
	}
}

// WithEncoder registers a marshaler instance for a specific mime type.
func WithEncoder(mime string, marshaler gwruntime.Marshaler) HTTPGatewayOption {
	return func(gw *HTTPGateway) error {
		gw.mu.Lock()
		defer gw.mu.Unlock()
		gw.encoders[mime] = marshaler
		return nil
	}
}

// WithFilter allows to customize the processing of requests. Filter functions are
// executed BEFORE the standard processing operations and could impact performance or
// prevent standard processing handled by the gateway instance. Should be used with
// care. If a filter function returns a non-nil error, any further processing of the
// request will be skipped.
func WithFilter(f ...HTTPGatewayFilter) HTTPGatewayOption {
	return func(gw *HTTPGateway) error {
		gw.mu.Lock()
		defer gw.mu.Unlock()
		gw.filters = append(gw.filters, f...)
		return nil
	}
}

// WithPrettyJSON provides a convenient mechanism to enable pretty printed JSON
// responses for requests with a specific content-type header. A usual value to
// use is `application/json+pretty`.
func WithPrettyJSON(mime string) HTTPGatewayOption {
	return func(gw *HTTPGateway) error {
		jm := &gwruntime.JSONPb{
			OrigName:     true,
			EnumsAsInts:  false,
			EmitDefaults: false,
			Indent:       "  ",
			AnyResolver:  nil,
		}
		gw.mu.Lock()
		defer gw.mu.Unlock()
		gw.encoders[mime] = jm
		return nil
	}
}
