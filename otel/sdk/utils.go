package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"go.bryk.io/pkg/log"
	"go.bryk.io/pkg/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdkMetric "go.opentelemetry.io/otel/sdk/metric"
	sdkResource "go.opentelemetry.io/otel/sdk/resource"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	semConv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding/gzip"
)

// Trace conventions
// https://opentelemetry.io/docs/concepts/signals/traces

const (
	lblSvcName          = string(semConv.ServiceNameKey)
	lblSvcVer           = string(semConv.ServiceVersionKey)
	lblStackTrace       = string(semConv.ExceptionStacktraceKey)
	lblExceptionMessage = string(semConv.ExceptionMessageKey)
	lblExceptionType    = string(semConv.ExceptionTypeKey)
	lblTraceID          = "telemetry.trace.id"
	lblSpanID           = "telemetry.span.id"
	lblSpanKind         = "telemetry.span.kind"
	lblChildCount       = "telemetry.span.child_count"
	lblDuration         = "duration"
	lblDurationMS       = "duration_ms"
	lblErrorMsg         = "error.message"
)

// WithExporterStdout is a utility method to automatically setup and attach
// trace and metric exporters to send the generated telemetry data to standard
// output.
func WithExporterStdout(pretty bool) []Option {
	var opts []Option
	se, me, err := ExporterStdout(pretty)
	if err == nil {
		opts = append(opts, WithSpanExporter(se))
		opts = append(opts, WithMetricExporter(me))
	}
	return opts
}

// WithExporterOTLP is a utility method to automatically setup and attach
// trace and metric exporters to send the generated telemetry data to an OTLP
// exporter instance.
// https://opentelemetry.io/docs/collector/
func WithExporterOTLP(endpoint string, insecure bool, headers map[string]string, protocol string) []Option {
	var opts []Option
	se, me, err := ExporterOTLP(endpoint, insecure, headers, protocol)
	if err == nil {
		opts = append(opts, WithSpanExporter(se))
		opts = append(opts, WithMetricExporter(me))
	}
	return opts
}

// ExporterStdout returns a new trace exporter to send telemetry data
// to standard output.
func ExporterStdout(pretty bool) (sdkTrace.SpanExporter, sdkMetric.Exporter, error) {
	var traceOpts []stdouttrace.Option
	if pretty {
		traceOpts = append(traceOpts, stdouttrace.WithPrettyPrint())
	}

	// Trace exporter
	traceExp, err := stdouttrace.New(traceOpts...)
	if err != nil {
		return nil, nil, err
	}

	// Metric exporter
	metricExp, err := stdoutmetric.New()
	if err != nil {
		return nil, nil, err
	}
	return traceExp, metricExp, nil
}

// ExporterOTLP returns an initialized OTLP exporter instance utilizing
// the requested protocol.
func ExporterOTLP(endpoint string, insecure bool, headers map[string]string, protocol string) (sdkTrace.SpanExporter, sdkMetric.Exporter, error) { // nolint:lll
	if protocol == "http" {
		return otlpHTTP(endpoint, insecure, headers)
	}
	return otlpGRPC(endpoint, insecure, headers)
}

// Returns an initialized OTLP exporter instance utilizing
// HTTP with protobuf payloads. The default endpoint for the collector
// is "localhost:4318".
func otlpHTTP(endpoint string, insecure bool, headers map[string]string) (sdkTrace.SpanExporter, sdkMetric.Exporter, error) { // nolint:lll
	if endpoint == "" {
		endpoint = "localhost:4318"
	}
	ctx := context.Background()
	traceOpts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithHeaders(headers),
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
	}
	metricOpts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithHeaders(headers),
		otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression),
	}
	if insecure {
		traceOpts = append(traceOpts, otlptracehttp.WithInsecure())
		metricOpts = append(metricOpts, otlpmetrichttp.WithInsecure())
	}

	// Trace exporter
	traceExp, err := otlptracehttp.New(ctx, traceOpts...)
	if err != nil {
		return nil, nil, err
	}

	// Metric exporter
	metricExp, err := otlpmetrichttp.New(ctx, metricOpts...)
	if err != nil {
		return nil, nil, err
	}
	return traceExp, metricExp, nil
}

