package sentry

// Based on the original: github.com/getsentry/sentry-go/otel

import (
	"context"

	sdk "github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type sentryPropagator struct{}

func newSentryPropagator() propagation.TextMapPropagator {
	return &sentryPropagator{}
}

// Inject sets Sentry-related values from the Context into the carrier.
//
// https://opentelemetry.io/docs/reference/specification/context/api-propagators/#inject
func (p sentryPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	var (
		traceHeader string
		baggageStr  string
		sentrySpan  *sdk.Span
	)
	spanContext := trace.SpanContextFromContext(ctx)

	// Get sentry span from the span map
	if spanContext.IsValid() {
		sentrySpan, _ = sentrySpanMap.Get(spanContext.SpanID())
	}

	// Propagate `sentry-trace` header
	if sentrySpan != nil {
		// Sentry span exists; generate "sentry-trace" from it and retrieve baggage
		traceHeader = sentrySpan.ToSentryTrace()
		baggageStr = sentrySpan.GetTransaction().ToBaggage()
	} else {
		// No span; propagate the incoming sentry-trace header, if exists
		traceHeader, _ = ctx.Value(sentryTraceHeaderContextKey{}).(string)
	}
	carrier.Set(sdk.SentryTraceHeader, traceHeader)

	// Propagate `baggage` header; preserving the original values
	// TODO: look for a more performant way to merge the baggage values
	bgg, _ := baggage.Parse(baggageStr)
	for _, m := range baggage.FromContext(ctx).Members() {
		bgg, _ = bgg.SetMember(m)
	}
	if bgg.Len() > 0 {
		carrier.Set(sdk.SentryBaggageHeader, bgg.String())
	}
}

// Extract reads cross-cutting concerns from the carrier into a Context.
//
// https://opentelemetry.io/docs/reference/specification/context/api-propagators/#extract
func (p sentryPropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	// preserve trace state
	sentryTraceHeader := carrier.Get(sdk.SentryTraceHeader)
	if sentryTraceHeader != "" {
		ctx = context.WithValue(ctx, sentryTraceHeaderContextKey{}, sentryTraceHeader)
		if traceParentContext, valid := sdk.ParseTraceParentContext([]byte(sentryTraceHeader)); valid {
			// Save traceParentContext because we'll at least need to know the
			// original `sampled` value in the span processor.
			ctx = context.WithValue(ctx, sentryTraceParentContextKey{}, traceParentContext)
			ctx = trace.ContextWithSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    trace.TraceID(traceParentContext.TraceID),
				SpanID:     trace.SpanID(traceParentContext.ParentSpanID),
				TraceFlags: trace.FlagsSampled,
				Remote:     true,
			}))
		}
	}

	// preserve the original baggage, if any
	baggageHeader := carrier.Get(sdk.SentryBaggageHeader)
	if baggageHeader != "" {
		if parsedBaggage, err := baggage.Parse(baggageHeader); err == nil {
			ctx = context.WithValue(ctx, baggageContextKey{}, parsedBaggage)
		}
	}

	// preserve sampling context; following cases should already be covered:
	// * We can extract a valid dynamic sampling context (DSC) from the baggage
	// * No baggage header is present
	// * No Sentry-related values are present
	// * We cannot parse the baggage header for whatever reason
	dynamicSamplingContext, err := sdk.DynamicSamplingContextFromHeader([]byte(baggageHeader))
	if err != nil {
		// In case of errors, create a new non-frozen sampleing context
		dynamicSamplingContext = sdk.DynamicSamplingContext{Frozen: false}
	}

	ctx = context.WithValue(ctx, dynamicSamplingContextKey{}, dynamicSamplingContext)
	return ctx
}

// Fields returns a list of fields that will be used by the propagator.
//
// https://opentelemetry.io/docs/reference/specification/context/api-propagators/#fields
func (p sentryPropagator) Fields() []string {
	return []string{sdk.SentryTraceHeader, sdk.SentryBaggageHeader}
}

// Context keys to be used with context.WithValue(...) and ctx.Value(...)
type dynamicSamplingContextKey struct{}
type sentryTraceHeaderContextKey struct{}
type sentryTraceParentContextKey struct{}
type baggageContextKey struct{}
