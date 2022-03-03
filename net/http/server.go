package http

import (
	"context"
	"crypto/tls"
	lib "net/http"
)

// Server provides the main HTTP(S) service provider.
type Server struct {
	nh   *lib.Server
	tls  *tls.Config
	port int
}

// NewServer returns a new read-to-use server instance adjusted with the
// provided configuration options.
func NewServer(options ...Option) (*Server, error) {
	srv := &Server{
		nh: &lib.Server{},
	}
	if err := srv.setup(options...); err != nil {
		return nil, err
	}
	return srv, nil
}

// Start the server instance and start receiving and handling requests.
func (srv *Server) Start() error {
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

func (srv *Server) setup(options ...Option) error {
	for _, opt := range options {
		if err := opt(srv); err != nil {
			return err
		}
	}
	return nil
}
