package otelgrpc

import (
	lib "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/filters"
	"google.golang.org/grpc"
)

// ClientInstrumentation can be used to easily instrument any gRPC client connection.
func ClientInstrumentation() grpc.DialOption {
	opts := []lib.Option{
		lib.WithFilter(filters.Not(filters.HealthCheck())),
		lib.WithMessageEvents(lib.ReceivedEvents, lib.SentEvents),
	}
	sh := lib.NewClientHandler(opts...)
	return grpc.WithStatsHandler(sh)
}

// ServerInstrumentation can be used to easily instrument any gRPC server instance.
func ServerInstrumentation() grpc.ServerOption {
	opts := []lib.Option{
		lib.WithFilter(filters.Not(filters.HealthCheck())),
		lib.WithMessageEvents(lib.ReceivedEvents, lib.SentEvents),
	}
	sh := lib.NewServerHandler(opts...)
	return grpc.StatsHandler(sh)
}
