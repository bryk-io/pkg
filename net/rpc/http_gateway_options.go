package rpc

import (
	"context"
	"net/http"

	gwRuntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type customHandler struct {
	method string
	path   string
	hf     http.HandlerFunc
}

// HTTPGatewayOption allows adjusting gateway settings following a functional pattern.
type HTTPGatewayOption func(*HTTPGateway) error

// HTTPGatewayInterceptor allows to further customize the processing of requests.
// If an interceptor function returns a non-nil error, any further processing of
// the request will be skipped.
type HTTPGatewayInterceptor func(http.ResponseWriter, *http.Request) error

// HTTPGatewayResponseMutator allows the user to completely control/adjust the response
// returned by the gateway. Some common uses cases include:
//   - Return a subset of response fields as HTTP response headers
//   - Set an application-specific token in a header
//   - Mutate the response messages to be returned
type HTTPGatewayResponseMutator func(context.Context, http.ResponseWriter, proto.Message) error

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
func WithCustomHandlerFunc(method string, path string, hf http.HandlerFunc) HTTPGatewayOption {
	return func(gw *HTTPGateway) error {
		gw.mu.Lock()
		defer gw.mu.Unlock()
		gw.customPaths = append(gw.customPaths, customHandler{
			method: method,
			path:   path,
			hf:     hf,
		})
		return nil
	}
}

// WithClientOptions configuration options for the gateway's internal client connection
// to the upstream RPC server.
func WithClientOptions(options ...ClientOption) HTTPGatewayOption {
	return func(gw *HTTPGateway) error {
		gw.mu.Lock()
		defer gw.mu.Unlock()
		gw.clientOptions = append(gw.clientOptions, options...)
		return nil
	}
}

// WithEncoder registers a marshaler instance for a specific mime type.
func WithEncoder(mime string, marshaler gwRuntime.Marshaler) HTTPGatewayOption {
	return func(gw *HTTPGateway) error {
		gw.mu.Lock()
		defer gw.mu.Unlock()
		gw.encoders[mime] = marshaler
		return nil
	}
}

// WithInterceptor allows to customize the processing of requests. Interceptors are
// executed BEFORE the standard processing operations and could impact performance or
// prevent standard processing handled by the gateway instance. Should be used with
// care. If an interceptor returns a non-nil error, any further processing of the
// request will be skipped.
func WithInterceptor(f ...HTTPGatewayInterceptor) HTTPGatewayOption {
	return func(gw *HTTPGateway) error {
		gw.mu.Lock()
		defer gw.mu.Unlock()
		gw.interceptors = append(gw.interceptors, f...)
		return nil
	}
}

// WithResponseMutator allows the user to completely control/adjust the response
// returned by the gateway. Some common uses cases include:
//   - Return a subset of response fields as HTTP response headers
//   - Set an application-specific token in a header
//   - Mutate the response messages to be returned
func WithResponseMutator(rm HTTPGatewayResponseMutator) HTTPGatewayOption {
	return func(gw *HTTPGateway) error {
		gw.mu.Lock()
		defer gw.mu.Unlock()
		gw.responseMut = rm
		return nil
	}
}

// WithPrettyJSON provides a convenient mechanism to enable pretty printed JSON
// responses for requests with a specific content-type header. A usual value to
// use is `application/json+pretty`.
func WithPrettyJSON(mime string) HTTPGatewayOption {
	return func(gw *HTTPGateway) error {
		jm := &gwRuntime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames:   true,
				UseEnumNumbers:  true,
				EmitUnpopulated: false,
				Indent:          "  ",
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}
		gw.mu.Lock()
		defer gw.mu.Unlock()
		gw.encoders[mime] = jm
		return nil
	}
}

// WithHandlerName will adjust the OpenTelemetry name used to report spans generated
// by the HTTP gateway instance. If not provided the default name `grpc-gateway`
// will be used.
func WithHandlerName(name string) HTTPGatewayOption {
	return func(gw *HTTPGateway) error {
		gw.handlerName = name
		return nil
	}
}
