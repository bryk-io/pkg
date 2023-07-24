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

	mwAuth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	mwRecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	mwValidator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	gwRuntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/soheilhy/cmux"
	"go.bryk.io/pkg/errors"
	"go.bryk.io/pkg/net/rpc/ws"
	otelGrpc "go.bryk.io/pkg/otel/grpc"
	otelHttp "go.bryk.io/pkg/otel/http"
	otelProm "go.bryk.io/pkg/otel/prometheus"
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

// Server provides an easy-to-setup RPC server handler with several utilities.
type Server struct {
	tlsOptions       ServerTLSConfig                // TLS settings
	services         []ServiceProvider              // Services enabled on the server
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
	gateway          *Gateway                       // HTTP gateway
	port             int                            // TCP port, if used
	tlsConfig        *tls.Config                    // TLS configuration
	tokenValidator   authFunc                       // Custom method to provide token-based authenticator
	grpc             *grpc.Server                   // gRPC server instance
	gw               *http.Server                   // Gateway HTTP server
	halt             context.CancelFunc             // Stop all internal processing
	wsProxy          *ws.Proxy                      // WebSocket proxy
	resourceLimits   ResourceLimits                 // Settings to prevent resources abuse
	panicRecovery    bool                           // Enable panic recovery interceptor
	inputValidation  bool                           // Enable automatic input validation
	reflection       bool                           // Enable server reflection protocol
	prometheus       otelProm.Operator              // Prometheus support
	mu               sync.Mutex
}

// NewServer is a constructor method that returns a ready-to-use new server instance.
func NewServer(options ...ServerOption) (*Server, error) {
	srv := &Server{}
	if err := srv.setup(options...); err != nil {
		return nil, errors.Wrap(err, "setup error")
	}
	return srv, nil
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

// Stop will terminate the server processing. When graceful is true, the server
// stops accepting new connections and requests and blocks until all the pending
// RPCs are finished. Otherwise, it cancels all active RPCs on the server side and
// the corresponding pending RPCs on the client side will get notified by connection
// errors.
func (srv *Server) Stop(graceful bool) error {
	// Nothing to do
	if srv.halt == nil {
		return nil
	}
	srv.halt()

	// Close HTTP gateway
	var e error
	if srv.gw != nil {
		if err := srv.gw.Shutdown(context.Background()); err != nil {
			e = errors.Wrap(err, "shutdown HTTP gateway")
		}
		if err := srv.gateway.conn.Close(); err != nil {
			e = errors.Wrap(err, "shutdown HTTP gateway connection")
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

// Start the server and wait for incoming requests. You can provide an optional
// notification handler to catch an event when the server is ready for use. If
// a handler is provided but poorly managed, the start process will continue after
// a timeout of 20 milliseconds to prevent blocking the process indefinitely.
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
	srv.opts = append(srv.opts, grpc.ChainUnaryInterceptor(unaryM...))
	srv.opts = append(srv.opts, grpc.ChainStreamInterceptor(streamM...))

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
	if srv.prometheus != nil {
		srv.prometheus.InitializeMetrics(srv.grpc)
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

// Reset will remove any previously configuration options returning the server
// to its default state.
func (srv *Server) reset() {
	if srv.halt != nil {
		srv.halt()
	}
	srv.ctx, srv.halt = context.WithCancel(context.Background())
	srv.net = netTCP
	srv.port = 12137
	srv.services = []ServiceProvider{}
	srv.address = "127.0.0.1"
	srv.tlsConfig = nil
	srv.clientCAs = [][]byte{}
	srv.panicRecovery = false
	srv.inputValidation = false
	srv.gateway = nil
	srv.opts = []grpc.ServerOption{}
	srv.middlewareUnary = []grpc.UnaryServerInterceptor{}
	srv.middlewareStream = []grpc.StreamServerInterceptor{}
	srv.prometheus = nil
	srv.tokenValidator = nil
}

// Setup will remove any existing setting and apply the provided configuration options.
func (srv *Server) setup(options ...ServerOption) error {
	srv.reset()
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

	// Start HTTP gateway using its own network listener
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

// Configure the main server's network interface with proper resource limits and
// TLS settings.
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
	// Verify if there's a gateway instance to set up
	if srv.gateway == nil {
		return nil
	}

	// Setup gateway network interface
	if err := srv.setupGatewayInterface(); err != nil {
		return err
	}

	// Establish gateway <-> server connection
	if err := srv.gateway.connect(srv.GetEndpoint()); err != nil {
		return err
	}

	// Gateway mux
	gwMux := gwRuntime.NewServeMux(srv.gateway.options()...)
	var gwMuxH = http.Handler(gwMux) // cast gateway mux as regular HTTP handler
	for _, s := range srv.services {
		hs, ok := s.(HTTPServiceProvider)
		if !ok || hs.GatewaySetup() == nil {
			// skip if the service doesn't provide a gateway setup function
			continue
		}
		if err := hs.GatewaySetup()(srv.ctx, gwMux, srv.gateway.conn); err != nil {
			return errors.Wrap(err, "HTTP gateway setup error")
		}
	}

	// Apply gateway interceptors
	if len(srv.gateway.interceptors) > 0 {
		gwMuxH = srv.gateway.interceptorWrapper(gwMuxH, srv.gateway.interceptors)
	}

	// Add custom paths
	for _, chf := range srv.gateway.customPaths {
		_ = gwMux.HandlePath(chf.method, chf.path, func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
			chf.hf(w, r)
		})
	}

	// Gateway middleware
	var gmw []func(http.Handler) http.Handler

	// Add OTEL as the first middleware in the chain automatically
	hmOpts := []otelHttp.Option{}
	if srv.gateway.spanFormatter != nil {
		hmOpts = append(hmOpts, otelHttp.WithSpanNameFormatter(srv.gateway.spanFormatter))
	}
	hm := otelHttp.NewMonitor(hmOpts...)
	gmw = append(gmw, hm.ServerMiddleware())
	for _, m := range append(gmw, srv.gateway.middleware...) {
		gwMuxH = m(gwMuxH)
	}

	// WebSocket support
	if srv.wsProxy != nil {
		gwMuxH = srv.wsProxy.Wrap(gwMuxH)
	}

	// Setup gateway server
	srv.mu.Lock()
	srv.gw = &http.Server{
		Handler:           gwMuxH,
		MaxHeaderBytes:    1024,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		IdleTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}
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

	// Set up a new network interface if the gateway uses a different TCP port
	// or if the main RPC server is using a UNIX socket as endpoint
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
	ui, si := otelGrpc.NewMonitor().Server()
	unary = append(unary, ui)
	stream = append(stream, si)

	// Setup prometheus metrics before any functional middleware
	if srv.prometheus != nil {
		ui, si := srv.prometheus.Server()
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
