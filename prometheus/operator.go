package prometheus

import (
	"net/http"
	"runtime"
	"time"

	gp "github.com/grpc-ecosystem/go-grpc-prometheus"
	lib "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	"go.bryk.io/pkg/log"
	"google.golang.org/grpc"
)

// Operator instances allows to easily collect and consume prometheus metrics.
type Operator interface {
	// GatherMetrics try to collect metrics available on a best-effort manner.
	GatherMetrics() ([]*dto.MetricFamily, error)

	// MetricsHandler returns an interface to gather metrics via HTTP.
	MetricsHandler() http.Handler

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

	// InitializeMetrics initializes all metrics, with their appropriate null value,
	// for all gRPC methods registered on a gRPC server. This is useful to ensure
	// that all metrics exist when collecting and querying. The server interceptors
	// MUST be registered BEFORE performing this operation.
	InitializeMetrics(srv *grpc.Server)
}

// Prometheus support capabilities. These are optional and abstracted away
// from the main operator instance.
type handler struct {
	registry   *lib.Registry     // Main metrics registry
	extras     []lib.Collector   // User-provided metric collectors
	srvMetrics *gp.ServerMetrics // Server metrics
	cltMetrics *gp.ClientMetrics // Client metrics
}

// NewOperator returns a ready-to-use operator instance. An operator allows to
// easily collect and consume instrumentation data. Host and runtime metrics are
// collected by default, in addition to any additional collector provided. If you
// don't provide a prometheus registry `reg`, a new empty one will be created by
// default.
//
//	prom, _ := NewOperator(prometheus.NewRegistry())
//	opts := []rpc.ServerOption{WithPrometheus(prom)}
func NewOperator(reg *lib.Registry, cols ...lib.Collector) (Operator, error) {
	if reg == nil {
		reg = lib.NewRegistry()
	}
	ps := &handler{
		registry: reg,
		extras:   append([]lib.Collector{}, cols...),
	}
	if err := ps.init(); err != nil {
		return nil, err
	}
	return ps, nil
}

func (ps *handler) init() (err error) {
	// Include a collector that exports metrics about the current Go process. This
	// includes memory stats. To collect those, runtime.ReadMemStats is called. This
	// requires to “stop the world”, which usually only happens for garbage collection
	// (GC). The performance impact of stopping the world is the more relevant the
	// more frequently metrics are collected.
	if err = ps.registry.Register(collectors.NewGoCollector()); err != nil {
		return err
	}

	// Include the current state of process metrics: CPU, memory and file descriptor
	// usage as well as the process start time. The collector only works on OSs with
	// a Linux-style proc filesystem and on Microsoft Windows. On other operating systems,
	// it will not collect any metrics.
	if runtime.GOOS == "linux" || runtime.GOOS == "windows" {
		po := collectors.ProcessCollectorOpts{ReportErrors: true}
		if err = ps.registry.Register(collectors.NewProcessCollector(po)); err != nil {
			return err
		}
	}

	// Register custom collectors
	for _, c := range ps.extras {
		if err = ps.registry.Register(c); err != nil {
			return err
		}
	}

	return
}

func (ps *handler) GatherMetrics() ([]*dto.MetricFamily, error) {
	return ps.registry.Gather()
}

func (ps *handler) MetricsHandler() http.Handler {
	return promhttp.HandlerFor(ps.registry, promhttp.HandlerOpts{
		ErrorLog:            &errorLogger{ll: log.Discard()}, // discard logs; silent mode
		ErrorHandling:       promhttp.ContinueOnError,        // best effort mode; ignore errors
		Registry:            ps.registry,                     // collect 'promhttp_metric_handler_errors_total'
		DisableCompression:  false,                           // always use compression
		MaxRequestsInFlight: 10,                              // maximum number of simultaneous requests
		Timeout:             5 * time.Second,                 // if exceeded, respond with a 503 ServiceUnavailable
		EnableOpenMetrics:   false,                           // disable `OpenMetrics` support
	})
}

func (ps *handler) InitializeMetrics(srv *grpc.Server) {
	if ps.srvMetrics == nil {
		return
	}
	ps.srvMetrics.InitializeMetrics(srv)
}

func (ps *handler) Client() (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	// Register client metrics
	if ps.cltMetrics == nil {
		ps.cltMetrics = gp.NewClientMetrics()
		ps.cltMetrics.EnableClientHandlingTimeHistogram()
		_ = ps.registry.Register(lib.Collector(ps.cltMetrics))
	}

	// Return interceptors
	return ps.cltMetrics.UnaryClientInterceptor(), ps.cltMetrics.StreamClientInterceptor()
}

func (ps *handler) Server() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	// Register server metrics
	if ps.srvMetrics == nil {
		ps.srvMetrics = gp.NewServerMetrics()
		ps.srvMetrics.EnableHandlingTimeHistogram()
		_ = ps.registry.Register(lib.Collector(ps.srvMetrics))
	}

	// Return interceptors
	return ps.srvMetrics.UnaryServerInterceptor(), ps.srvMetrics.StreamServerInterceptor()
}

// Minimal prometheus error logger implementation.
type errorLogger struct {
	ll log.Logger
}

func (el *errorLogger) Println(v ...any) {
	el.ll.Print(log.Warning, v...)
}
