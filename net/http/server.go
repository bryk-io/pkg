package http

import (
	"context"
	"crypto/tls"
	lib "net/http"
	"sync"
	"time"
)

// Server provides the main HTTP(S) service provider.
type Server struct {
	nh   *lib.Server
	sh   lib.Handler
	mw   []func(lib.Handler) lib.Handler
	mu   sync.Mutex
	tls  *tls.Config
	port int
}

// NewServer returns a new read-to-use server instance adjusted with the
// provided configuration options.
func NewServer(options ...Option) (*Server, error) {
	srv := &Server{
		nh: &lib.Server{
			MaxHeaderBytes:    1024,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			IdleTimeout:       10 * time.Second,
			WriteTimeout:      10 * time.Second,
		},
		mw: []func(lib.Handler) lib.Handler{},
	}

	// Apply user settings
	for _, opt := range options {
		if err := opt(srv); err != nil {
			return nil, err
		}
	}

	// Apply middleware
	for _, mw := range srv.mw {
		srv.sh = mw(srv.sh)
	}
	return srv, nil
}

// Start the server instance and start receiving and handling requests.
func (srv *Server) Start() error {
	srv.nh.Handler = srv.sh
	if srv.tls != nil {
		return srv.nh.ListenAndServeTLS("", "")
	}
	return srv.nh.ListenAndServe()
}

// Stop the server instance. If graceful is set, the server closes without
// interrupting any active connections by first closing all open listeners,
// then closing all idle connections, and then waiting indefinitely for
// connections to return to idle.
func (srv *Server) Stop(graceful bool) error {
	if !graceful {
		return srv.nh.Close()
	}
	return srv.nh.Shutdown(context.Background())
}
