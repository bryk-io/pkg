package sdk

import (
	"time"

	"go.bryk.io/pkg/log"
	"go.bryk.io/pkg/otel"
	"go.opentelemetry.io/otel/propagation"
	sdkMetric "go.opentelemetry.io/otel/sdk/metric"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
)

// Option provide a functional style configuration mechanism
// to instrument an application.
type Option func(*Instrumentation)

// WithServiceName adjust the `service.name` attribute.
func WithServiceName(name string) Option {
	return func(op *Instrumentation) {
		op.attrs.Set(lblSvcName, name)
	}
}

// WithServiceVersion adjust the `service.version` attribute.
func WithServiceVersion(version string) Option {
	return func(op *Instrumentation) {
		op.attrs.Set(lblSvcVer, version)
	}
}

// WithSpanLimits allows to adjust the limits bound any Span created by
// the tracer.
// https://pkg.go.dev/go.opentelemetry.io/otel/sdk/trace#SpanLimits
func WithSpanLimits(sl sdkTrace.SpanLimits) Option {
	return func(op *Instrumentation) {
		op.spanLimits = sl
	}
}

// WithPropagator add a new propagator to the application. OpenTelemetry
// propagators are used to extract and inject context data from and into
// messages exchanged by applications. The application supports the following
// propagation mechanisms by default:
//   - W3C Trace Context (https://www.w3.org/TR/trace-context/)
//   - W3C Baggage (https://www.w3.org/TR/baggage/)
func WithPropagator(mp propagation.TextMapPropagator) Option {
	return func(op *Instrumentation) {
		op.props = append(op.props, mp)
	}
}

// WithSpanProcessor registers a new span processor in the trace provider
// processing chain.
func WithSpanProcessor(sp sdkTrace.SpanProcessor) Option {
	return func(op *Instrumentation) {
		op.spanProcessors = append(op.spanProcessors, sp)
	}
}

// WithResourceAttributes allows extending (or overriding) the core
// attributes used globally by the application. The core attributes must
// provide information at the resource level. These attributes are used
// to configure the application's tracer and logger instances.
func WithResourceAttributes(fields otel.Attributes) Option {
	return func(op *Instrumentation) {
		op.attrs = join(op.attrs, fields)
	}
}

// WithBaseLogger set the output handler. If not provided, all output is
// discarded by default. The application will create an extended logger
// using all the attributes discovered/provided during the setup process.
func WithBaseLogger(ll log.Logger) Option {
	return func(op *Instrumentation) {
		op.log = ll
	}
}

// WithExporter enables a trace (i.e. span) exporter as data sink for the
// application. If no exporter is set, all traces are discarded by default.
func WithExporter(exp sdkTrace.SpanExporter) Option {
	return func(op *Instrumentation) {
		op.traceExporter = exp
	}
}

// WithSampler adjust the sampling strategy used by the application.
// All traces are sampled by default.
//
// https://opentelemetry.io/docs/instrumentation/go/exporting_data/#sampling
func WithSampler(ss sdkTrace.Sampler) Option {
	return func(op *Instrumentation) {
		op.sampler = ss
	}
}

// WithMetricReader configures the application's meter provider to export
// the measured data. Readers take two forms: ones that push to an endpoint
// (NewPeriodicReader), and ones that an endpoint pulls from. See the
// `go.opentelemetry.io/otel/exporters` package for exporters that can be
// used as or with these Readers.
func WithMetricReader(exp sdkMetric.Reader) Option {
	return func(op *Instrumentation) {
		op.metricExporter = exp
	}
}

// WithHostMetrics enables the application to capture the conventional host
// metric instruments specified by OpenTelemetry. Host metric events are
// sometimes collected through the OpenTelemetry Collector `host metrics`
// receiver running as an agent; this instrumentation option provides an
// alternative for processes that want to record the same information without
// an agent.
func WithHostMetrics() Option {
	return func(op *Instrumentation) {
		op.hostMetrics = true
	}
}

// WithRuntimeMetrics enables the application to capture the conventional runtime
// metrics specified by OpenTelemetry. The provided interval value sets the
// minimum interval between calls to runtime.ReadMemStats(), which is a relatively
// expensive call to make frequently. The default interval value is 10 seconds,
// passing a value <= 0 uses the default.
func WithRuntimeMetrics(interval time.Duration) Option {
	return func(op *Instrumentation) {
		if interval.Seconds() <= 0 {
			interval = 10 * time.Second
		}
		op.runtimeMetrics = true
		op.runtimeMetricsInt = interval
	}
}

// WithExemplars enable experimental support for exemplars by settings
// the adequate ENV variables
//
// https://github.com/open-telemetry/opentelemetry-go/blob/main/sdk/metric/internal/x/README.md
func WithExemplars() Option {
	return func(op *Instrumentation) {
		op.exemplars = true
	}
}
