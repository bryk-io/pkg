package otel

import (
	"context"
	"encoding/json"

	"go.bryk.io/pkg/log"
	"go.bryk.io/pkg/metadata"
	apiErrors "go.bryk.io/pkg/otel/errors"
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
	attrs                   Attributes                    // base component attributes
	reporter                apiErrors.Reporter            // error reporter
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

	// create error reporting operation
	// - store operation reference in span's context
	// - add span attributes as operation tags; don't use baggage values
	//   since those usually have very high cardinality and are not meant
	//   to be used as event filters/selectors
	sp.op = cmp.reporter.Start(ctx, name)
	sp.op.Tags(sp.attrs.Values())
	ctx = cmp.reporter.ToContext(ctx, sp.op)

	// attach user data if provided in span attributes
	if usr, ok := extractUser(sp.attrs.Values()); ok {
		sp.op.User(usr)
	}

	// if available, add baggage values to the span's context
	if !sp.bgg.IsEmpty() {
		bag, _ := baggage.New(members(sp.bgg.Values())...)
		ctx = baggage.ContextWithBaggage(ctx, bag)

		// add baggage values as an error reporting operation "segment" too
		sp.op.Segment("Baggage", sp.bgg.Values())
	}

	// create OTEL span
	sp.opts = append(sp.opts, apiTrace.WithAttributes(expand(sp.attrs.Values())...))
	sp.ctx, sp.span = cmp.ot.Start(ctx, name, sp.opts...)
	sp.span.SetStatus(otelCodes.Unset, "")

	// add OTEL details as a "segment" on error reporting operation
	sp.op.Segment("OTEL", map[string]interface{}{
		"trace.id": sp.TraceID(),
		"span.id":  sp.ID(),
	})
	return sp
}

// SpanFromContext returns a reference to the current span stored in the
// context. You can use this reference to add events to it, but you can't
// close it directly.
//
// You can also use the `Context()` of the managed span to initiate child
// tasks of your own.
func (cmp *Component) SpanFromContext(ctx context.Context) SpanManaged {
	// retrieve OTEL span from `ctx`
	sp := apiTrace.SpanFromContext(ctx)

	// retrieve error reporting operation from `ctx`
	op := cmp.reporter.FromContext(ctx)
	if op == nil {
		op = apiErrors.NoOpOperation()
	}

	return &span{
		op:    op,             // error reporter operation
		cp:    cmp.propagator, // context propagation mechanism
		ctx:   ctx,            // provided context
		span:  sp,             // restored span from provided context
		attrs: metadata.New(), // provided attributes
	}
}

// Export available span details. Useful when manually propagating a task
// context across process boundaries.
func (cmp *Component) Export(ctx context.Context) ([]byte, error) {
	md := propagation.MapCarrier{}
	cmp.propagator.Inject(ctx, md)
	if op := cmp.reporter.FromContext(ctx); op != nil {
		op.Inject(md)
	}
	return json.Marshal(md)
}

// Restore previously exported span context data.
func (cmp *Component) Restore(data []byte) (context.Context, error) {
	ctx := context.Background()
	md := propagation.MapCarrier{}
	if err := json.Unmarshal(data, &md); err != nil {
		return ctx, err
	}
	ctx = cmp.propagator.Extract(ctx, md)           // build context with details in the carrier
	bgg := baggage.FromContext(ctx)                 // restore baggage
	spanCtx := apiTrace.SpanContextFromContext(ctx) // restore span context
	ctx = baggage.ContextWithBaggage(ctx, bgg)      // add baggage to context
	ctx = cmp.reporter.Extract(ctx, md)             // restore error reporting operation
	if spanCtx.IsRemote() {                         // restore remote span context
		ctx = apiTrace.ContextWithRemoteSpanContext(ctx, spanCtx)
	} else {
		ctx = apiTrace.ContextWithSpanContext(ctx, spanCtx)
	}
	return ctx, nil
}

// ErrorReporter returns the error reporting instance setup with the component.
func (cmp *Component) ErrorReporter() apiErrors.Reporter {
	return cmp.reporter
}

// Default span structure.
func (cmp *Component) newSpan(name string) *span {
	return &span{
		name:  name,                        // task name
		kind:  SpanKindUnspecified,         // default kind
		bgg:   metadata.New(),              // no baggage by default
		attrs: metadata.FromMap(cmp.attrs), // inherit base component attributes
		cp:    cmp.propagator,              // inherit context propagation mechanism
		opts:  []apiTrace.SpanStartOption{},
	}
}
