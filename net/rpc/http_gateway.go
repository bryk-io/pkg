package rpc

import (
	"net/http"
	"strings"
	"sync"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
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
	clientOptions []ClientOption
	port          int
	customPathsF  map[string]http.HandlerFunc
	customPathsH  map[string]http.Handler
	encoders      map[string]gwruntime.Marshaler
	middleware    []func(http.Handler) http.Handler
	filters       []HTTPGatewayFilter
	handlerName   string
	mu            sync.Mutex
}

// NewHTTPGateway setups an HTTP interface for an RPC server.
func NewHTTPGateway(options ...HTTPGatewayOption) (*HTTPGateway, error) {
	gw := &HTTPGateway{
		port:          0,
		clientOptions: []ClientOption{},
		customPathsF:  make(map[string]http.HandlerFunc),
		customPathsH:  make(map[string]http.Handler),
		encoders:      make(map[string]gwruntime.Marshaler),
		middleware:    []func(http.Handler) http.Handler{},
		filters:       []HTTPGatewayFilter{},
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

func (gw *HTTPGateway) dialOption() (grpc.DialOption, error) {
	cl, err := NewClient(gw.clientOptions...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to acquire HTTP gateway's internal client instance")
	}
	if cl.tlsConf == nil {
		return grpc.WithTransportCredentials(insecure.NewCredentials()), nil
	}
	return grpc.WithTransportCredentials(credentials.NewTLS(cl.tlsConf)), nil
}

func (gw *HTTPGateway) options() (opts []gwruntime.ServeMuxOption) {
	// Encoders
	for mime, enc := range gw.encoders {
		opts = append(opts, gwruntime.WithMarshalerOption(mime, enc))
	}

	// Preserve all incoming and outgoing HTTP headers as gRPC context
	// metadata by default.
	opts = append(opts, gwruntime.WithIncomingHeaderMatcher(preserveHeaders()))
	opts = append(opts, gwruntime.WithOutgoingHeaderMatcher(preserveHeaders()))

	return opts
}

func (gw *HTTPGateway) filterWrapper(h http.Handler, filters []HTTPGatewayFilter) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		for _, f := range filters {
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
