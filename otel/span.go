package otel

import (
	"context"
	"net/http"
	"runtime/debug"
	"sync"

	xlog "go.bryk.io/pkg/log"
	"go.opentelemetry.io/otel/baggage"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	apitrace "go.opentelemetry.io/otel/trace"
)

// Span represents a unit of work, performed over a certain period of time.
// A span supports 2 independent data mechanisms that need to be properly
// propagated across service boundaries to the spans to be captured correctly.
//
// The trace context provides trace information (trace IDs, span IDs, etc.),
// which ensure that all spans for a single request are part of the same trace.
//
// Baggage, which are arbitrary key/value pairs that you can use to pass
// observability data between services (for example, sharing a customer ID from
// one service to the next).
type Span struct {
	name  string
	kind  SpanKind
	attrs Attributes
	opts  []apitrace.SpanStartOption
	span  apitrace.Span
	ctx   context.Context
	cp    propagation.TextMapPropagator
	mu    sync.Mutex
}

// ID returns the span identifier, if any.
func (s *Span) ID() string {
	if !s.span.SpanContext().HasSpanID() {
		return ""
	}
	return s.span.SpanContext().SpanID().String()
}

// TraceID returns the span's parent trace identifier, if any.
func (s *Span) TraceID() string {
	if !s.span.SpanContext().HasTraceID() {
		return ""
	}
	return s.span.SpanContext().TraceID().String()
}

// End will mark the span as completed.
func (s *Span) End() {
	s.span.End()
}

// SetStatus will update the status of the span.
func (s *Span) SetStatus(code otelcodes.Code, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.span.SetStatus(code, msg)
}

// Context of the span instance. Creating a new span with this context
// will establish a parent -> child relationship.
func (s *Span) Context() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ctx
}

// Event produces a log marker during the execution of the span. The attributes provided
// here will be merged with those provided when creating the span.
func (s *Span) Event(message string, attributes Attributes) {
	s.span.AddEvent(message, apitrace.WithAttributes(attributes.Expand()...))
}

// Error adds an annotation to the span with an error event.
// More information: https://bit.ly/3lqxl5b
func (s *Span) Error(level xlog.Level, err error, attributes Attributes) {
	// Base error details
	fields := Attributes{
		"event":                "error",
		"error.level":          level,
		"error.message":        err.Error(),
		"exception.message":    err.Error(),
		"exception.stacktrace": string(debug.Stack()),
	}
	if level == xlog.Error || level == xlog.Fatal || level == xlog.Panic {
		fields.Set("exception.escaped", true)
	}
	if attributes != nil {
		fields = join(fields, attributes)
	}

	// Log event and record error on the span
	s.Event(err.Error(), fields)
	s.span.RecordError(err)
}

// SetAttributes allows adding values that are applied as metadata to the span and
// are useful for aggregating, filtering, and grouping traces. If a key from the provided
// `attributes` already exists for an attribute of the Span, it will be overwritten
// with the new value provided.
func (s *Span) SetAttributes(attributes Attributes) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attrs = join(s.attrs, attributes)
	s.span.SetAttributes(s.attrs.Expand()...)
}

// SetAttribute allows adding or adjusting a single key/value pair on the span metadata.
func (s *Span) SetAttribute(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attrs.Set(key, value)
	s.span.SetAttributes(s.attrs.Expand()...)
}

// GetAttributes returns the data elements available in the span.
func (s *Span) GetAttributes() Attributes {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.attrs
}

// SetBaggage allows setting arbitrary key/value pairs that you can use to
// propagate observability data between services.
func (s *Span) SetBaggage(attributes Attributes) {
	s.mu.Lock()
	defer s.mu.Unlock()
	bag, _ := baggage.New(attributes.members()...)
	s.ctx = baggage.ContextWithBaggage(s.ctx, bag)
}

// GetBaggage returns any attribute available in the span's baggage context.
func (s *Span) GetBaggage() Attributes {
	s.mu.Lock()
	defer s.mu.Unlock()
	attrs := Attributes{}
	for _, m := range baggage.FromContext(s.ctx).Members() {
		attrs.Set(m.Key(), m.Value())
	}
	return attrs
}

// ClearBaggage will remove any baggage attributes currently set in the span.
func (s *Span) ClearBaggage() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx = baggage.ContextWithoutBaggage(s.ctx)
}

// Headers return the cross-cutting concerns from span context as a set of HTTP
// headers. This is particularly useful when manually propagating the span across
// network boundaries using a non-conventional transport, like Websockets.
func (s *Span) Headers() http.Header {
	headers := http.Header{}
	s.cp.Inject(s.ctx, propagation.HeaderCarrier(headers))
	return headers
}

// Get fields required when logging span-related messages by combining
// the extra attributes provided with those already set on the span.
// nolint: unused
func (s *Span) logFields(extras Attributes) xlog.Fields {
	// NOTE:
	// Right now this function is not being used since we are producing
	// log messages when the span is closed and processed (processor_log).
	// Another alternative is to log messages directly from the span
	// instance (as in the past); TBD.

	// Add span and correlation attributes
	fields := join(s.GetAttributes(), extras)

	// Remove unwanted fields from logged output
	for _, nl := range noLogFields {
		if st := fields.Get(nl); st != nil {
			delete(fields, nl)
		}
	}

	// Add trace context details
	if spanID := s.ID(); spanID != "" {
		fields.Set(lblSpanID, spanID)
	}
	if traceID := s.TraceID(); traceID != "" {
		fields.Set(lblTraceID, traceID)
	}
	fields.Set(lblSpanKind, s.kind)
	return xlog.Fields(fields)
}
