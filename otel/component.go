package otel

import (
	"context"
	"encoding/json"

	"go.bryk.io/pkg/log"
	"go.opentelemetry.io/otel/baggage"
	otelCodes "go.opentelemetry.io/otel/codes"
	apiMetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	apiTrace "go.opentelemetry.io/otel/trace"
)

// Component instances provide an abstraction to support all the main primitives
// required to instrument an application (or individual portion of one): logs,
// traces and metrics. Component attributes are attached by default to all spans
// started from it.
type Component struct {
	ot                      apiTrace.Tracer               // underlying OTEL tracer
	propagator              propagation.TextMapPropagator // context propagation mechanism
	attrs                   Attributes                    // base component attributes
	log.Logger                                            // embedded main logger instance
	apiMetric.MeterProvider                               // embedded metric provider
}

// Start a new span with the provided details. Remember to call "End" to properly
// mark the span as completed. All spans are initialized with an "OK" status code
// by default.
//
//    task := op.Span(context.Background(), "my-task")
//    defer task.End()
func (cmp *Component) Start(ctx context.Context, name string, options ...SpanOption) *Span {
	sp := cmp.newSpan(name)
	for _, opt := range options {
		opt(sp)
	}
	if sp.bgg != nil {
		bag, _ := baggage.New(sp.bgg.members()...)
		ctx = baggage.ContextWithBaggage(ctx, bag)
		if sp.bggPropagate {
			sp.attrs.Join(sp.bgg) // attach baggage as attributes
		}
	}
	sp.opts = append(sp.opts, apiTrace.WithAttributes(sp.attrs.Expand()...))
	sp.ctx, sp.span = cmp.ot.Start(ctx, name, sp.opts...)
	sp.span.SetStatus(otelCodes.Ok, "ok")
	return sp
}

// SpanFromContext returns the current span stored in the context. Useful when
// starting a child span across processes boundaries.
func (cmp *Component) SpanFromContext(ctx context.Context, attrs ...Attributes) *Span {
	fields := Attributes{}
	fields.Join(attrs...)
	sp := apiTrace.SpanFromContext(ctx)
	sp.SetAttributes(fields.Expand()...)
	return &Span{
		ctx:   ctx,            // provided context
		span:  sp,             // restored span from provided context
		cp:    cmp.propagator, // context propagation mechanism
		attrs: fields,         // provided attributes
	}
}

// Export available span details. Useful when manually propagating a task context
// across process boundaries.
func (cmp *Component) Export(ctx context.Context) ([]byte, error) {
	md := propagation.MapCarrier{}
	cmp.propagator.Inject(ctx, md)
	return json.Marshal(md)
}

// Restore previously exported span context data.
func (cmp *Component) Restore(data []byte) (context.Context, error) {
	ctx := context.TODO()
	md := propagation.MapCarrier{}
	if err := json.Unmarshal(data, &md); err != nil {
		return ctx, err
	}
	ctx = cmp.propagator.Extract(ctx, md)           // build context with details in the carrier
	bgg := baggage.FromContext(ctx)                 // restore baggage
	spanCtx := apiTrace.SpanContextFromContext(ctx) // restore span context
	ctx = baggage.ContextWithBaggage(ctx, bgg)      // add baggage to context
	if spanCtx.IsRemote() {                         // restore remote span context
		ctx = apiTrace.ContextWithRemoteSpanContext(ctx, spanCtx)
	} else {
		ctx = apiTrace.ContextWithSpanContext(ctx, spanCtx)
	}
	return ctx, nil
}

// Default span structure.
func (cmp *Component) newSpan(name string) *Span {
	return &Span{
		name:  name,                // task name
		kind:  SpanKindUnspecified, // default kind
		cp:    cmp.propagator,      // inherit context propagation mechanism
		attrs: cmp.attrs,           // inherit base component attributes
		bgg:   nil,                 // no baggage by default
		opts:  []apiTrace.SpanStartOption{},
	}
}
