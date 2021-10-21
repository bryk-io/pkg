package rpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	mwAuth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	mwRecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	mwValidator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	gwRuntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/soheilhy/cmux"
	"go.bryk.io/pkg/net/rpc/ws"
	"go.bryk.io/pkg/otel"
	"golang.org/x/net/netutil"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	// Import the gzip package to automatically register the compressor method
	// when initializing a server instance.
	_ "google.golang.org/grpc/encoding/gzip"
)

const netTCP = "tcp"
const netUNIX = "unix"

type authFunc func(ctx context.Context) (context.Context, error)

// Server provides a easy-to-setup usable service handler with several utilities.
type Server struct {
	tlsOptions       ServerTLSConfig                // TLS settings
	services         []*Service                     // Services enabled on the server
	clientCAs        [][]byte                       // Custom CAs used for client authentication
	middlewareUnary  []grpc.UnaryServerInterceptor  // Unary methods middleware
	middlewareStream []grpc.StreamServerInterceptor // Stream methods middleware
	customUnary      []grpc.UnaryServerInterceptor  // Unary middleware provided by the user
	customStream     []grpc.StreamServerInterceptor // Stream middleware provided by the user
	opts             []grpc.ServerOption            // gRPC server options
	net              string                         // Network spaced used by the server (tcp or unix)
	netInterface     string                         // Name of the main network interface
	address          string                         // Main server address
	cm               cmux.CMux                      // Main multiplexer to use when using 2 network interfaces
	nl               net.Listener                   // Base RPC network interface
	ctx              context.Context                // Context shared by server's internal tasks
	gwNl             net.Listener                   // HTTP gateway network interface, if required
	gateway          *HTTPGateway                   // HTTP gateway
	port             int                            // TCP port, if used
	tlsConfig        *tls.Config                    // TLS configuration
	tokenValidator   authFunc                       // Custom method to provide token-based authenticator
	grpc             *grpc.Server                   // gRPC server instance
	gw               *http.Server                   // Gateway HTTP server
	halt             context.CancelFunc             // Stop all internal processing
	wsProxy          *ws.Proxy                      // WebSocket proxy
	oop              *otel.Operator                 // Handle observability requirements
	resourceLimits   ResourceLimits                 // Settings to prevent resources abuse
	panicRecovery    bool                           // Enable panic recovery interceptor
	inputValidation  bool                           // Enable automatic input validation
	reflection       bool                           // Enable server reflection protocol
	mu               sync.Mutex
}

// NewServer is a constructor method that returns a ready-to-use new server instance.
func NewServer(options ...ServerOption) (*Server, error) {
	srv := &Server{}
	if err := srv.Setup(options...); err != nil {
		return nil, errors.Wrap(err, "setup error")
	}
	return srv, nil
}

// Reset will remove any previously configuration options returning the server
// to its default state.
func (srv *Server) Reset() {
	if srv.halt != nil {
		srv.halt()
	}
	srv.ctx, srv.halt = context.WithCancel(context.TODO())
	srv.net = netTCP
	srv.port = 12137
	srv.services = []*Service{}
	srv.address = "127.0.0.1"
	srv.tlsConfig = nil
	srv.clientCAs = [][]byte{}
	srv.panicRecovery = false
	srv.inputValidation = false
	srv.gateway = nil
	srv.opts = []grpc.ServerOption{}
	srv.middlewareUnary = []grpc.UnaryServerInterceptor{}
	srv.middlewareStream = []grpc.StreamServerInterceptor{}
	srv.tokenValidator = nil
}

// Setup will remove any existing setting and apply the provided configuration options.
func (srv *Server) Setup(options ...ServerOption) error {
	srv.Reset()
	for _, opt := range options {
		if err := opt(srv); err != nil {
			return errors.WithStack(err)
		}
	}

	// Additional TLS configuration
	if srv.tlsConfig != nil && len(srv.clientCAs) > 0 {
		cp := x509.NewCertPool()
		for _, c := range srv.clientCAs {
			if !cp.AppendCertsFromPEM(c) {
				return errors.New("failed to append provided CA certificates")
			}
		}
		srv.tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		srv.tlsConfig.ClientCAs = cp
	}
	return nil
}

// GetEndpoint returns the server's main entry point.
func (srv *Server) GetEndpoint() string {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.net == netUNIX {
		return fmt.Sprintf("%s://%s", netUNIX, srv.address)
	}
	return fmt.Sprintf("%s:%d", srv.address, srv.port)
}