// Returns an initialized OTLP exporter instance utilizing gRPC.
// The default endpoint for the collector is "localhost:4317".
func otlpGRPC(endpoint string, insecure bool, headers map[string]string) (sdkTrace.SpanExporter, sdkMetric.Exporter, error) { // nolint:lll
	if endpoint == "" {
		endpoint = "localhost:4317"
	}
	ctx := context.Background()
	traceOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithHeaders(headers),
		otlptracegrpc.WithCompressor(gzip.Name),
	}
	metricOpts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithHeaders(headers),
		otlpmetricgrpc.WithCompressor(gzip.Name),
	}
	if insecure {
		traceOpts = append(traceOpts, otlptracegrpc.WithInsecure())
		metricOpts = append(metricOpts, otlpmetricgrpc.WithInsecure())
	} else {
		creds := credentials.NewClientTLSFromCert(nil, "")
		traceOpts = append(traceOpts, otlptracegrpc.WithTLSCredentials(creds))
		metricOpts = append(metricOpts, otlpmetricgrpc.WithTLSCredentials(creds))
	}

	// Trace exporter
	traceExp, err := otlptracegrpc.New(ctx, traceOpts...)
	if err != nil {
		return nil, nil, err
	}

	// Metric exporter
	metricExp, err := otlpmetricgrpc.New(ctx, metricOpts...)
	if err != nil {
		return nil, nil, err
	}
	return traceExp, metricExp, nil
}

// Collect environment information and setup the OTEL resource.
func setupResource(attrs otel.Attributes) (*sdkResource.Resource, error) {
	return sdkResource.New(context.Background(),
		sdkResource.WithOS(),
		sdkResource.WithHost(),
		sdkResource.WithContainer(),
		sdkResource.WithFromEnv(),
		sdkResource.WithTelemetrySDK(),
		sdkResource.WithProcessRuntimeName(),
		sdkResource.WithProcessRuntimeVersion(),
		sdkResource.WithProcessRuntimeDescription(),
		sdkResource.WithAttributes(expand(attrs)...))
}

// Match simple identifiers with log level values.
func levelFromString(val string) log.Level {
	switch val {
	case "debug":
		return log.Debug
	case "info":
		return log.Info
	case "warning":
		return log.Warning
	case "error":
		return log.Error
	case "panic":
		return log.Panic
	case "fatal":
		return log.Fatal
	default:
		return log.Debug
	}
}

// Expand allows converting from attributes to a key/value list.
func expand(attrs otel.Attributes) []attribute.KeyValue {
	var list []attribute.KeyValue
	for k, v := range attrs {
		if strings.TrimSpace(k) != "" {
			list = append(list, kvAny(k, v))
		}
	}
	return list
}

// Join any number of attribute sets into a single collection.
// Duplicated values are override int the order in which the sets
// containing those values are presented to Join.
func join(list ...otel.Attributes) otel.Attributes {
	out := otel.Attributes{}
	for _, md := range list {
		for k, v := range md {
			if strings.TrimSpace(k) != "" {
				out[k] = v
			}
		}
	}
	return out
}

// Any creates a new key-value pair instance with a passed name and
// automatic type inference. This is slower, and not type-safe.
func kvAny(k string, value any) attribute.KeyValue {
	if value == nil {
		return attribute.String(k, "<nil>")
	}

	if stringer, ok := value.(fmt.Stringer); ok {
		return attribute.String(k, stringer.String())
	}

	rv := reflect.ValueOf(value)

	// nolint:forcetypeassert, errcheck
	switch rv.Kind() {
	case reflect.Array:
		rv = rv.Slice(0, rv.Len())
		fallthrough
	case reflect.Slice:
		switch reflect.TypeOf(value).Elem().Kind() {
		case reflect.Bool:
			return attribute.BoolSlice(k, rv.Interface().([]bool))
		case reflect.Int:
			return attribute.IntSlice(k, rv.Interface().([]int))
		case reflect.Int64:
			return attribute.Int64Slice(k, rv.Interface().([]int64))
		case reflect.Float64:
			return attribute.Float64Slice(k, rv.Interface().([]float64))
		case reflect.String:
			return attribute.StringSlice(k, rv.Interface().([]string))
		default:
			return attribute.String(k, "<nil>")
		}
	case reflect.Bool:
		return attribute.Bool(k, rv.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return attribute.Int64(k, rv.Int())
	case reflect.Float64:
		return attribute.Float64(k, rv.Float())
	case reflect.String:
		return attribute.String(k, rv.String())
	default:
		if b, err := json.Marshal(value); b != nil && err == nil {
			return attribute.String(k, string(b))
		}
		return attribute.String(k, fmt.Sprint(value))
	}
}
