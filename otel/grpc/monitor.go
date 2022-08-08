package otelgrpc

import (
	mw "github.com/grpc-ecosystem/go-grpc-middleware"
	apiErrors "go.bryk.io/pkg/otel/errors"
	sentrygrpc "go.bryk.io/pkg/otel/sentry/grpc"
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

type grpcMonitor struct {
	rep apiErrors.Reporter
}

// NewMonitor returns a ready to use monitor instance that can be used to
// easily instrument gRPC clients and servers.
func NewMonitor(rep apiErrors.Reporter) Monitor {
	return &grpcMonitor{rep: rep}
}

func (e *grpcMonitor) Client() (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	// Build client interceptors
	sui, ssi := sentrygrpc.Client(e.rep)
	ui := mw.ChainUnaryClient(unaryClientInterceptor(), sui)
	si := mw.ChainStreamClient(streamClientInterceptor(), ssi)
	return ui, si
}

func (e *grpcMonitor) Server() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	// Build server interceptors
	sui, ssi := sentrygrpc.Server(e.rep)
	ui := mw.ChainUnaryServer(unaryServerInterceptor(), sui)
	si := mw.ChainStreamServer(streamServerInterceptor(), ssi)
	return ui, si
}