// Stop will terminate the server processing. When graceful is true, the server stops
// accepting new connections and RPCs and blocks until all the pending RPCs are finished.
// Otherwise it cancels all active RPCs on the server side and the corresponding pending
// RPCs on the client side will get notified by connection errors.
func (srv *Server) Stop(graceful bool) error {
	// Nothing to do
	if srv.halt == nil {
		return nil
	}
	srv.halt()

	// Close HTTP gateway
	var e error
	if srv.gw != nil {
		if err := srv.gw.Shutdown(context.TODO()); err != nil {
			e = errors.WithMessage(err, "failed to shutdown HTTP gateway")
		}
	}

	// Stop RPC server
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if graceful {
		srv.grpc.GracefulStop()
	} else {
		srv.grpc.Stop()
	}

	// Close gateway network interface
	if srv.gwNl != nil {
		if err := srv.gwNl.Close(); err != nil {
			e = errors.Wrap(e, err.Error())
		}
	}

	// Close main network interface
	if err := srv.nl.Close(); err != nil {
		var terr *net.OpError
		if !errors.As(err, &terr) || terr.Op != "close" {
			e = errors.Wrap(e, err.Error())
		}
	}
	return errors.Wrap(e, "stop error")
}

// Start the server and wait for incoming requests. An optional notification handler to catch
// an event when the server is ready for use. If a handler is provided but poorly managed the
// start process will continue after a timeout of 20 milliseconds to prevent blocking the
// process indefinitely.
func (srv *Server) Start(ready chan<- bool) (err error) {
	// In case of errors, close the provided notification channel as cleanup
	var cancel = func() {
		if ready != nil {
			close(ready)
		}
	}

	// Validate RPC services are provided
	if len(srv.services) == 0 {
		defer cancel()
		srv.halt()
		return errors.New("no services registered")
	}

	// Add middleware
	unaryM, streamM := srv.getMiddleware()
	srv.opts = append(srv.opts, grpc_middleware.WithUnaryServerChain(unaryM...))
	srv.opts = append(srv.opts, grpc_middleware.WithStreamServerChain(streamM...))

	// Create RPC instance and setup services
	srv.mu.Lock()
	srv.grpc = grpc.NewServer(srv.opts...)
	for _, s := range srv.services {
		s.ServerSetup(srv.grpc)
	}
	srv.mu.Unlock()

	// Enable reflection protocol
	if srv.reflection {
		reflection.Register(srv.grpc)
	}

	// Initialize server metrics
	if srv.oop != nil {
		srv.oop.PrometheusInitializeServer(srv.grpc)
	}

	// Setup main server network interface
	if srv.nl, err = srv.setupNetworkInterface(srv.net, srv.getAddress()); err != nil {
		defer cancel()
		srv.halt()
		return errors.Wrap(err, "failed to setup main network interface")
	}

	// Setup HTTP gateway
	if err = srv.setupGateway(); err != nil {
		defer cancel()
		srv.halt()
		return errors.Wrap(err, "failed to setup HTTP gateway")
	}

	// Start network handlers
	return errors.Wrap(srv.start(ready, 20*time.Millisecond), "failed to start request processing")
}

// Start server's network handlers.
func (srv *Server) start(ready chan<- bool, timeout time.Duration) error {
	// Setup main multiplexer and sub-tasks group
	var tasks errgroup.Group
	srv.cm = cmux.New(srv.nl)

	// Start gRPC server
	http2Matcher := cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc")
	grpcL := srv.cm.MatchWithWriters(http2Matcher)
	tasks.Go(func() error {
		return errors.Wrap(srv.grpc.Serve(grpcL), "grpc server error")
	})

	// Start HTTP gateway using it's own network listener
	if srv.gwNl != nil {
		tasks.Go(func() error {
			return errors.Wrap(srv.gw.Serve(srv.gwNl), "HTTP gateway with custom network listener error")
		})
	}

	// Start HTTP gateway using the main multiplexer
	if srv.gwNl == nil && srv.gw != nil {
		httpL := srv.cm.Match(cmux.HTTP1Fast())
		tasks.Go(func() error {
			return errors.Wrap(srv.gw.Serve(httpL), "HTTP gateway error")
		})
	}

	// Start main multiplexer processing
	tasks.Go(func() error {
		return errors.Wrap(srv.cm.Serve(), "failed to start main multiplexer")
	})

	// Setup notification handler
	if ready != nil {
		go func() {
			select {
			case ready <- true:
				// Continue after the notification has been received
				return
			case <-time.After(timeout):
				// Continue after timeout to prevent blocking the server due to a poorly manage handler
				return
			}
		}()
	}

	// Return any error from sub-tasks
	return errors.WithStack(tasks.Wait())
}

// Return the server's main interface address.
func (srv *Server) getAddress() string {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.net == netUNIX {
		return srv.address
	}
	return fmt.Sprintf("%s:%d", srv.address, srv.port)
}

