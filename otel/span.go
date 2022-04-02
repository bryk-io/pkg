package otel

import (
	"context"
	"net/http"
	"sync"

	xlog "go.bryk.io/pkg/log"
	"go.opentelemetry.io/otel/baggage"
	otelCodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	apiTrace "go.opentelemetry.io/otel/trace"
)

// Span represents a unit of work, performed over a certain period of time.
// A span supports 2 independent data mechanisms that need to be properly
// propagated across service boundaries for the spans to be captured correctly.
//
// The trace context provides trace information (trace IDs, span IDs, etc.),
// which ensure that all spans for a single request are part of the same trace.
//
// Baggage, which are arbitrary key/value pairs that you can use to pass
// observability data between services (for example, sharing a customer ID from
// one service to the next).
type Span struct {
	name         string
	kind         SpanKind
	attrs        Attributes
	bgg          Attributes
	bggPropagate bool
	opts         []apiTrace.SpanStartOption
	span         apiTrace.Span
	ctx          context.Context
	cp           propagation.TextMapPropagator
	mu           sync.Mutex
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
func (s *Span) SetStatus(code otelCodes.Code, msg string) {
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

// IsSampled returns if the sampling bit is set in the span context's.
func (s *Span) IsSampled() bool {
	return s.span.SpanContext().IsSampled()
}

// Event produces a log marker during the execution of the span. The attributes
// provided here will be merged with those provided when creating the span.
func (s *Span) Event(message string, attributes Attributes) {
	s.span.AddEvent(message, apiTrace.WithAttributes(attributes.Expand()...))
}

// Error adds an annotation to the span with an error event. If `setStatus` is true
// the status of the span will also be adjusted.
// More information: https://bit.ly/3lqxl5b
func (s *Span) Error(level xlog.Level, err error, attributes Attributes, setStatus bool) {
	// Base error details
	fields := Attributes{
		"event":                "error",
		"error.level":          level,
		"error.message":        err.Error(),
		"exception.stacktrace": getStack(1),
	}
	if level == xlog.Error || level == xlog.Fatal || level == xlog.Panic {
		fields.Set("exception.escaped", true)
	}
	if attributes != nil {
		fields = join(fields, attributes)
	}

	// Record error on the span
	s.span.RecordError(err, apiTrace.WithAttributes(fields.Expand()...))
	if setStatus {
		s.span.SetStatus(otelCodes.Error, err.Error())
	}
}

// GetAttributes returns the data elements available in the span.
func (s *Span) GetAttributes() Attributes {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.attrs
}

// GetBaggage returns any attribute available in the span's bgg context.
func (s *Span) GetBaggage() Attributes {
	s.mu.Lock()
	defer s.mu.Unlock()
	attrs := Attributes{}
	for _, m := range baggage.FromContext(s.ctx).Members() {
		attrs.Set(m.Key(), m.Value())
	}
	return attrs
}

// Headers return the cross-cutting concerns from span context as a set of HTTP
// headers. This is particularly useful when manually propagating the span across
// network boundaries using a non-conventional transport, like Websockets.
func (s *Span) Headers() http.Header {
	headers := http.Header{}
	s.cp.Inject(s.ctx, propagation.HeaderCarrier(headers))
	return headers
}
