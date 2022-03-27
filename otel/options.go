package otel

import (
	"errors"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	xlog "go.bryk.io/pkg/log"
	sdkMetric "go.opentelemetry.io/otel/sdk/metric/export"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
)

// OperatorOption provide a functional style configuration mechanism
// for observability operator instances.
type OperatorOption func(*Operator) error

// WithServiceName adjust the `service.name` attribute. If no service name is
// provided, the default value "service" will be used.
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

// WithResourceAttributes allows extending (or overriding) the core attributes used
// globally by the operator. The core attributes must provide information
// at the resource level. These attributes are used to configure the
// operator's tracer and logger instances; are inherited by all spans created
// and included in logged messages.
func WithResourceAttributes(fields Attributes) OperatorOption {
	return func(op *Operator) error {
		op.userAttributes = join(op.userAttributes, fields)
		return nil
	}
}

// WithLogger set the output handler. If not provided, all output is discarded by default.
func WithLogger(ll xlog.Logger) OperatorOption {
	return func(op *Operator) error {
		op.log = ll
		return nil
	}
}

// WithExporter enables a trace (i.e. span) exporter as data sink for the operator.
// If no exporter is set, all traces are discarded by default.
func WithExporter(exp sdkTrace.SpanExporter) OperatorOption {
	return func(op *Operator) error {
		op.exporter = exp
		return nil
	}
}

// WithMetricExporter enables a metric exporter as data sink for the operator.
// If no exporter is set, all metrics are discarded by default.
func WithMetricExporter(exp sdkMetric.Exporter) OperatorOption {
	return func(op *Operator) error {
		op.metricExporter = exp
		return nil
	}
}

// WithHostMetrics enables the operator to capture the conventional host metric instruments
// specified by OpenTelemetry. Host metric events are sometimes collected through the
// OpenTelemetry Collector `host metrics` receiver running as an agent; this instrumentation
// is an alternative for processes that want to record the same information without an agent.
func WithHostMetrics(capture bool) OperatorOption {
	return func(op *Operator) error {
		op.hostMetrics = capture
		return nil
	}
}

// WithRuntimeMetricsPeriod enables the operator to capture the conventional runtime
// metrics specified by OpenTelemetry. The provided `memoryInterval` values sets the
// minimum interval between calls to runtime.ReadMemStats(), which is a relatively
// expensive call to make frequently. The default interval value is 10 seconds, passing
// a value of 0 uses the default.
func WithRuntimeMetricsPeriod(value time.Duration) OperatorOption {
	return func(op *Operator) error {
		if value.Seconds() < 0 {
			return errors.New("negative runtime memory capture period")
		}
		op.runtimeMetrics = true
		if value != 0 {
			op.runtimeMetricsInt = value
		}
		return nil
	}
}

// WithPrometheusSupport enables the operator instance to collect and provide prometheus
// metrics. If enabled, host and runtime metrics are collected by default, in addition to
// any collector specified here.
func WithPrometheusSupport(extras ...prometheus.Collector) OperatorOption {
	return func(op *Operator) error {
		op.prom = newPrometheusHandler()
		if len(extras) > 0 {
			op.prom.extras = append(op.prom.extras, extras...)
		}
		return nil
	}
}

// WithMetricPushPeriod sets the time interval between each push operation for collected
// metrics. If no value is provided (i.e., 0) the default period is set to 5 seconds.
func WithMetricPushPeriod(value time.Duration) OperatorOption {
	return func(op *Operator) error {
		if value.Seconds() < 0 {
			return errors.New("negative metric push period")
		}
		if value != 0 {
			op.metricsPushInt = value
		}
		return nil
	}
}
