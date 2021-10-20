package otel

import (
	"context"
	"encoding/json"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/baggage"
	apitrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

// Propagator provides a simple mechanism to manually handle JSON encoded span
// context data. This is useful when requiring to manually propagate the span
// details and metadata across non-standard mechanisms, for example when using
// message queue and pub/sub components.
type Propagator struct{}

// Export available span details. Useful when manually propagating a task context
// across process boundaries.
func (jp *Propagator) Export(ctx context.Context) ([]byte, error) {
	md := metadata.New(nil)
	otelgrpc.Inject(ctx, &md)
	return json.Marshal(md)
}

// Restore previously exported span context data.
func (jp *Propagator) Restore(data []byte) (context.Context, error) {
	ctx := context.TODO()
	md := metadata.New(nil)
	if err := json.Unmarshal(data, &md); err != nil {
		return ctx, err
	}
	bag, spanCtx := otelgrpc.Extract(ctx, &md) // extract baggage and span context
	ctx = baggage.ContextWithBaggage(ctx, bag) // restore baggage
	if spanCtx.IsRemote() {                    // restore remote span context
		ctx = apitrace.ContextWithRemoteSpanContext(ctx, spanCtx)
	} else {
		ctx = apitrace.ContextWithSpanContext(ctx, spanCtx)
	}
	return ctx, nil
}
