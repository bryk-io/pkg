package otel

import (
	"context"
	"net/http"
	"sync"

	"go.bryk.io/pkg/log"
	apiErrors "go.bryk.io/pkg/otel/errors"
	"go.opentelemetry.io/otel/baggage"
	otelCodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	apiTrace "go.opentelemetry.io/otel/trace"
)

// SpanManaged represents a unit of work that was initiated by another
// component. You may get a reference to the span to add events or additional
// metadata, but you can't close it directly.
//
// You can also use the `Context()` of the managed span to initiate child
// tasks of your own.
type SpanManaged interface {
	// Context of the span instance. Creating a new span with this context
	// will establish a parent -> child relationship.
	Context() context.Context

	// ID returns the span identifier, if any.
	ID() string

	// TraceID returns the span's parent trace identifier, if any.
	TraceID() string

	// IsSampled returns if the sampling bit is set in the span context's.
	IsSampled() bool

	// Event produces a log marker during the execution of the span. The attributes
	// provided here will be merged with those provided when creating the span.
	Event(message string, attributes ...Attributes)

	// Error adds an annotation to the span with an error event.
	// More information: https://bit.ly/3lqxl5b
	Error(level log.Level, err error, attributes Attributes)

	// GetAttributes returns the data elements available in the span.
	GetAttributes() Attributes

	// GetBaggage returns any attribute available in the span's bgg context.
	GetBaggage() Attributes

	// Headers return the cross-cutting concerns from span context as a set of HTTP
	// headers. This is particularly useful when manually propagating the span across
	// network boundaries using a non-conventional transport, like Websockets.
	Headers() http.Header
}

// Span represents a unit of work, performed over a certain period of time.
// You MUST finish all spans you create using the `End` method.
//
// A span supports 2 independent data mechanisms that need to be properly
// propagated across service boundaries for the spans to be captured correctly.
//
// The trace context provides trace information (trace IDs, span IDs, etc.),
// which ensure that all spans for a single request are part of the same trace.
//
// Baggage, which are arbitrary key/value pairs that you can use to pass
// observability data between services (for example, sharing a customer ID from
// one service to the next).
type Span interface {
	// End will mark the span as completed.
	End()

	// SetStatus will update the status of the span.
	SetStatus(code otelCodes.Code, msg string)

	SpanManaged // inherit "managed" span functionality
}

type span struct {
	name  string
	kind  SpanKind
	attrs Attributes
	bgg   Attributes
	opts  []apiTrace.SpanStartOption
	span  apiTrace.Span
	ctx   context.Context
	cp    propagation.TextMapPropagator
	op    apiErrors.Operation
	mu    sync.Mutex
}

// ID returns the span identifier, if any.
func (s *span) ID() string {
	if !s.span.SpanContext().HasSpanID() {
		return ""
	}
	return s.span.SpanContext().SpanID().String()
}

// TraceID returns the span's parent trace identifier, if any.
func (s *span) TraceID() string {
	if !s.span.SpanContext().HasTraceID() {
		return ""
	}
	return s.span.SpanContext().TraceID().String()
}

// End will mark the span as completed.
func (s *span) End() {
	s.op.Finish()
	s.span.End()
}

// SetStatus will update the status of the span.
func (s *span) SetStatus(code otelCodes.Code, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.span.SetStatus(code, msg)
	if code != otelCodes.Ok {
		s.op.Status("error")
	}
}

// Context of the span instance. Creating a new span with this context
// will establish a parent -> child relationship.
func (s *span) Context() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ctx
}

// IsSampled returns if the sampling bit is set in the span context's.
func (s *span) IsSampled() bool {
	return s.span.SpanContext().IsSampled()
}

// Event produces a log marker during the execution of the span. The attributes
// provided here will be merged with those provided when creating the span.
func (s *span) Event(message string, attributes ...Attributes) {
	attrs := join(attributes...)
	s.span.AddEvent(message, apiTrace.WithAttributes(attrs.Expand()...))
	s.op.Event(message, attrs)
}

// Error adds an annotation to the span with an error event. If `setStatus` is true
// the status of the span will also be adjusted.
// More information: https://bit.ly/3lqxl5b
func (s *span) Error(level log.Level, err error, attributes Attributes) {
	// Base error details
	fields := Attributes{
		"event":                "error",
		"error.level":          level,
		"error.message":        err.Error(),
		"exception.stacktrace": getStack(1),
	}
	if attributes != nil {
		fields = join(fields, attributes)
	}

	// Record error on the span
	s.span.RecordError(err, apiTrace.WithAttributes(fields.Expand()...))

	// Report error
	if s.IsSampled() {
		s.op.Level(string(level))
		s.op.Tags(join(s.GetAttributes(), attributes))
		if bgg := s.GetBaggage(); len(bgg) > 0 {
			s.op.Segment("Baggage", bgg)
		}
		s.op.Segment("OTEL", map[string]interface{}{
			"trace.id": s.TraceID(),
			"span.id":  s.ID(),
		})
		s.op.Report(err)
	}
}

// GetAttributes returns the data elements available in the span.
func (s *span) GetAttributes() Attributes {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.attrs
}

// GetBaggage returns any attribute available in the span's bgg context.
func (s *span) GetBaggage() Attributes {
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
func (s *span) Headers() http.Header {
	headers := http.Header{}
	s.cp.Inject(s.ctx, propagation.HeaderCarrier(headers))
	s.op.Inject(propagation.HeaderCarrier(headers))
	return headers
}
