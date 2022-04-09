package drpc

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"

	srvmw "go.bryk.io/pkg/net/drpc/middleware/server"
	"go.bryk.io/pkg/net/drpc/ws"
	"golang.org/x/sync/errgroup"
	"storj.io/drpc"
	"storj.io/drpc/drpchttp"
	"storj.io/drpc/drpcmigrate"
	"storj.io/drpc/drpcmux"
	"storj.io/drpc/drpcserver"
)

// Server instances are intended to expose DRPC services for network-based
// consumption.
type Server struct {
	ntp  string                 // network protocol used
	tls  *tls.Config            // TLS settings
	net  net.Listener           // main network interface
	mux  *drpcmux.Mux           // main routes mux
	dlm  *drpcmigrate.ListenMux // HTTP-support mux
	dsv  *drpcserver.Server     // DRPC server
	hsv  http.Server            // HTTP server
	sps  []ServiceProvider      // registered services
	bgt  sync.WaitGroup         // background tasks
	mdw  []srvmw.Middleware     // middleware set
	wsp  *ws.Proxy              // web-socket proxy
	mtx  sync.RWMutex           // concurrency access lock
	ctx  context.Context        // main context
	halt context.CancelFunc     // halt notification trigger
	addr string                 // user-provided network address
	http bool                   // HTTP support-enabled flag
}

// ServiceProvider elements define the services that are to be exposed through
// a server instance. Some points to note:
//  - A single server can expose several services
//  - The user is responsible to ensure services are free of collisions
//  - The service provider MUST also itself provide the implementation for the
//    service defined by the "DRPCDescription" method. If this is not the case
//    you can use the server's "RegisterService" method to manually specify
//    the implementation element and service description independently
type ServiceProvider interface {
	// DRPCDescription must return the service description as generated by the
	// "protoc-gen-go-drpc" compiler plugin.
	//   More information: https://storj.github.io/drpc/docs.html
	DRPCDescription() drpc.Description
}

// NewServer returns a ready-to-use server instance.
func NewServer(options ...Option) (*Server, error) {
	srv := &Server{
		sps:  []ServiceProvider{},  // no default services
		mdw:  []srvmw.Middleware{}, // no default middleware
		bgt:  sync.WaitGroup{},     // background tasks
		ntp:  "tcp",                // use TCP
		addr: "127.0.0.1:0",        // select random (local-only) port
	}
	srv.ctx, srv.halt = context.WithCancel(context.Background())
	var err error
	if err = srv.setup(options...); err != nil {
		return nil, err
	}
	if err = srv.networkInterface(); err != nil {
		return nil, err
	}
	srv.mux = drpcmux.New()
	for _, sp := range srv.sps {
		if err = srv.mux.Register(sp, sp.DRPCDescription()); err != nil {
			return nil, err
		}
	}
	return srv, nil
}

// Start the server and wait for incoming requests.
func (srv *Server) Start() error {
	// background tasks
	var tasks errgroup.Group

	// Apply middleware to server handler
	var srvHandler drpc.Handler = srv.mux
	for _, mw := range srv.mdw {
		srvHandler = mw(srvHandler)
	}

	// DRPC server
	srv.dsv = drpcserver.New(srvHandler)
	tasks.Go(func() error {
		return srv.dsv.Serve(srv.ctx, srv.net)
	})

	if srv.http {
		// HTTP handler
		httpHandler := drpchttp.New(srvHandler)

		// Enable web-socket proxy
		if srv.wsp != nil {
			httpHandler = srv.wsp.Wrap(srvHandler, httpHandler)
		}

		// HTTP server
		srv.hsv = http.Server{Handler: httpHandler}
		tasks.Go(func() error {
			return srv.hsv.Serve(srv.dlm.Default())
		})

		// Multiplexer
		tasks.Go(func() error {
			return srv.dlm.Run(srv.ctx)
		})
	}

	return tasks.Wait()
}

// Stop the server's network interfaces. Any blocked operations will be
// unblocked and return errors.
func (srv *Server) Stop() error {
	if srv.http {
		// gracefully stop HTTP server
		_ = srv.hsv.Shutdown(srv.ctx)
	}
	srv.halt()             // trigger halt signal
	srv.bgt.Wait()         // wait for background processes to complete
	return srv.net.Close() // close network interface
}

// RegisterService associates the RPCs described by `desc` to the provided `impl`
// element and exposes them through the server instance.
func (srv *Server) RegisterService(impl interface{}, desc drpc.Description) error {
	return srv.mux.Register(impl, desc)
}

// Use will register middleware elements to be applied to the server instance.
// Middleware is executed before the processing of RPC requests is started.
// When providing middleware the ordering is very important; middleware will be
// applied in the same order provided.
//   For example:
//     Use(foo bar baz)
//   Will be applied as:
//     baz( bar( foo(handler) ) )
func (srv *Server) Use(mw ...srvmw.Middleware) {
	srv.mtx.Lock()
	for _, m := range mw {
		srv.mdw = append([]srvmw.Middleware{m}, srv.mdw...)
	}
	srv.mtx.Unlock()
}

// Apply user provided configuration options.
func (srv *Server) setup(opts ...Option) (err error) {
	for _, opt := range opts {
		if err = opt(srv); err != nil {
			return
		}
	}
	return
}

// Setup server's main network interface.
func (srv *Server) networkInterface() (err error) {
	srv.net, err = net.Listen(srv.ntp, srv.addr)
	if err != nil {
		return
	}
	if srv.tls != nil {
		srv.net = tls.NewListener(srv.net, srv.tls)
	}
	if srv.http {
		srv.dlm = drpcmigrate.NewListenMux(srv.net, len(drpcmigrate.DRPCHeader))
		srv.net = srv.dlm.Route(drpcmigrate.DRPCHeader)
	}
	return
}