// Configure the main server's network interface with proper resource limits and TLS settings.
func (srv *Server) setupNetworkInterface(network, address string) (net.Listener, error) {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	// Get network interface
	nl, err := net.Listen(network, address)
	if err != nil {
		return nil, errors.Errorf("failed to acquire network interface for %s on %s", network, address)
	}

	// Apply resource limits
	if srv.resourceLimits.Connections > 0 {
		nl = netutil.LimitListener(nl, int(srv.resourceLimits.Connections))
	}

	// Apply TLS configuration
	if srv.tlsConfig != nil {
		nl = tls.NewListener(nl, srv.tlsConfig)
	}
	return nl, nil
}

// Configure the server's HTTP gateway network interface and multiplexer.
func (srv *Server) setupGateway() error {
	// Verify if there's a gateway instance to setup
	if srv.gateway == nil {
		return nil
	}

	// Setup gateway network interface
	if err := srv.setupGatewayInterface(); err != nil {
		return err
	}

	// Internal client dial options
	authOpt, err := srv.gateway.dialOption()
	if err != nil {
		return errors.Wrap(err, "failed to get HTTP gateway's auth options")
	}

	// Gateway mux
	gwMux := gwRuntime.NewServeMux(srv.gateway.options()...)
	for _, s := range srv.services {
		if s.GatewaySetup != nil {
			if err := s.GatewaySetup(srv.ctx, gwMux, srv.GetEndpoint(), []grpc.DialOption{authOpt}); err != nil {
				return errors.Wrap(err, "HTTP gateway setup error")
			}
		}
	}

	// Base server mux and root handler
	mux := http.NewServeMux()
	handler := http.Handler(gwMux)

	// Add custom paths
	for path, hh := range srv.gateway.customPathsH {
		mux.Handle(path, hh)
	}
	for path, hf := range srv.gateway.customPathsF {
		mux.HandleFunc(path, hf)
	}

	// Apply gateway filters
	if len(srv.gateway.filters) > 0 {
		handler = srv.gateway.filterWrapper(handler, srv.gateway.filters)
	}

	// Gateway middleware
	var gmw []func(http.Handler) http.Handler
	if srv.oop != nil {
		// Add OTEL as the first middleware in the chain automatically
		gmw = append(gmw, srv.oop.HTTPServerMiddleware("grpc-gateway"))
	}
	for _, mw := range append(gmw, srv.gateway.middleware...) {
		handler = mw(handler)
	}

	// WebSocket support
	if srv.wsProxy != nil {
		handler = srv.wsProxy.Wrap(handler)
	}

	// Register root handler with base server mux
	mux.Handle("/", handler)

	// Setup gateway server
	srv.mu.Lock()
	srv.gw = &http.Server{Handler: mux}
	srv.mu.Unlock()

	// All good!
	return nil
}

// Prepare the HTTP gateway network interface when required.
func (srv *Server) setupGatewayInterface() error {
	// Use the same main server port by default if no port is provided
	if srv.net != netUNIX && srv.gateway.port == 0 {
		srv.gateway.port = srv.port
	}

	// Setup a new network interface if the gateway uses a different TCP port or if the
	// main RPC server is using a UNIX socket as endpoint
	if srv.net == netUNIX || srv.gateway.port != srv.port {
		addr := ""
		if srv.net != netUNIX {
			addr = srv.address
		}
		var err error
		srv.gwNl, err = srv.setupNetworkInterface(netTCP, fmt.Sprintf("%s:%d", addr, srv.gateway.port))
		if err != nil {
			return errors.Wrap(err, "failed to setup HTTP gateway network interface")
		}
	}
	return nil
}

// Return properly setup server middleware.
func (srv *Server) getMiddleware() (unary []grpc.UnaryServerInterceptor, stream []grpc.StreamServerInterceptor) {
	// Setup observability before anything else
	if srv.oop != nil {
		ui, si := srv.oop.RPCServer()
		unary = append(unary, ui)
		stream = append(stream, si)
	}

	// If enabled, token validator must be the first operational middleware in the chain
	if srv.tokenValidator != nil {
		unary = append(unary, mwAuth.UnaryServerInterceptor(mwAuth.AuthFunc(srv.tokenValidator)))
		stream = append(stream, mwAuth.StreamServerInterceptor(mwAuth.AuthFunc(srv.tokenValidator)))
	}

	// If enabled, input validation should be executed right after authentication
	if srv.inputValidation {
		unary = append(unary, mwValidator.UnaryServerInterceptor())
		stream = append(stream, mwValidator.StreamServerInterceptor())
	}

	// Add registered middleware
	unary = append(unary, srv.middlewareUnary...)
	stream = append(stream, srv.middlewareStream...)

	// Add custom middleware
	unary = append(unary, srv.customUnary...)
	stream = append(stream, srv.customStream...)

	// If enabled, panic recovery must be the last middleware to chain
	if srv.panicRecovery {
		unary = append(unary, mwRecovery.UnaryServerInterceptor())
		stream = append(stream, mwRecovery.StreamServerInterceptor())
	}
	return unary, stream
}
