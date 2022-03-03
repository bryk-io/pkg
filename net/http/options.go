package http

import (
	"fmt"
	lib "net/http"
	"time"
)

// Option allows to adjust server settings following a functional pattern.
type Option func(srv *Server) error

// WithPort sets the TCP port to handle requests.
func WithPort(port int) Option {
	return func(srv *Server) error {
		srv.nh.Addr = fmt.Sprintf(":%d", port)
		srv.port = port
		return nil
	}
}

// WithIdleTimeout sets the maximum amount of time to wait for the
// next request when "keep-alive" is enabled.
func WithIdleTimeout(timeout time.Duration) Option {
	return func(srv *Server) error {
		srv.nh.IdleTimeout = timeout
		return nil
	}
}

// WithHandler sets the HTTP handler used by the server.
func WithHandler(handler lib.Handler) Option {
	return func(srv *Server) error {
		srv.nh.Handler = handler
		return nil
	}
}

// WithTLS enable secure communications with server using
// HTTPS connections.
func WithTLS(settings TLS) Option {
	return func(srv *Server) error {
		var err error
		srv.tls, err = settings.Expand()
		if err == nil {
			srv.nh.TLSConfig = srv.tls
		}
		return err
	}
}
