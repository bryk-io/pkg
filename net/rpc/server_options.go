package rpc

import (
	"context"
	"syscall"

	"go.bryk.io/pkg/errors"
	"go.bryk.io/pkg/net/rpc/ws"
	"go.bryk.io/pkg/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NetworkInterfaceLocal defines the local loopback interface (i.e., localhost / 127.0.0.1).
const NetworkInterfaceLocal = "local"

// NetworkInterfaceAll will set up a network listener on all available `unicast` and
// `anycast` IP addresses of the local system.
const NetworkInterfaceAll = "all"

// ServerOption allows adjusting server settings following a functional pattern.
type ServerOption func(*Server) error

// TokenValidator represents an external authentication mechanism used to validate
// bearer credentials. In case of success return codes.OK; for any error return
// a proper status code (like codes.Unauthenticated or codes.PermissionDenied) and,
// optionally, a custom message.
type TokenValidator func(token string) (codes.Code, string)

// WithServiceProvider adds an RPC service handler to the server instance, at least one
// service provider is required when starting the server.
func WithServiceProvider(sp ServiceProvider) ServerOption {
	return func(srv *Server) error {
		srv.mu.Lock()
		defer srv.mu.Unlock()
		srv.services = append(srv.services, sp)
		return nil
	}
}

// WithPanicRecovery allows the server to convert panic events into a gRPC error with
// status 'Internal'.
func WithPanicRecovery() ServerOption {
	return func(srv *Server) error {
		srv.mu.Lock()
		srv.panicRecovery = true
		srv.mu.Unlock()
		return nil
	}
}

// WithNetworkInterface specifies which network interface to use to listen for incoming
// requests.
func WithNetworkInterface(name string) ServerOption {
	return func(srv *Server) (err error) {
		srv.mu.Lock()
		defer srv.mu.Unlock()
		srv.address, err = GetAddress(name)
		if err != nil {
			return errors.WithStack(err)
		}
		srv.net = netTCP
		srv.netInterface = name
		return
	}
}

// WithPort specifies which TCP port the server use.
func WithPort(port int) ServerOption {
	return func(srv *Server) error {
		srv.mu.Lock()
		defer srv.mu.Unlock()
		srv.net = netTCP
		srv.port = port
		return nil
	}
}

// WithUnixSocket specifies the server should use a UNIX socket as main access point.
func WithUnixSocket(socket string) ServerOption {
	return func(srv *Server) error {
		srv.mu.Lock()
		defer srv.mu.Unlock()
		if err := syscall.Unlink(socket); err != nil {
			return errors.Wrap(err, "failed to unlink socket")
		}
		srv.net = netUNIX
		srv.address = socket
		return nil
	}
}

// WithResourceLimits applies constraints to the resources the server instance can consume.
func WithResourceLimits(limits ResourceLimits) ServerOption {
	return func(srv *Server) error {
		srv.mu.Lock()
		defer srv.mu.Unlock()
		srv.resourceLimits = limits
		if limits.Requests > 0 {
			srv.opts = append(srv.opts, grpc.MaxConcurrentStreams(limits.Requests))
		}
		if limits.Rate > 0 {
			srv.opts = append(srv.opts, grpc.InTapHandle(newRateTap(limits.Rate).handler))
		}
		return nil
	}
}

// WithInputValidation will automatically detect any errors on received messages by
// detecting if a `Validate` method is available and returning any produced errors
// with an `InvalidArgument` status code.
//
// To further automate input validation use:
//
//	https://github.com/envoyproxy/protoc-gen-validate
func WithInputValidation() ServerOption {
	return func(srv *Server) error {
		srv.mu.Lock()
		srv.inputValidation = true
		srv.mu.Unlock()
		return nil
	}
}

// WithProtoValidate enables automatic input validation using the `protovalidate`
// package. Any validation errors will be returned with status code `InvalidArgument`.
// https://github.com/bufbuild/protovalidate
func WithProtoValidate() ServerOption {
	return func(srv *Server) (err error) {
		srv.mu.Lock()
		srv.enableValidator = true
		srv.mu.Unlock()
		return nil
	}
}

// WithTLS enables the server to use secure communication channels with the provided
// credentials and settings. If a certificate is provided the server name MUST match
// the identifier included in the certificate.
func WithTLS(opts ServerTLSConfig) ServerOption {
	return func(srv *Server) (err error) {
		srv.mu.Lock()
		srv.tlsOptions = opts
		srv.tlsConfig, err = serverTLSConf(srv.tlsOptions)
		srv.mu.Unlock()
		return errors.WithStack(err)
	}
}

// WithAuthByCertificate enables certificate-based authentication on the server. It
// can be used multiple times to allow for several certificate authorities. This option
// is only applicable when operating the server through a TLS channel, otherwise will
// simply be ignored.
func WithAuthByCertificate(clientCA []byte) ServerOption {
	return func(srv *Server) error {
		srv.mu.Lock()
		srv.clientCAs = append(srv.clientCAs, clientCA)
		srv.mu.Unlock()
		return nil
	}
}

// WithAuthByToken allows to use an external authentication mechanism for the server
// using bearer tokens as credentials. Setting this option will enable automatic
// authentication for all methods enabled on the server. When a server requires to
// support both authenticated and unauthenticated methods, the verification process
// can be performed manually per-method.
//
//	token, err := GetAuthToken(ctx, "bearer")
//	... validate token ...
func WithAuthByToken(tv TokenValidator) ServerOption {
	return func(srv *Server) error {
		srv.mu.Lock()
		defer srv.mu.Unlock()

		// Prepare authentication function
		srv.tokenValidator = func(ctx context.Context) (context.Context, error) {
			token, err := GetAuthToken(ctx, "bearer")
			if err != nil {
				return nil, err
			}
			if code, msg := tv(token); code != codes.OK {
				return nil, status.Errorf(code, "invalid auth token: %s", msg)
			}
			return ctx, nil
		}
		return nil
	}
}

// WithUnaryMiddleware allows including custom middleware functions when processing
// incoming unary RPC requests. Order is important when chaining multiple middleware.
func WithUnaryMiddleware(entry ...grpc.UnaryServerInterceptor) ServerOption {
	return func(srv *Server) error {
		srv.mu.Lock()
		srv.customUnary = append(srv.customUnary, entry...)
		srv.mu.Unlock()
		return nil
	}
}

// WithStreamMiddleware allows including custom middleware functions when processing
// stream RPC operations. Order is important when chaining multiple middleware.
func WithStreamMiddleware(entry ...grpc.StreamServerInterceptor) ServerOption {
	return func(srv *Server) error {
		srv.mu.Lock()
		srv.customStream = append(srv.customStream, entry...)
		srv.mu.Unlock()
		return nil
	}
}

// WithHTTPGateway registers a gateway interface to allow for HTTP access to
// the server.
func WithHTTPGateway(gw *Gateway) ServerOption {
	return func(srv *Server) error {
		srv.mu.Lock()
		srv.gateway = gw
		srv.mu.Unlock()
		return nil
	}
}

// WithHTTPGatewayOptions adjust the behavior of the HTTP gateway.
func WithHTTPGatewayOptions(opts ...GatewayOption) ServerOption {
	return func(srv *Server) error {
		srv.mu.Lock()
		srv.gatewayOpts = append(srv.gatewayOpts, opts...)
		srv.mu.Unlock()
		return nil
	}
}

// WithWebSocketProxy configure the server to support bidirectional streaming over
// HTTP utilizing web sockets.
func WithWebSocketProxy(opts ...ws.ProxyOption) ServerOption {
	return func(srv *Server) error {
		proxy, err := ws.New(opts...)
		if err != nil {
			return err
		}
		srv.mu.Lock()
		srv.wsProxy = proxy
		srv.mu.Unlock()
		return nil
	}
}

// WithPrometheus allows generating and consuming metrics from the server
// instance using the Prometheus standards and tooling.
func WithPrometheus(prometheus prometheus.Operator) ServerOption {
	return func(srv *Server) error {
		srv.mu.Lock()
		defer srv.mu.Unlock()
		srv.prometheus = prometheus
		return nil
	}
}

// WithReflection enables the server to provide information about publicly-accessible
// services and assists clients at runtime to construct RPC requests and responses
// without precompiled service information. It is used by gRPC CLI, which can be
// used to introspect server protos and send/receive test RPCs.
//
// More information about the reflection protocol:
//
//	https://github.com/grpc/grpc/blob/master/doc/server-reflection.md
//
// More information about the gRPC CLI tool:
//
//	https://github.com/grpc/grpc/blob/master/doc/command_line_tool.md
func WithReflection() ServerOption {
	return func(srv *Server) error {
		srv.mu.Lock()
		defer srv.mu.Unlock()
		srv.reflection = true
		return nil
	}
}

// WithHealthCheck enables the server to provide health check information
// to clients. If an error is returned by the provided health check function
// the service will be marked as unavailable and respond with a status code
// of `NOT_SERVING`.
//
// More information about the health check protocol:
//
//	https://github.com/grpc/grpc/blob/master/doc/health-checking.md
func WithHealthCheck(check HealthCheck) ServerOption {
	return func(srv *Server) error {
		srv.mu.Lock()
		defer srv.mu.Unlock()
		srv.healthCheck = check
		return nil
	}
}
