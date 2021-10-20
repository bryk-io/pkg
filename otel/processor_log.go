package otel

import (
	"context"
	"fmt"
	"time"

	xlog "go.bryk.io/pkg/log"
	otelcodes "go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Fields to remove when logging messages.
var noLogFields = []string{
	"stack",
	"error.stack",
	"exception.stacktrace",
}

// Custom `sdktrace.SpanProcessor` that logs all spans as they are completed.
type logSpans struct {
	log xlog.Logger

	// Next is the next SpanProcessor in the chain.
	Next sdktrace.SpanProcessor
}

// OnEnd is used to log a message once each span has ended.
func (f logSpans) OnEnd(s sdktrace.ReadOnlySpan) {
	level := xlog.Info
	if s.Status().Code == otelcodes.Error {
		level = xlog.Error
	}
	for _, event := range s.Events() {
		eventLvl, eventAttrs := f.event(event, f.fields(s, false))
		f.log.WithFields(eventAttrs).Print(eventLvl, event.Name)
	}
	f.log.WithFields(f.fields(s, true)).Printf(level, "%s completed", s.Name())
	f.Next.OnEnd(s)
}

// OnStart is used to log a message when a new span is created.
func (f logSpans) OnStart(parent context.Context, s sdktrace.ReadWriteSpan) {
	f.log.WithFields(f.fields(s.(sdktrace.ReadOnlySpan), false)).Info(s.Name())
	f.Next.OnStart(parent, s)
}

func (f logSpans) Shutdown(ctx context.Context) error {
	return f.Next.Shutdown(ctx)
}

func (f logSpans) ForceFlush(ctx context.Context) error {
	return f.Next.ForceFlush(ctx)
}

func (f logSpans) fields(s sdktrace.ReadOnlySpan, end bool) xlog.Fields {
	// Get span attributes
	fields := Attributes{}
	fields.load(s.Attributes())
	fields.Set(lblSpanID, s.SpanContext().SpanID().String())
	fields.Set(lblSpanKind, s.SpanKind().String())
	fields.Set(lblTraceID, s.SpanContext().TraceID().String())
	if end {
		// Round the duration to the nearest millisecond to avoid unnecessarily
		// large fractional values.
		duration := s.EndTime().Sub(s.StartTime()).Round(1 * time.Millisecond)
		fields.Set(lblDuration, duration.String())
		fields.Set(lblDurationMS, duration.Milliseconds())
	}

	// Remove unwanted fields from logged output
	for _, nl := range noLogFields {
		if st := fields.Get(nl); st != nil {
			delete(fields, nl)
		}
	}

	return xlog.Fields(fields)
}

func (f logSpans) event(event sdktrace.Event, fields xlog.Fields) (xlog.Level, xlog.Fields) {
	eventLvl := xlog.Debug
	attrs := Attributes{}
	attrs.Set("time", event.Time)
	attrs.load(event.Attributes)
	attrs.Join(Attributes(fields))
	for _, nl := range noLogFields {
		if st := attrs.Get(nl); st != nil {
			delete(attrs, nl)
		}
	}
	if lvl := attrs.Get("error.level"); lvl != nil {
		eventLvl = xlog.Level(fmt.Sprintf("%s", lvl))
	}
	return eventLvl, xlog.Fields(attrs)
}
