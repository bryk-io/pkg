package otel

import (
	"errors"
	"net/http"
	"runtime"
	"time"

	gp "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	xlog "go.bryk.io/pkg/log"
	"google.golang.org/grpc"
)

// Prometheus support capabilities. These are optional and abstracted away
// from the main operator instance for easy removal and will be replaced with
// native OTEL metrics in the future.
type prometheusSupport struct {
	registry   *prometheus.Registry   // Main metrics registry
	extras     []prometheus.Collector // User-provided metric collectors
	grpcServer *gp.ServerMetrics      // gRPC server metric collectors
	enabled    bool
}

// Initialize an empty prometheus support handler.
func newPrometheusHandler() *prometheusSupport {
	ps := &prometheusSupport{
		registry: prometheus.NewRegistry(),
		extras:   []prometheus.Collector{},
		enabled:  true,
	}
	return ps
}

// Internal setup, called only if "WithPrometheusSupport" option is enabled.
func (ps *prometheusSupport) init() error {
	// Include a collector that exports metrics about the current Go process. This
	// includes memory stats. To collect those, runtime.ReadMemStats is called. This
	// requires to “stop the world”, which usually only happens for garbage collection
	// (GC). The performance impact of stopping the world is the more relevant the
	// more frequently metrics are collected.
	if err := ps.registry.Register(collectors.NewGoCollector()); err != nil {
		return err
	}

	// Include the current state of process metrics including CPU, memory and file
	// descriptor usage as well as the process start time. The collector in only
	// available on Linux and Windows.
	if runtime.GOOS == "linux" || runtime.GOOS == "windows" {
		po := collectors.ProcessCollectorOpts{ReportErrors: true}
		_ = ps.registry.Register(collectors.NewProcessCollector(po))
	}

	// Custom collectors
	for _, c := range ps.extras {
		if err := ps.registry.Register(c); err != nil {
			return err
		}
	}

	// All good!
	return nil
}

// PrometheusGatherMetrics try to collect metrics available on the operator instance
// on a best-effort manner. Returns an error if prometheus support is not enabled in
// the operator instance.
func (op *Operator) PrometheusGatherMetrics() ([]*dto.MetricFamily, error) {
	if op.prom == nil {
		return nil, errors.New("prometheus support not enabled")
	}
	return op.prom.registry.Gather()
}

// PrometheusMetricsHandler returns an interface to gather metrics via HTTP. Returns an
// error if prometheus support is not enabled in the operator instance.
func (op *Operator) PrometheusMetricsHandler() (http.Handler, error) {
	if op.prom == nil {
		return nil, errors.New("prometheus support not enabled")
	}
	return promhttp.HandlerFor(op.prom.registry, promhttp.HandlerOpts{
		ErrorLog:            &errorLogger{ll: op.Logger},
		ErrorHandling:       promhttp.ContinueOnError, // Best effort mode
		Registry:            op.prom.registry,         // Collect 'promhttp_metric_handler_errors_total'
		DisableCompression:  false,                    // Always use compression
		MaxRequestsInFlight: 10,                       // Maximum number of simultaneous requests
		Timeout:             5 * time.Second,          // If exceeded, respond with a 503 ServiceUnavailable
		EnableOpenMetrics:   false,                    // OpenMetrics support
	}), nil
}

// PrometheusInitializeServer ensure all metric handlers for a gRPC server exist and
// are initialized to null values. This step must be executed after the server
// middleware is properly set on the server instance.
func (op *Operator) PrometheusInitializeServer(srv *grpc.Server) {
	if op.prom != nil && op.prom.grpcServer != nil {
		op.prom.grpcServer.InitializeMetrics(srv)
	}
}

// Minimal prometheus error logger implementation.
type errorLogger struct {
	ll xlog.Logger
}

func (el *errorLogger) Println(v ...interface{}) {
	el.ll.Print(xlog.Warning, v...)
}
