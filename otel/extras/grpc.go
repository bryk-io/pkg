package extras

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

// GRPCMonitor provide easy-to-use instrumentation primitives for gRPC clients
// and servers.
type GRPCMonitor interface {
	// Client returns the unary and stream interceptor required to instrument a
	// gRPC client instance.
	Client() (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor)

	// Server returns required gRPC interceptors to instrument a server instance.
	Server() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor)
}

type grpcMonitor struct{}

// NewGRPCMonitor returns a ready to use monitor instance that can be used to
// easily instrument gRPC clients and servers.
func NewGRPCMonitor() GRPCMonitor {
	return &grpcMonitor{}
}

func (e *grpcMonitor) settings() []otelgrpc.Option {
	// Propagator, metric provider and trace provider are taking from globals
	// setup during the otel.Operator initialization.
	return []otelgrpc.Option{}
}

func (e *grpcMonitor) Client() (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	// Settings
	opts := e.settings()

	// Client interceptors
	return otelgrpc.UnaryClientInterceptor(opts...), otelgrpc.StreamClientInterceptor(opts...)
}

func (e *grpcMonitor) Server() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	// Settings
	opts := e.settings()

	// Server interceptors
	return otelgrpc.UnaryServerInterceptor(opts...), otelgrpc.StreamServerInterceptor(opts...)
}
