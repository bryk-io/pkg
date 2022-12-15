package otel

import (
	"errors"
	"time"

	"go.bryk.io/pkg/log"
	apiErrors "go.bryk.io/pkg/otel/errors"
	"go.opentelemetry.io/otel/propagation"
	sdkMetric "go.opentelemetry.io/otel/sdk/metric"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
)

// OperatorOption provide a functional style configuration mechanism
// for observability operator instances.
type OperatorOption func(*Operator) error

// WithServiceName adjust the `service.name` attribute.
func WithServiceName(name string) OperatorOption {
	return func(op *Operator) error {
		op.coreAttributes.Set(lblSvcName, name)
		return nil
	}
}

// WithServiceVersion adjust the `service.version` attribute.
func WithServiceVersion(version string) OperatorOption {
	return func(op *Operator) error {
		op.coreAttributes.Set(lblSvcVer, version)
		return nil
	}
}

// WithTracerName adjust the `otel.library.name` attribute.
func WithTracerName(name string) OperatorOption {
	return func(op *Operator) error {
		if name != "" {
			op.tracerName = name
		}
		return nil
	}
}

// WithSpanLimits allows to adjust the limits bound any Span created by the tracer.
// https://pkg.go.dev/go.opentelemetry.io/otel/sdk/trace#SpanLimits
func WithSpanLimits(sl sdkTrace.SpanLimits) OperatorOption {
	return func(op *Operator) error {
		op.spanLimits = sl
		return nil
	}
}

// WithPropagator add a new propagator to the operator instance. OpenTelemetry propagators are
// used to extract and inject context data from and into messages exchanged by applications.
// The operator supports by default the W3C Trace Context (https://www.w3.org/TR/trace-context/),
// and W3C Baggage (https://www.w3.org/TR/baggage/) propagation mechanisms.
func WithPropagator(mp propagation.TextMapPropagator) OperatorOption {
	return func(op *Operator) error {
		op.props = append(op.props, mp)
		return nil
	}
}

// WithResourceAttributes allows extending (or overriding) the core attributes used
// globally by the operator. The core attributes must provide information at the resource
// level. These attributes are used to configure the operator's tracer and logger instances;
// are inherited by all spans created and included in logged messages.
func WithResourceAttributes(fields Attributes) OperatorOption {
	return func(op *Operator) error {
		op.userAttributes = join(op.userAttributes, fields)
		return nil
	}
}

// WithLogger set the output handler. If not provided, all output is discarded by default.
func WithLogger(ll log.Logger) OperatorOption {
	return func(op *Operator) error {
		op.log = ll
		return nil
	}
}

// WithExporter enables a trace (i.e. span) exporter as data sink for the operator.
// If no exporter is set, all traces are discarded by default.
func WithExporter(exp sdkTrace.SpanExporter) OperatorOption {
	return func(op *Operator) error {
		op.traceExporter = exp
		return nil
	}
}

// WithSampler adjust the sampling strategy used by the operator instance. All traces are
// sampled by default.
//
// https://opentelemetry.io/docs/instrumentation/go/exporting_data/#sampling
func WithSampler(ss sdkTrace.Sampler) OperatorOption {
	return func(op *Operator) error {
		op.sampler = ss
		return nil
	}
}

// WithErrorReporter enables the operator to capture exceptions data and submit it
// to an external service. If not provided, all output is discarded by default.
func WithErrorReporter(rep apiErrors.Reporter) OperatorOption {
	return func(op *Operator) error {
		op.reporter = rep
		return nil
	}
}

// WithMetricReader configures the operator's meter provider to export the measured
// data. Readers take two forms: ones that push to an endpoint (NewPeriodicReader),
// and ones that an endpoint pulls from. See the `go.opentelemetry.io/otel/exporters`
// package for exporters that can be used as or with these Readers.
func WithMetricReader(exp sdkMetric.Reader) OperatorOption {
	return func(op *Operator) error {
		op.metricExporter = exp
		return nil
	}
}

// WithHostMetrics enables the operator to capture the conventional host metric instruments
// specified by OpenTelemetry. Host metric events are sometimes collected through the
// OpenTelemetry Collector `host metrics` receiver running as an agent; this instrumentation
// is an alternative for processes that want to record the same information without an agent.
func WithHostMetrics() OperatorOption {
	return func(op *Operator) error {
		op.hostMetrics = true
		return nil
	}
}

// WithRuntimeMetrics enables the operator to capture the conventional runtime
// metrics specified by OpenTelemetry. The provided interval value sets the
// minimum interval between calls to runtime.ReadMemStats(), which is a relatively
// expensive call to make frequently. The default interval value is 10 seconds, passing
// a value of 0 uses the default.
func WithRuntimeMetrics(interval time.Duration) OperatorOption {
	return func(op *Operator) error {
		if interval.Seconds() < 0 {
			return errors.New("negative runtime memory capture period")
		}
		op.runtimeMetrics = true
		if interval != 0 {
			op.runtimeMetricsInt = interval
		}
		return nil
	}
}
