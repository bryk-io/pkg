package otelgrpc

import (
	mw "github.com/grpc-ecosystem/go-grpc-middleware"
	apiErrors "go.bryk.io/pkg/otel/errors"
	sentrygrpc "go.bryk.io/pkg/otel/sentry/grpc"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
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

func (e *grpcMonitor) settings() []otelgrpc.Option {
	// Propagator, metric provider and trace provider are taking from globals
	// setup during the otel.Operator initialization.
	return []otelgrpc.Option{}
}

func (e *grpcMonitor) Client() (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	// Settings
	opts := e.settings()

	// Build client interceptors
	sui, ssi := sentrygrpc.Client(e.rep)
	ui := mw.ChainUnaryClient(otelgrpc.UnaryClientInterceptor(opts...), sui)
	si := mw.ChainStreamClient(otelgrpc.StreamClientInterceptor(opts...), ssi)
	return ui, si
}

func (e *grpcMonitor) Server() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	// Settings
	opts := e.settings()

	// Build server interceptors
	sui, ssi := sentrygrpc.Server(e.rep)
	ui := mw.ChainUnaryServer(otelgrpc.UnaryServerInterceptor(opts...), sui)
	si := mw.ChainStreamServer(otelgrpc.StreamServerInterceptor(opts...), ssi)
	return ui, si
}
