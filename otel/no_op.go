package otel

import (
	"context"

	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	apiTrace "go.opentelemetry.io/otel/trace"
)

// NoOpSpan returns a span interface that will not be reported and discard all output.
func NoOpSpan() Span {
	sp := &span{
		name:  "no-op",             // task name
		kind:  SpanKindUnspecified, // default kind
		cp:    nil,                 // inherit context propagation mechanism
		attrs: Attributes{},        // empty attributes set
		opts:  []apiTrace.SpanStartOption{},
	}
	sp.ctx, sp.span = noOpTraceProvider.Start(context.TODO(), "no-op")
	return sp
}

// No-op trace provider.
var noOpTraceProvider = apiTrace.NewNoopTracerProvider().Tracer("no-op")

// No-op exporter.
type noOpExporter struct{}

func (n *noOpExporter) ExportSpans(_ context.Context, _ []sdkTrace.ReadOnlySpan) error {
	return nil
}

func (n *noOpExporter) Shutdown(_ context.Context) error {
	return nil
}
