package otel

import (
	"context"

	"go.opentelemetry.io/otel/metric"
	expmetric "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	apitrace "go.opentelemetry.io/otel/trace"
)

// NoOpSpan returns an span interface that will not be reported and discard all output.
func NoOpSpan() *Span {
	sp := &Span{
		name:  "no-op",             // task name
		kind:  SpanKindUnspecified, // default kind
		cp:    nil,                 // inherit context propagation mechanism
		attrs: Attributes{},        // empty attributes set
		opts:  []apitrace.SpanStartOption{},
	}
	sp.ctx, sp.span = noOpTraceProvider.Start(context.TODO(), "no-op")
	return sp
}

// No-op trace provider.
var noOpTraceProvider = apitrace.NewNoopTracerProvider().Tracer("no-op")

// No-op exporter.
type noOpExporter struct{}

func (n *noOpExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	return nil
}

func (n *noOpExporter) Shutdown(ctx context.Context) error {
	return nil
}

func (n *noOpExporter) Export(
	ctx context.Context,
	resource *sdkresource.Resource,
	reader expmetric.InstrumentationLibraryReader) error {
	return nil
}

func (n *noOpExporter) ExportKindFor(
	descriptor *metric.Descriptor,
	aggregatorKind aggregation.Kind) expmetric.ExportKind {
	return expmetric.CumulativeExportKind
}
