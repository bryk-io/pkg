package sdk

import (
	"context"

	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
)

// No-op exporter.
type noOpExporter struct{}

func (n *noOpExporter) ExportSpans(_ context.Context, _ []sdkTrace.ReadOnlySpan) error {
	return nil
}

func (n *noOpExporter) Shutdown(_ context.Context) error {
	return nil
}
