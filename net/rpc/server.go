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
	otelgrpc "go.bryk.io/pkg/otel/grpc"
	"go.bryk.io/pkg/prometheus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	// import the gzip package to automatically register the compressor method
	// when initializing a server instance.
	_ "google.golang.org/grpc/encoding/gzip"
)

const netTCP string = "tcp"
const netUNIX string = "unix"

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
	net              string                         // Network space used by the server (tcp or unix)
	netInterface     string                         // Name of the main network interface
	address          string                         // Main server address
	cm               cmux.CMux                      // Main multiplexer, used when using 2 network interfaces
	nl               net.Listener                   // Base RPC network interface
	ctx              context.Context                // Context shared by server's internal tasks
	halt             context.CancelFunc             // Stop all internal processing
	gateway          *Gateway                       // HTTP gateway
	gatewayOpts      []GatewayOption                // HTTP gateway options
	gwNl             net.Listener                   // HTTP gateway network interface, if required
	port             int                            // TCP port, if used
	tlsConfig        *tls.Config                    // TLS configuration
	tokenValidator   authFunc                       // Custom method to provide token-based authenticator
	grpc             *grpc.Server                   // gRPC server instance
	gw               *http.Server                   // Gateway HTTP server
	wsProxy          *ws.Proxy                      // WebSocket proxy
	resourceLimits   ResourceLimits                 // Settings to prevent resources abuse
	enableValidator  bool                           // Enable protobuf validation
	panicRecovery    bool                           // Enable panic recovery interceptor
	inputValidation  bool                           // Enable automatic input validation
	reflection       bool                           // Enable server reflection protocol
	healthCheck      HealthCheck                    // Enable health checks
	prometheus       prometheus.Operator            // Prometheus support
	mu               sync.Mutex
}

// NewServer is a constructor method that returns a ready-to-use new server instance.
func NewServer(options ...ServerOption) (*Server, error) {
	srv := &Server{}
	if err := srv.setup(options...); err != nil {
		return nil, errors.Wrap(err, "setup error")
	}

	// enable server instrumentation by default
	srv.opts = append(srv.opts, otelgrpc.ServerInstrumentation())
	return srv, nil
}

// Endpoint returns the server's main entry point.
func (srv *Server) Endpoint() string {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.net == netUNIX {
		return fmt.Sprintf("%s://%s", netUNIX, srv.address)
	}
	return fmt.Sprintf("%s:%d", srv.address, srv.port)
}

// Stop the server processing. When graceful is true, the server  stops accepting new
// connections or requests and blocks until all the pending RPCs are finished. Otherwise,
// it cancels all active requests on the server side, the client side will get notified
// by connection errors.
func (srv *Server) Stop(graceful bool) error {
	// nothing to do
	if srv.halt == nil {
		return nil
	}

	// dispatch halt signal
	srv.mu.Lock()
	defer srv.mu.Unlock()
	srv.halt()

	// close HTTP gateway
	var e error
	if srv.gw != nil {
		if err := srv.gw.Shutdown(context.Background()); err != nil {
			e = errors.Wrap(err, "shutdown HTTP gateway")
		}
		if err := srv.gateway.conn.Close(); err != nil {
			e = errors.Wrap(err, "shutdown HTTP gateway connection")
		}
	}

	// stop RPC server
	if graceful {
		srv.grpc.GracefulStop()
	} else {
		srv.grpc.Stop()
	}

	// close gateway network interface
	if srv.gwNl != nil {
		if err := srv.gwNl.Close(); err != nil {
			e = errors.Wrap(e, err.Error())
		}
	}

	// close main network interface
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
	// in case of errors, close the provided notification channel as cleanup
	var cancel = func() {
		if ready != nil {
			close(ready)
		}
	}

	// enable health checks protocol
	if srv.healthCheck != nil {
		srv.services = append(srv.services, &healthSvc{srv: srv})
	}

	// validate RPC services are provided
	if len(srv.services) == 0 {
		defer cancel()
		srv.halt()
		return errors.New("no services registered")
	}

	// add middleware
	unaryM, streamM := srv.getMiddleware()
	srv.opts = append(srv.opts, grpc.ChainUnaryInterceptor(unaryM...))
	srv.opts = append(srv.opts, grpc.ChainStreamInterceptor(streamM...))

	// create RPC instance and setup services
	srv.mu.Lock()
	srv.grpc = grpc.NewServer(srv.opts...)
	for _, s := range srv.services {
		s.ServerSetup(srv.grpc)
	}
	srv.mu.Unlock()

	// enable reflection protocol
	if srv.reflection {
		reflection.Register(srv.grpc)
	}

	// initialize server metrics
	if srv.prometheus != nil {
		srv.prometheus.InitializeMetrics(srv.grpc)
	}

	// setup main server network interface
	if srv.nl, err = srv.setupNetworkInterface(srv.net, srv.getAddress()); err != nil {
		defer cancel()
		srv.halt()
		return errors.Wrap(err, "failed to setup main network interface")
	}

	// setup HTTP gateway
	if err = srv.setupGateway(); err != nil {
		defer cancel()
		srv.halt()
		return errors.Wrap(err, "failed to setup HTTP gateway")
	}

	// Start network handlers
	return errors.Wrap(srv.start(ready, 50*time.Millisecond), "failed to start request processing")
}

