package sdk

import (
	"context"
	"fmt"
	"time"

	"go.bryk.io/pkg/log"
	"go.bryk.io/pkg/otel"
	otelCodes "go.opentelemetry.io/otel/codes"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
)

// Fields to remove when logging messages.
var noLogFields = []string{
	// commonly used to capture stacktraces
	"stack",
	"stacktrace",
	"error.stack",
	"error.stacktrace",
	"error.report",
	"exception.stack",
	"exception.stacktrace",
	"exception.report",
	// commonly used to hold original error value to report on Sentry
	"sentry.error",
}

// Custom `sdkTrace.SpanProcessor` that logs all spans as they are completed.
type logSpans struct {
	log log.Logger

	// Next is the next SpanProcessor in the chain.
	Next sdkTrace.SpanProcessor
}

// OnStart is used to log a message when a new span is created.
func (f logSpans) OnStart(parent context.Context, s sdkTrace.ReadWriteSpan) {
	// log "start" operation message
	if rs, ok := s.(sdkTrace.ReadOnlySpan); ok {
		f.log.WithFields(f.fields(rs, false)).Info(s.Name())
	}
	f.Next.OnStart(parent, s)
}

// OnEnd is used to log a message when a span has ended.
func (f logSpans) OnEnd(s sdkTrace.ReadOnlySpan) {
	// log "intermediary" operation events
	for _, event := range s.Events() {
		eventLvl, eventAttrs := f.event(event, f.fields(s, false))
		f.log.WithFields(eventAttrs).Print(eventLvl, event.Name)
	}

	// Log "end" operation message
	level := log.Info
	if s.Status().Code == otelCodes.Error {
		level = log.Error
	}
	f.log.WithFields(f.fields(s, true)).Printf(level, "%s completed", s.Name())
	f.Next.OnEnd(s)
}

func (f logSpans) Shutdown(ctx context.Context) error {
	return f.Next.Shutdown(ctx)
}

func (f logSpans) ForceFlush(ctx context.Context) error {
	return f.Next.ForceFlush(ctx)
}

func (f logSpans) fields(s sdkTrace.ReadOnlySpan, end bool) otel.Attributes {
	// Get span attributes
	fields := otel.Attributes{}
	fields.Load(s.Attributes())
	fields.Set(lblSpanID, s.SpanContext().SpanID().String())
	fields.Set(lblSpanKind, s.SpanKind().String())
	fields.Set(lblTraceID, s.SpanContext().TraceID().String())
	if end {
		// Round the duration to the nearest millisecond to avoid unnecessarily
		// large fractional values.
		duration := s.EndTime().Sub(s.StartTime()).Round(1 * time.Millisecond)
		fields.Set(lblDuration, duration.String())
		fields.Set(lblDurationMS, duration.Milliseconds())
		fields.Set(lblChildCount, s.ChildSpanCount())
	}

	// Remove unwanted fields from logged output
	for _, nl := range noLogFields {
		if st := fields.Get(nl); st != nil {
			delete(fields, nl)
		}
	}
	return fields
}

func (f logSpans) event(event sdkTrace.Event, fields otel.Attributes) (log.Level, otel.Attributes) {
	eventLvl := log.Debug
	attrs := otel.Attributes{}
	attrs.Set("time", event.Time)
	attrs.Load(event.Attributes)
	attrs.Join(fields)
	for _, nl := range noLogFields {
		if st := attrs.Get(nl); st != nil {
			delete(attrs, nl)
		}
	}
	if lvl := attrs.Get("error.level"); lvl != nil {
		eventLvl = levelFromString(fmt.Sprintf("%s", lvl))
	}
	return eventLvl, attrs
}
