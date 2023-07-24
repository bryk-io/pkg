package api

import (
	"context"
	"fmt"

	"go.bryk.io/pkg/errors"
	"go.bryk.io/pkg/otel"
	"go.opentelemetry.io/otel/codes"
	semConv "go.opentelemetry.io/otel/semconv/v1.20.0"
	apiTrace "go.opentelemetry.io/otel/trace"
)

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

// End will mark the span as completed. If `err` is not nil, the
// status for the span will be marked as failed.
func (s span) End(err error) {
	// finish task
	if err == nil {
		s.sp.SetStatus(codes.Ok, "")
		s.sp.End()
		return
	}

	// exception event metadata attributes
	attrs := otel.Attributes{}

	// preserve original error value to be reported when
	// using the Sentry integration
	if errPayload, encErr := errCodec.Marshal(err); encErr == nil {
		attrs.Set("sentry.error", string(errPayload))
	}

	// record error
	opts := []apiTrace.EventOption{}
	var se errors.HasStack
	if errors.As(err, &se) {
		// preserve original error stacktrace
		attrs.Set(string(semConv.ExceptionStacktraceKey), fmt.Sprintf("%+v", err))
	} else {
		// if there's no stacktrace in the error already, let the
		// framework capture one
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
func (s span) Event(msg string, attrs ...otel.Attributes) {
	fields := otel.Attributes{}
	fields.Join(attrs...)
	s.sp.AddEvent(msg, apiTrace.WithAttributes(fields.Expand()...))
}
