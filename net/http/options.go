package http

import (
	"fmt"
	lib "net/http"
	"time"
)

// Option allows adjusting server settings following a functional pattern.
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
// next request when "keep-alive" is enabled. You can use `0` to
// disable all the server's timeouts.
func WithIdleTimeout(timeout time.Duration) Option {
	return func(srv *Server) error {
		srv.nh.IdleTimeout = timeout
		srv.nh.WriteTimeout = timeout
		srv.nh.ReadTimeout = timeout
		srv.nh.ReadHeaderTimeout = timeout
		return nil
	}
}

// WithHandler sets the HTTP handler used by the server.
func WithHandler(handler lib.Handler) Option {
	return func(srv *Server) error {
		srv.sh = handler
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

// WithMiddleware register the provided middleware to customize/extend the
// processing of HTTP requests. When applying middleware the ordering is very
// important, in this case it will be applied in the same order provided.
// For example:
//
//	Use(foo bar baz)
//
// Will be applied as:
//
//	baz( bar( foo(handler) ) )
func WithMiddleware(md ...func(lib.Handler) lib.Handler) Option {
	return func(srv *Server) error {
		srv.mu.Lock()
		defer srv.mu.Unlock()
		srv.mw = append(srv.mw, md...)
		return nil
	}
}
