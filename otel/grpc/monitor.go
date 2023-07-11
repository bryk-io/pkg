package otelgrpc

import (
	"google.golang.org/grpc"
)

// Monitor provide easy-to-use instrumentation primitives for gRPC clients
// and servers.
type Monitor interface {
	// Client returns the unary and stream interceptor required to instrument a
	// gRPC client instance.
	Client() (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor)

	// Server returns required gRPC interceptors to instrument a server instance.
	Server() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor)
}

type grpcMonitor struct{}

// NewMonitor returns a ready to use monitor instance that can be used to
// easily instrument gRPC clients and servers.
func NewMonitor() Monitor {
	return new(grpcMonitor)
}

func (e *grpcMonitor) Client() (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	// Build client interceptors
	return unaryClientInterceptor(), streamClientInterceptor()
}

func (e *grpcMonitor) Server() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	// Build server interceptors
	return unaryServerInterceptor(), streamServerInterceptor()
}
