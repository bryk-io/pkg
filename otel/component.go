package otel

import (
	"context"
	"encoding/json"

	"go.bryk.io/pkg/log"
	"go.bryk.io/pkg/metadata"
	"go.opentelemetry.io/otel/baggage"
	otelCodes "go.opentelemetry.io/otel/codes"
	apiMetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	apiTrace "go.opentelemetry.io/otel/trace"
)

// Component instances provide an abstraction to support all the main
// primitives required to instrument an application (or individual portions
// of one): logs, traces and metrics. Component attributes are attached
// by default to all spans started from it.
type Component struct {
	ot                      apiTrace.Tracer               // underlying OTEL tracer
	spp                     SpanInterceptor               // custom span pre-processor
	attrs                   Attributes                    // base component attributes
	propagator              propagation.TextMapPropagator // context propagation mechanism
	log.Logger                                            // embedded logger instance
	apiMetric.MeterProvider                               // embedded metric provider
}

// Start a new span with the provided details. Remember to call "End()"
// to properly mark the span as completed.
//
//	task := op.Span(context.Background(), "my-task")
//	defer task.End(err)
func (cmp *Component) Start(ctx context.Context, name string, options ...SpanOption) Span {
	// bare span instance
	sp := cmp.newSpan(name)
	for _, opt := range options {
		opt(sp)
	}

	// if available, add baggage values to the span's context
	existingBgg := metadata.New()
	for _, m := range baggage.FromContext(ctx).Members() {
		existingBgg.Set(m.Key(), m.Value())
	}
	bgg, _ := baggage.New(members(join(sp.bgg.Values(), existingBgg.Values()))...)
	ctx = baggage.ContextWithBaggage(ctx, bgg)

	// create OTEL span
	sp.opts = append(sp.opts, apiTrace.WithAttributes(expand(sp.attrs.Values())...))
	sp.ctx, sp.span = cmp.ot.Start(ctx, name, sp.opts...)
	sp.span.SetStatus(otelCodes.Unset, "")
	return sp
}

// SpanFromContext returns a reference to the current span stored in the
// context. You can use this reference to add events to it, but you can't
// close it directly.
//
// You can also use the `Context()` of the managed span to initiate child
// tasks of your own.
func (cmp *Component) SpanFromContext(ctx context.Context) SpanManaged {
	return &span{
		cp:    cmp.propagator,                // context propagation mechanism
		ctx:   ctx,                           // provided context
		span:  apiTrace.SpanFromContext(ctx), // restored span from provided context
		attrs: metadata.New(),                // empty attributes
	}
}

// Export available span details. Useful when manually propagating a task
// context across process boundaries.
func (cmp *Component) Export(ctx context.Context) ([]byte, error) {
	md := propagation.MapCarrier{}
	cmp.propagator.Inject(ctx, md)
	return json.Marshal(md)
}

// Restore previously exported span context data.
func (cmp *Component) Restore(data []byte) (context.Context, error) {
	ctx := context.Background()
	md := propagation.MapCarrier{}
	if err := json.Unmarshal(data, &md); err != nil {
		return ctx, err
	}
	bgg := baggage.FromContext(ctx)                 // restore baggage
	ctx = baggage.ContextWithBaggage(ctx, bgg)      // add baggage to context
	ctx = cmp.propagator.Extract(ctx, md)           // build context with details in the carrier
	spanCtx := apiTrace.SpanContextFromContext(ctx) // restore span context
	if spanCtx.IsRemote() {                         // restore remote span context
		ctx = apiTrace.ContextWithRemoteSpanContext(ctx, spanCtx)
	} else {
		ctx = apiTrace.ContextWithSpanContext(ctx, spanCtx)
	}
	return ctx, nil
}

// Default span structure.
func (cmp *Component) newSpan(name string) *span {
	return &span{
		name:  name,                        // task name
		kind:  SpanKindUnspecified,         // default kind
		spp:   cmp.spp,                     // custom span pre-processor
		bgg:   metadata.New(),              // no baggage by default
		attrs: metadata.FromMap(cmp.attrs), // inherit base component attributes
		cp:    cmp.propagator,              // inherit context propagation mechanism
		opts:  []apiTrace.SpanStartOption{},
	}
}