// Reset will remove any previously set configuration options returning the server
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

	// additional TLS configuration
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

	// get network interface
	nl, err := net.Listen(network, address)
	if err != nil {
		return nil, errors.Errorf("failed to acquire %s network interface on %s", network, address)
	}

	// apply resource limits
	if srv.resourceLimits.Connections > 0 {
		nl = createLimitListener(nl, int(srv.resourceLimits.Connections))
	}

	// apply TLS configuration
	if srv.tlsConfig != nil {
		nl = tls.NewListener(nl, srv.tlsConfig)
	}
	return nl, nil
}

// Prepare the HTTP gateway network interface.
func (srv *Server) setupGatewayInterface() error {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	// use the same main server port by default if no port is provided
	if srv.net != netUNIX && srv.gateway.port == 0 {
		srv.gateway.port = srv.port
	}

	// set up a new network interface if the gateway uses a different TCP port
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

// Configure the server's HTTP gateway network interface and multiplexer.
// nolint: gocyclo
func (srv *Server) setupGateway() (err error) {
	switch {
	// no gateway or options, nothing to do
	case srv.gateway == nil && len(srv.gatewayOpts) == 0:
		return nil
	// no gateway instance but options provided, create a new one
	case srv.gateway == nil && len(srv.gatewayOpts) > 0:
		srv.gateway, err = NewGateway(srv.gatewayOpts...)
		if err != nil {
			return err
		}
	// gateway instance and options provided, setup the gateway
	case srv.gateway != nil && len(srv.gatewayOpts) > 0:
		if err = srv.gateway.setup(srv.gatewayOpts...); err != nil {
			return err
		}
	}

	// setup gateway network interface
	if err = srv.setupGatewayInterface(); err != nil {
		return err
	}

	// establish gateway <-> server connection
	if err = srv.gateway.connect(srv.Endpoint()); err != nil {
		return err
	}

	// prepare gateway mux
	gwMux := gwRuntime.NewServeMux(srv.gateway.options()...)
	var gwMuxH = http.Handler(gwMux) // cast gateway mux as regular HTTP handler

	// register services that support HTTP
	for _, s := range srv.services {
		if hs, ok := s.(HTTPServiceProvider); ok {
			if err := hs.GatewaySetup()(srv.ctx, gwMux, srv.gateway.conn); err != nil {
				return errors.Wrap(err, "HTTP gateway setup error")
			}
		}
	}

	// apply gateway interceptors
	if len(srv.gateway.interceptors) > 0 {
		gwMuxH = srv.gateway.interceptorWrapper(gwMuxH, srv.gateway.interceptors)
	}

	// add custom path handlers
	for _, ch := range srv.gateway.customPaths {
		_ = gwMux.HandlePath(ch.method, ch.path, func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
			ch.hf(w, r)
		})
	}

	// apply gateway middleware
	for _, m := range srv.gateway.middleware {
		gwMuxH = m(gwMuxH)
	}

	// WebSocket support
	if srv.wsProxy != nil {
		gwMuxH = srv.wsProxy.Wrap(gwMuxH)
	}

	// setup gateway server with sane default settings
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

	// all good!
	return nil
}

// Return properly setup server middleware.
func (srv *Server) getMiddleware() (unary []grpc.UnaryServerInterceptor, stream []grpc.StreamServerInterceptor) {
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

	// If enabled, execute protobuf validation using the `protovalidate` package
	if srv.enableValidator {
		unary = append(unary, pvUnaryServerInterceptor())
		stream = append(stream, pvStreamServerInterceptor())
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

// Start server's network handlers.
func (srv *Server) start(ready chan<- bool, timeout time.Duration) error {
	// setup main multiplexer and sub-tasks group
	var tasks errgroup.Group
	srv.cm = cmux.New(srv.nl)

	// start gRPC server
	http2Matcher := cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc")
	grpcL := srv.cm.MatchWithWriters(http2Matcher)
	tasks.Go(func() error {
		return errors.Wrap(srv.grpc.Serve(grpcL), "failed to start grpc server")
	})

	// start HTTP gateway using its own network listener.
	// used if the gateway uses a different TCP port or if the main RPC server is using
	// a UNIX socket as endpoint.
	if srv.gwNl != nil {
		tasks.Go(func() error {
			return errors.Wrap(srv.gw.Serve(srv.gwNl), "failed to start HTTP gateway")
		})
	}

	// start HTTP gateway using the main multiplexer.
	// used when both the gRPC and HTTP server are listening for the requests in the
	// same TCP port.
	if srv.gwNl == nil && srv.gw != nil {
		httpL := srv.cm.Match(cmux.HTTP1Fast())
		tasks.Go(func() error {
			return errors.Wrap(srv.gw.Serve(httpL), "failed to start HTTP gateway with multiplexer")
		})
	}

	// start main multiplexer processing
	tasks.Go(func() error {
		return errors.Wrap(srv.cm.Serve(), "failed to start main multiplexer")
	})

	// setup notification handler
	if ready != nil {
		go func() {
			select {
			case ready <- true:
				// continue after the notification has been received
				return
			case <-time.After(timeout):
				// continue after timeout to prevent blocking the server due to a poorly manage handler
				return
			}
		}()
	}

	// return any error from sub-tasks
	return errors.WithStack(tasks.Wait())
}
