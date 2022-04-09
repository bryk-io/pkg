package extras

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

// PrometheusIntegration allows to easily collect and consume prometheus metrics.
type PrometheusIntegration interface {
	// GatherMetrics try to collect metrics available on a best-effort manner.
	GatherMetrics() ([]*dto.MetricFamily, error)

	// MetricsHandler returns an interface to gather metrics via HTTP.
	MetricsHandler() http.Handler

	// InitializeMetrics initializes all metrics, with their appropriate null value,
	// for all gRPC methods registered on a gRPC server. This is useful, to ensure
	// that all metrics exist when collecting and querying. The server interceptors
	// MUST be registered BEFORE performing this operation.
	InitializeMetrics(srv *grpc.Server)

	// Client returns the unary and stream interceptor required to instrument a
	// gRPC client instance. Captured metrics include histograms by default; this
	// allows calculating service latency but is expensive.
	//   https://github.com/grpc-ecosystem/go-grpc-prometheus#histograms
	Client() (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor)

	// Server returns required gRPC interceptors to instrument a server instance.
	// Captured metrics include histograms by default; this allows calculating service
	// latency but is expensive.
	//   https://github.com/grpc-ecosystem/go-grpc-prometheus#histograms
	//
	// Example Grafana base dashboard:
	//   https://grafana.com/grafana/dashboards/9186
	Server() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor)
}

// Prometheus support capabilities. These are optional and abstracted away
// from the main operator instance.
type prometheusSupport struct {
	registry   *prometheus.Registry   // Main metrics registry
	extras     []prometheus.Collector // User-provided metric collectors
	srvMetrics *gp.ServerMetrics      // Server metrics
	cltMetrics *gp.ClientMetrics      // Client metrics
}

// PrometheusMetrics allows to easily collect and consume instrumentation data.
// Host and runtime metrics are collected by default, in addition to any additional
// collector provided.
func PrometheusMetrics(reg *prometheus.Registry, cols ...prometheus.Collector) (PrometheusIntegration, error) {
	if reg == nil {
		return nil, errors.New("registry is required")
	}
	ps := &prometheusSupport{
		registry: reg,
		extras:   append([]prometheus.Collector{}, cols...),
	}
	if err := ps.init(); err != nil {
		return nil, err
	}
	return ps, nil
}

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
	// descriptor usage as well as the process start time. The collector only works on
	// operating systems with a Linux-style proc filesystem and on Microsoft Windows.
	// On other operating systems, it will not collect any metrics.
	if runtime.GOOS == "linux" || runtime.GOOS == "windows" {
		po := collectors.ProcessCollectorOpts{ReportErrors: true}
		if err := ps.registry.Register(collectors.NewProcessCollector(po)); err != nil {
			return err
		}
	}

	// Register custom collectors
	for _, c := range ps.extras {
		if err := ps.registry.Register(c); err != nil {
			return err
		}
	}

	// All good!
	return nil
}

func (ps *prometheusSupport) GatherMetrics() ([]*dto.MetricFamily, error) {
	return ps.registry.Gather()
}

func (ps *prometheusSupport) MetricsHandler() http.Handler {
	return promhttp.HandlerFor(ps.registry, promhttp.HandlerOpts{
		ErrorLog:            &errorLogger{ll: xlog.Discard()},
		ErrorHandling:       promhttp.ContinueOnError, // Best effort mode
		Registry:            ps.registry,              // Collect 'promhttp_metric_handler_errors_total'
		DisableCompression:  false,                    // Always use compression
		MaxRequestsInFlight: 10,                       // Maximum number of simultaneous requests
		Timeout:             5 * time.Second,          // If exceeded, respond with a 503 ServiceUnavailable
		EnableOpenMetrics:   false,                    // OpenMetrics support
	})
}

func (ps *prometheusSupport) InitializeMetrics(srv *grpc.Server) {
	if ps.srvMetrics == nil {
		return
	}
	ps.srvMetrics.InitializeMetrics(srv)
}

func (ps *prometheusSupport) Client() (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	// Register client metrics
	if ps.cltMetrics == nil {
		ps.cltMetrics = gp.NewClientMetrics()
		ps.cltMetrics.EnableClientHandlingTimeHistogram()
		_ = ps.registry.Register(prometheus.Collector(ps.cltMetrics))
	}

	// Return interceptors
	return ps.cltMetrics.UnaryClientInterceptor(), ps.cltMetrics.StreamClientInterceptor()
}

func (ps *prometheusSupport) Server() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	// Register server metrics
	if ps.srvMetrics == nil {
		ps.srvMetrics = gp.NewServerMetrics()
		ps.srvMetrics.EnableHandlingTimeHistogram()
		_ = ps.registry.Register(prometheus.Collector(ps.srvMetrics))
	}

	// Return interceptors
	return ps.srvMetrics.UnaryServerInterceptor(), ps.srvMetrics.StreamServerInterceptor()
}

// Minimal prometheus error logger implementation.
type errorLogger struct {
	ll xlog.Logger
}

func (el *errorLogger) Println(v ...interface{}) {
	el.ll.Print(xlog.Warning, v...)
}
