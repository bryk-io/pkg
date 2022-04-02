package otel

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"runtime"

	xlog "go.bryk.io/pkg/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	semConv "go.opentelemetry.io/otel/semconv/v1.7.0"
	rpcCodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding/gzip"
)

const (
	lblSvcName        = string(semConv.ServiceNameKey)
	lblSvcVer         = string(semConv.ServiceVersionKey)
	lblHostArch       = string(semConv.HostArchKey)
	lblHostName       = string(semConv.HostNameKey)
	lblHostOS         = string(semConv.OSTypeKey)
	lblLibName        = string(semConv.TelemetrySDKNameKey)
	lblLibVer         = string(semConv.TelemetrySDKVersionKey)
	lblLibLang        = string(semConv.TelemetrySDKLanguageKey)
	lblProcessRuntime = string(semConv.ProcessRuntimeDescriptionKey)
	lblTraceID        = "telemetry.trace.id"
	lblSpanID         = "telemetry.span.id"
	lblSpanKind       = "telemetry.span.kind"
	lblChildCount     = "telemetry.span.child_count"
	lblDuration       = "duration"
	lblDurationMS     = "duration_ms"
)

// ExporterStdout returns a new trace exporter to send telemetry data
// to standard output.
func ExporterStdout(pretty bool) (*stdouttrace.Exporter, *stdoutmetric.Exporter, error) {
	var traceOpts []stdouttrace.Option
	var metricOpts []stdoutmetric.Option
	if pretty {
		traceOpts = append(traceOpts, stdouttrace.WithPrettyPrint())
		metricOpts = append(metricOpts, stdoutmetric.WithPrettyPrint())
	}

	// Trace exporter
	traceExp, err := stdouttrace.New(traceOpts...)
	if err != nil {
		return nil, nil, err
	}

	// Metric exporter
	metricExp, err := stdoutmetric.New(metricOpts...)
	if err != nil {
		return nil, nil, err
	}

	return traceExp, metricExp, nil
}

// ExporterOTLP returns an initialized OTLP exporter instance.
func ExporterOTLP(
	endpoint string,
	insecure bool,
	headers map[string]string) (*otlptrace.Exporter, *otlpmetric.Exporter, error) {
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
	traceExp, err := otlptrace.New(ctx, otlptracegrpc.NewClient(traceOpts...))
	if err != nil {
		return nil, nil, err
	}

	// Metric exporter
	metricExp, err := otlpmetric.New(ctx, otlpmetricgrpc.NewClient(metricOpts...))
	if err != nil {
		return nil, nil, err
	}
	return traceExp, metricExp, nil
}

// CoreAttributes returns a set of basic environment attributes.
// https://github.com/open-telemetry/opentelemetry-specification/tree/master/specification
func coreAttributes() Attributes {
	core := Attributes{
		lblSvcName:        "service",
		lblHostOS:         runtime.GOOS,
		lblHostArch:       runtime.GOARCH,
		lblProcessRuntime: runtime.Version(),
		lblLibVer:         otel.Version(),
		lblLibName:        "opentelemetry",
		lblLibLang:        "go",
	}
	if host, err := os.Hostname(); err == nil {
		core.Set(lblHostName, host)
	}
	return core
}

// Map an OTEL code value to a valid logging level.
// nolint:gocyclo,deadcode,unused
func codeToLevel(code rpcCodes.Code) xlog.Level {
	switch code {
	// Info
	case rpcCodes.OK:
		return xlog.Info
	case rpcCodes.Canceled:
		return xlog.Info
	case rpcCodes.NotFound:
		return xlog.Info
	case rpcCodes.AlreadyExists:
		return xlog.Info
	// Warning
	case rpcCodes.Unavailable:
		return xlog.Warning
	case rpcCodes.InvalidArgument:
		return xlog.Warning
	case rpcCodes.DeadlineExceeded:
		return xlog.Warning
	case rpcCodes.PermissionDenied:
		return xlog.Warning
	case rpcCodes.Unauthenticated:
		return xlog.Warning
	case rpcCodes.ResourceExhausted:
		return xlog.Warning
	case rpcCodes.FailedPrecondition:
		return xlog.Warning
	case rpcCodes.Aborted:
		return xlog.Warning
	case rpcCodes.OutOfRange:
		return xlog.Warning
	// Errors
	case rpcCodes.Unknown:
		return xlog.Error
	case rpcCodes.Unimplemented:
		return xlog.Error
	case rpcCodes.Internal:
		return xlog.Error
	case rpcCodes.DataLoss:
		return xlog.Error
	// Non-matched codes are identified as errors
	default:
		return xlog.Error
	}
}

// Any creates a new key-value pair instance with a passed name and
// automatic type inference. This is slower, and not type-safe.
func kvAny(k string, value interface{}) attribute.KeyValue {
	if value == nil {
		return attribute.String(k, "<nil>")
	}

	if stringer, ok := value.(fmt.Stringer); ok {
		return attribute.String(k, stringer.String())
	}

	rv := reflect.ValueOf(value)

	// nolint:forcetypeassert
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
