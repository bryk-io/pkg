package rpc

import (
	"net/http"
	"slices"
	"strings"
	"sync"

	gwRuntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.bryk.io/pkg/errors"
	otelHttp "go.bryk.io/pkg/otel/http"
	"google.golang.org/grpc"
)

// Invalid HTTP2 headers
// https://datatracker.ietf.org/doc/html/rfc7540#section-8.1.2.2
var invalidHeaders = []string{
	"connection",
	"keep-alive",
	"proxy-connection",
	"transfer-encoding",
	"upgrade",
}

// Gateway permits to consume HTTP2 RPC-based services through a flexible HTTP1.1
// REST interface.
type Gateway struct {
	port          int                               // TCP port
	customPaths   []customHandler                   // additional "routes" on the server
	encoders      map[string]gwRuntime.Marshaler    // custom encoding mechanisms
	middleware    []func(http.Handler) http.Handler // HTTP middleware
	interceptors  []GatewayInterceptor              // registered request interceptors
	responseMut   GatewayResponseMutator            // main response mutator
	unaryErrorMut GatewayUnaryErrorHandler          // unary error response mutator
	handlerName   string                            // gateway server name, used for observability
	conn          *grpc.ClientConn                  // internal connection to the underlying gRPC server
	clientOptions []ClientOption                    // internal gRPC client connection settings
	spanFormatter otelHttp.SpanNameFormatter        // otel span name formatter
	mu            sync.Mutex
}

// NewGateway setups an HTTP interface for an RPC server.
func NewGateway(options ...GatewayOption) (*Gateway, error) {
	gw := &Gateway{
		port:          0,
		clientOptions: []ClientOption{},
		customPaths:   []customHandler{},
		middleware:    []func(http.Handler) http.Handler{},
		interceptors:  []GatewayInterceptor{},
		handlerName:   "grpc-gateway",
		encoders:      map[string]gwRuntime.Marshaler{},
	}
	if err := gw.setup(options...); err != nil {
		return nil, errors.Wrap(err, "setup error")
	}
	return gw, nil
}

func (gw *Gateway) setup(options ...GatewayOption) error {
	for _, opt := range options {
		if err := opt(gw); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func (gw *Gateway) connect(endpoint string) (err error) {
	gw.conn, err = NewClientConnection(endpoint, gw.clientOptions...)
	return err
}

func (gw *Gateway) options() (opts []gwRuntime.ServeMuxOption) {
	// encoders
	for mime, enc := range gw.encoders {
		opts = append(opts, gwRuntime.WithMarshalerOption(mime, enc))
	}

	// preserve all (valid) incoming and outgoing HTTP headers as gRPC context
	// metadata by default
	opts = append(opts, gwRuntime.WithIncomingHeaderMatcher(preserveHeaders()))
	opts = append(opts, gwRuntime.WithOutgoingHeaderMatcher(preserveHeaders()))

	// if set, register response mutator
	if gw.responseMut != nil {
		opts = append(opts, gwRuntime.WithForwardResponseOption(gw.responseMut))
	}

	// if set, register error handler
	if gw.unaryErrorMut != nil {
		opts = append(opts, gwRuntime.WithErrorHandler(gwRuntime.ErrorHandlerFunc(gw.unaryErrorMut)))
	}

	return opts
}

func (gw *Gateway) interceptorWrapper(h http.Handler, list []GatewayInterceptor) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		for _, f := range list {
			if err := f(res, req); err != nil {
				return
			}
		}
		h.ServeHTTP(res, req)
	})
}

func preserveHeaders() func(v string) (string, bool) {
	return func(v string) (string, bool) {
		return strings.TrimRight(v, "\r\n"), !slices.Contains(invalidHeaders, strings.ToLower(v))
	}
}
