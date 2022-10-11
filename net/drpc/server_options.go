package drpc

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	srvMW "go.bryk.io/pkg/net/drpc/middleware/server"
	"go.bryk.io/pkg/net/drpc/ws"
)

// Option allows adjusting server settings following a functional pattern.
type Option func(srv *Server) error

// WithServiceProvider can be used to expose RPC services described and implemented
// by "sp" through a server instance.
func WithServiceProvider(sp ServiceProvider) Option {
	return func(srv *Server) error {
		srv.sps = append(srv.sps, sp)
		return nil
	}
}

// WithPort specifies which TCP port the server use.
func WithPort(port uint) Option {
	return func(srv *Server) error {
		srv.ntp = "tcp"
		srv.addr = fmt.Sprintf(":%d", port)
		return nil
	}
}

// WithUnixSocket specifies the path to a UNIX socket to use as main access
// point. If the provided socket file doesn't exist it will be created by default.
func WithUnixSocket(socket string) Option {
	return func(srv *Server) error {
		sf := filepath.Clean(socket)
		if !exists(sf) {
			f, err := os.OpenFile(sf, os.O_CREATE|os.O_RDONLY, 0600)
			if err != nil {
				return err
			}
			_ = f.Close()
		}
		if err := syscall.Unlink(sf); err != nil {
			return err
		}
		srv.ntp = "unix"
		srv.addr = sf
		return nil
	}
}

// WithTLS enables the server to use secure communication channels with the provided
// credentials and settings. If a certificate is provided the server name MUST match
// the identifier included in the certificate.
func WithTLS(opts ServerTLS) Option {
	return func(srv *Server) error {
		tc, err := opts.conf()
		if err != nil {
			return err
		}
		srv.tls = tc
		return nil
	}
}

// WithAuthByCertificate enables certificate-based authentication on the server.
// It can be used multiple times to allow for several certificate authorities.
// This requires the client and the server to use a TLS communication channel,
// otherwise this option will be ignored.
func WithAuthByCertificate(clientCA []byte) Option {
	return func(srv *Server) error {
		srv.clientCAs = append(srv.clientCAs, clientCA)
		return nil
	}
}

// WithMiddleware register the provided middleware to customize/extend the
// processing of RPC requests. When applying middleware the ordering is very
// important, in this case it will be applied in the same order provided.
// For example:
//
//	Use(foo bar baz)
//
// Will be applied as:
//
//	baz( bar( foo(handler) ) )
func WithMiddleware(mw ...srvMW.Middleware) Option {
	return func(srv *Server) error {
		srv.Use(mw...)
		return nil
	}
}

// WithHTTP enable access to the services exposed by the server via HTTP / JSON.
// When using HTTP support on the server, clients MUST properly include the
// `drpcmigrate.DRPCHeader` header for selecting the protocol to use. To set the
// header automatically use the `WithProtocolHeader` client option when creating a
// new connection.
//
// When exposing services via HTTP the default routes are set as:
//
//	POST: {server_url}/{proto_package}.{service}/{method}
func WithHTTP() Option {
	return func(srv *Server) error {
		srv.http = true
		return nil
	}
}

// WithWebSocketProxy enable bidirectional streaming on the DRPC server via
// websocket connections.
func WithWebSocketProxy(opts ...ws.ProxyOption) Option {
	return func(srv *Server) error {
		wsp, err := ws.New(opts...)
		if err != nil {
			return err
		}
		srv.wsp = wsp
		return nil
	}
}
