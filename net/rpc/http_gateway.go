package rpc

import (
	"net/http"
	"strings"
	"sync"

	gwRuntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
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

// HTTPGateway permits to consume an HTTP2 RPC-based service through a flexible HTTP1.1
// REST interface.
type HTTPGateway struct {
	port          int                               // TCP port
	customPaths   []customHandler                   // additional "routes" on the server
	encoders      map[string]gwRuntime.Marshaler    // custom encoding mechanisms
	middleware    []func(http.Handler) http.Handler // HTTP middleware
	interceptors  []HTTPGatewayInterceptor          // registered request interceptors
	responseMut   HTTPGatewayResponseMutator        // main response mutator
	unaryErrorMut HTTPGatewayUnaryErrorHandler      // unary error response mutator
	handlerName   string                            // gateway server name, used for observability
	conn          *grpc.ClientConn                  // internal connection to the underlying gRPC server
	clientOptions []ClientOption                    // internal gRPC client connection settings
	mu            sync.Mutex
}

// NewHTTPGateway setups an HTTP interface for an RPC server.
func NewHTTPGateway(options ...HTTPGatewayOption) (*HTTPGateway, error) {
	gw := &HTTPGateway{
		port:          0,
		clientOptions: []ClientOption{},
		customPaths:   []customHandler{},
		encoders:      make(map[string]gwRuntime.Marshaler),
		middleware:    []func(http.Handler) http.Handler{},
		interceptors:  []HTTPGatewayInterceptor{},
		handlerName:   "grpc-gateway",
	}
	if err := gw.setup(options...); err != nil {
		return nil, errors.Wrap(err, "setup error")
	}
	return gw, nil
}

func (gw *HTTPGateway) setup(options ...HTTPGatewayOption) error {
	for _, opt := range options {
		if err := opt(gw); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func (gw *HTTPGateway) connect(endpoint string) (err error) {
	gw.conn, err = NewClientConnection(endpoint, gw.clientOptions...)
	return err
}

func (gw *HTTPGateway) options() (opts []gwRuntime.ServeMuxOption) {
	// Encoders
	for mime, enc := range gw.encoders {
		opts = append(opts, gwRuntime.WithMarshalerOption(mime, enc))
	}

	// Preserve all (valid) incoming and outgoing HTTP headers as gRPC context
	// metadata by default
	opts = append(opts, gwRuntime.WithIncomingHeaderMatcher(preserveHeaders()))
	opts = append(opts, gwRuntime.WithOutgoingHeaderMatcher(preserveHeaders()))

	// Register response mutator
	if gw.responseMut != nil {
		opts = append(opts, gwRuntime.WithForwardResponseOption(gw.responseMut))
	}

	// Register error handler
	if gw.unaryErrorMut != nil {
		opts = append(opts, gwRuntime.WithErrorHandler(gwRuntime.ErrorHandlerFunc(gw.unaryErrorMut)))
	}

	return opts
}

func (gw *HTTPGateway) interceptorWrapper(h http.Handler, list []HTTPGatewayInterceptor) http.Handler {
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
		return strings.TrimRight(v, "\r\n"), isHeaderValid(strings.ToLower(v))
	}
}

func isHeaderValid(header string) bool {
	for _, h := range invalidHeaders {
		if h == header {
			return false
		}
	}
	return true
}
