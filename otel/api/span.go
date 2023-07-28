package api

import (
	"context"

	"go.bryk.io/pkg/otel"
)

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
	// End will mark the span as completed. If `err` is not nil, the
	// status for the span will be marked as failed.
	End(err error)

	SpanManaged // inherit "managed" span functionality
}

// SpanManaged represents a unit of work that was initiated by another
// component. You may get a read-only reference to the span to inspect
// it or add additional events to it but you won't be able to close it
// directly.
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

	// Event produces a log marker during the execution of the span.
	Event(msg string, attributes ...otel.Attributes)

	// SetAttribute adjust `key` to report `value` as attribute of the Span.
	// If a `key` already exists for an attribute of the Span it will be
	// overwritten with `value`.
	SetAttribute(key string, value interface{})
}
