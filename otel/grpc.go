package otel

import (
	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	gp "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

func defaultRPCSettings(op *Operator) []otelgrpc.Option {
	return []otelgrpc.Option{
		otelgrpc.WithTracerProvider(op.traceProvider),
		otelgrpc.WithPropagators(op.propagator),
	}
}

// RPCClient returns the unary and stream interceptor required to instrument a
// gRPC client instance.
func (op *Operator) RPCClient() (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	// Settings
	opts := defaultRPCSettings(op)

	// Client interceptors
	ui := []grpc.UnaryClientInterceptor{otelgrpc.UnaryClientInterceptor(opts...)}
	si := []grpc.StreamClientInterceptor{otelgrpc.StreamClientInterceptor(opts...)}

	// Prometheus metrics
	if op.prom != nil && op.prom.enabled {
		// Register client metrics collector with operator registry. Including
		// histograms allows calculating service latency but is expensive.
		// https://github.com/grpc-ecosystem/go-grpc-prometheus#histograms
		metrics := gp.NewClientMetrics()
		metrics.EnableClientHandlingTimeHistogram()
		_ = op.prom.registry.Register(prometheus.Collector(metrics))

		// Attach interceptors
		ui = append(ui, metrics.UnaryClientInterceptor())
		si = append(si, metrics.StreamClientInterceptor())
	}

	// Return chained interceptors
	return middleware.ChainUnaryClient(ui...), middleware.ChainStreamClient(si...)
}

// RPCServer return required gRPC interceptors to instrument a server instance.
//
// Example Grafana base dashboard:
// https://grafana.com/grafana/dashboards/9186
func (op *Operator) RPCServer() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	// Settings
	opts := defaultRPCSettings(op)

	// Server interceptors
	var ui []grpc.UnaryServerInterceptor
	var si []grpc.StreamServerInterceptor

	// Server unary interceptors
	ui = append(ui, otelgrpc.UnaryServerInterceptor(opts...))

	// Server stream interceptors
	si = append(si, otelgrpc.StreamServerInterceptor(opts...))

	// Prometheus metrics
	if op.prom != nil && op.prom.enabled {
		// gRPC server metrics collector with operator registry. Including
		// histograms allows calculating service latency but is expensive.
		// https://github.com/grpc-ecosystem/go-grpc-prometheus#histograms
		metrics := gp.NewServerMetrics()
		metrics.EnableHandlingTimeHistogram()
		_ = op.prom.registry.Register(prometheus.Collector(metrics))

		// Attach interceptors
		ui = append(ui, metrics.UnaryServerInterceptor())
		si = append(si, metrics.StreamServerInterceptor())
		op.prom.grpcServer = metrics
	}

	// Return chained interceptors
	return middleware.ChainUnaryServer(ui...), middleware.ChainStreamServer(si...)
}
