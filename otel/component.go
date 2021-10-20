package otel

import (
	"context"

	xlog "go.bryk.io/pkg/log"
	otelcodes "go.opentelemetry.io/otel/codes"
	apimetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	apitrace "go.opentelemetry.io/otel/trace"
)

// Component elements provide an abstraction to support all the main primitives
// required to instrument an application (or individual portion of one): logs,
// traces and metrics.
type Component struct {
	ot                      apitrace.Tracer               // underlying OTEL tracer
	propagator              propagation.TextMapPropagator // context propagation mechanism
	attrs                   Attributes                    // base component attributes
	xlog.Logger                                           // embedded main logger instance
	apimetric.MeterProvider                               // embedded metric provider
}

// Start a new span with the provided details. Remember to call "End" to properly
// mark the span as completed. All spans are initialized with an "OK" status code
// by default.
//
//    task := op.Span(context.Background(), "my-task")
//    defer task.End()
func (cmp *Component) Start(ctx context.Context, name string, options ...SpanOption) *Span {
	sp := cmp.newSpan(name)
	options = append(options, WithSpanAttributes(cmp.attrs)) // attach component attributes
	for _, opt := range options {
		opt(sp)
	}
	sp.ctx, sp.span = cmp.ot.Start(ctx, name, sp.opts...)
	sp.span.SetStatus(otelcodes.Ok, "ok")
	return sp
}

// SpanFromContext returns the current span stored in the context. Useful when
// starting a child span across processes boundaries.
func (cmp *Component) SpanFromContext(ctx context.Context) *Span {
	sp := &Span{
		ctx:   ctx,                           // provided context
		span:  apitrace.SpanFromContext(ctx), // restored span from provided context
		cp:    cmp.propagator,                // context propagation mechanism
		attrs: Attributes{},                  // empty attributes set
	}
	return sp
}

// Default span structure.
func (cmp *Component) newSpan(name string) *Span {
	return &Span{
		name:  name,                // task name
		kind:  SpanKindUnspecified, // default kind
		cp:    cmp.propagator,      // inherit context propagation mechanism
		attrs: cmp.attrs,           // inherit base component attributes
		opts:  []apitrace.SpanStartOption{},
	}
}
