package api

import (
	"context"
	"fmt"

	"go.bryk.io/pkg/errors"
	"go.bryk.io/pkg/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semConv "go.opentelemetry.io/otel/semconv/v1.30.0"
	apiTrace "go.opentelemetry.io/otel/trace"
)

// ExceptionReportKey is the attribute Key conforming to the
// "exception.report" semantic conventions. It represents a report
// obtained from the error provided when closing/ending a span.
//
//	Type: string
//	RequirementLevel: Optional
//	Stability: stable
var ExceptionReportKey = attribute.Key("exception.report")

// Tracer instances can be used to create spans.
type Tracer interface {
	// Start a new span.
	Start(ctx context.Context, name string, opts ...SpanOption) Span
}

type tracer struct {
	tr apiTrace.Tracer
}

func (t tracer) Start(ctx context.Context, name string, opts ...SpanOption) Span {
	conf := defaultSpanConf()
	for _, opt := range opts {
		opt(conf)
	}
	ctx, sp := t.tr.Start(ctx, name, conf.startOpts()...)
	return span{
		ctx: ctx,
		sp:  sp,
	}
}

type span struct {
	sp  apiTrace.Span
	ctx context.Context
}

// Unwrap returns the underlying OpenTelemetry span instance.
// Useful for integrating with 3rd party libraries.
func (s span) Unwrap() apiTrace.Span {
	return s.sp
}

// End will mark the span as completed. If `err` is not nil, the
// status for the span will be marked as failed. If the provided
// error implements the `errors.HasStack` interface, the original
// stacktrace will be preserved in the span's attributes. If a
// report can be extracted from the error, it will be stored in the
// `exception.report` attribute.
func (s span) End(err error) {
	// finish task
	if err == nil {
		s.sp.SetStatus(codes.Ok, "")
		s.sp.End()
		return
	}

	// exception event metadata attributes
	attrs := otel.Attributes{}

	// record error
	opts := []apiTrace.EventOption{}
	var se errors.HasStack
	if errors.As(err, &se) {
		// add the `exception.report` if we are able to extract one
		// from the error instance
		if errPayload, encErr := errCodec.Marshal(err); encErr == nil {
			attrs.Set(string(ExceptionReportKey), string(errPayload))
		}

		// add the `exception.stacktrace` and prevent the framework from
		// duplicating/replacing the value
		attrs.Set(string(semConv.ExceptionStacktraceKey), fmt.Sprintf("%+v", err))
		opts = append(opts, apiTrace.WithStackTrace(false))
	} else {
		// if there's no stacktrace in the error already, let the
		// framework try to capture one
		opts = append(opts, apiTrace.WithStackTrace(true))
	}
	opts = append(opts, apiTrace.WithAttributes(attrs.Expand()...))
	s.sp.RecordError(err, opts...)
	s.sp.SetStatus(codes.Error, err.Error())
	s.sp.End()
}

// Context of the span instance. Creating a new span with this context
// will establish a parent -> child relationship.
func (s span) Context() context.Context {
	return s.ctx
}

// ID returns the span identifier, if any.
func (s span) ID() string {
	return s.sp.SpanContext().SpanID().String()
}

// TraceID returns the span's parent trace identifier, if any.
func (s span) TraceID() string {
	return s.sp.SpanContext().TraceID().String()
}

// IsSampled returns if the sampling bit is set in the span context's.
func (s span) IsSampled() bool {
	return s.sp.SpanContext().IsSampled()
}

// Event produces a log marker during the execution of the span.
func (s span) Event(msg string, attrs ...map[string]any) {
	fields := otel.Attributes{}
	for _, attr := range attrs {
		fields.Join(attr)
	}
	s.sp.AddEvent(msg, apiTrace.WithAttributes(fields.Expand()...))
}

// SetAttribute adjust `key` to report `value` as attribute of the Span.
// If a `key` already exists for an attribute of the Span it will be
// overwritten with `value`.
func (s span) SetAttribute(key string, value any) {
	s.sp.SetAttributes(otel.Attributes{key: value}.Expand()...)
}

// SetAttributes adjust multiple attributes of the Span.
func (s span) SetAttributes(attributes map[string]any) {
	attrs := otel.Attributes(attributes)
	s.sp.SetAttributes(attrs.Expand()...)
}
