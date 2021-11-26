package otel

import (
	"context"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	apitrace "go.opentelemetry.io/otel/trace"
)

// NoOpSpan returns a span interface that will not be reported and discard all output.
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
