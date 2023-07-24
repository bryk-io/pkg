package api

import (
	"context"
	"encoding/json"
	"net/http"

	"go.bryk.io/pkg/errors"
	apiOtel "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	apiTrace "go.opentelemetry.io/otel/trace"
)

// default global tracer.
var pkgTracer apiTrace.Tracer

// default global error codec; used to transmit errors associated
// with individual spans to other components in the instrumentation
// pipeline.
var errCodec errors.Codec

// default value for `otel.library.name`.
const tracerName = "go.bryk.io/pkg/otel"

// default value for `otel.library.version`.
const tracerVersion = "0.2.0"

func init() {
	// setup global tracer instance
	pkgTracer = apiOtel.Tracer(tracerName, apiTrace.WithInstrumentationVersion(tracerVersion))

	// setup main error codec
	errCodec = errors.CodecJSON(false)
}

// GetTracer returns the global tracer instance.
func GetTracer() Tracer {
	return tracer{tr: pkgTracer}
}

// Start a new span using the global tracer instance. Remember to
// mark it as complete using `End` when done.
//
//	task := Start(context.TODO(), "my-task")
//	defer task.End(nil)
func Start(ctx context.Context, name string, opts ...SpanOption) Span {
	return GetTracer().Start(ctx, name, opts...)
}

// Export available span details. Useful when manually propagating a task
// context across process boundaries.
func Export(ctx context.Context) ([]byte, error) {
	md := propagation.MapCarrier{}
	apiOtel.GetTextMapPropagator().Inject(ctx, md)
	return json.Marshal(md)
}

// Restore previously exported span context data.
func Restore(data []byte) (context.Context, error) {
	ctx := context.Background()
	md := propagation.MapCarrier{}
	if err := json.Unmarshal(data, &md); err != nil {
		return ctx, err
	}
	bgg := baggage.FromContext(ctx)                       // restore baggage
	ctx = baggage.ContextWithBaggage(ctx, bgg)            // add baggage to context
	ctx = apiOtel.GetTextMapPropagator().Extract(ctx, md) // build context with details in the carrier
	spanCtx := apiTrace.SpanContextFromContext(ctx)       // restore span context
	if spanCtx.IsRemote() {                               // restore remote span context
		ctx = apiTrace.ContextWithRemoteSpanContext(ctx, spanCtx)
	} else {
		ctx = apiTrace.ContextWithSpanContext(ctx, spanCtx)
	}
	return ctx, nil
}

// Headers return the cross-cutting concerns from span context as a set of HTTP
// headers. This is particularly useful when manually propagating the span across
// network boundaries using a non-conventional transport, like Websockets.
func Headers(sp SpanManaged) http.Header {
	headers := http.Header{}
	apiOtel.GetTextMapPropagator().Inject(sp.Context(), propagation.HeaderCarrier(headers))
	return headers
}
