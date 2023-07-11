package sentry

import (
	"context"
	"strings"
	"time"

	sdk "github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel/attribute"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	semConv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

type sentrySpanProcessor struct {
	hub *sdk.Hub
	ft  time.Duration
}

// Singleton instance of the Sentry span processor.
// At the moment we do not support multiple instances.
var sentrySpanProcessorInstance *sentrySpanProcessor

func newSentrySpanProcessor(hub *sdk.Hub, ft time.Duration) sdkTrace.SpanProcessor {
	if sentrySpanProcessorInstance != nil {
		return sentrySpanProcessorInstance
	}
	sdk.AddGlobalEventProcessor(linkTraceContextToErrorEvent)
	sentrySpanProcessorInstance := &sentrySpanProcessor{
		hub: hub,
		ft:  ft,
	}
	return sentrySpanProcessorInstance
}

// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/sdk.md#onstart
func (ssp *sentrySpanProcessor) OnStart(parent context.Context, s sdkTrace.ReadWriteSpan) {
	// Get OTEL span details
	otelSpanContext := s.SpanContext()
	otelSpanID := otelSpanContext.SpanID()
	otelTraceID := otelSpanContext.TraceID()
	otelParentSpanID := s.Parent().SpanID()

	// Get Sentry parent span
	var sentryParentSpan *sdk.Span
	if otelSpanContext.IsValid() {
		sentryParentSpan, _ = sentrySpanMap.Get(otelParentSpanID)
	}

	// Add child span
	if sentryParentSpan != nil {
		span := sentryParentSpan.StartChild(s.Name())
		span.SpanID = sdk.SpanID(otelSpanID)
		span.StartTime = s.StartTime()
		span.Status = sdk.SpanStatusOK
		sentrySpanMap.Set(otelSpanID, span)
		return
	}

	// Create new parent transaction
	traceParentContext := getTraceParentContext(parent)
	transaction := sdk.StartTransaction(
		parent,
		s.Name(),
		sdk.WithSpanSampled(traceParentContext.Sampled),
	)
	transaction.SpanID = sdk.SpanID(otelSpanID)
	transaction.TraceID = sdk.TraceID(otelTraceID)
	transaction.ParentSpanID = sdk.SpanID(otelParentSpanID)
	transaction.StartTime = s.StartTime()
	transaction.Status = sdk.SpanStatusOK
	if dynamicSamplingContext, valid := parent.Value(dynamicSamplingContextKey{}).(sdk.DynamicSamplingContext); valid {
		transaction.SetDynamicSamplingContext(dynamicSamplingContext)
	}
	sentrySpanMap.Set(otelSpanID, transaction)
}

// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/sdk.md#onendspan
func (ssp *sentrySpanProcessor) OnEnd(s sdkTrace.ReadOnlySpan) {
	// retrieve Sentry span
	otelSpanID := s.SpanContext().SpanID()
	sentrySpan, ok := sentrySpanMap.Get(otelSpanID)
	if !ok || sentrySpan == nil {
		return
	}

	// do not handle Sentry request spans; internally used to report spans
	if isSentryRequestSpan(sentrySpan.Context(), s) {
		sentrySpanMap.Delete(otelSpanID)
		return
	}

	// attach OTEL data to Sentry span
	if sentrySpan.IsTransaction() {
		updateTransactionWithOtelData(sentrySpan, s, ssp.hub)
	} else {
		updateSpanWithOtelData(sentrySpan, s)
	}

	// capture span
	sentrySpan.Status = getStatus(s)
	sentrySpan.EndTime = s.EndTime()
	sentrySpan.Finish()
	sentrySpanMap.Delete(otelSpanID)
}

// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/sdk.md#shutdown-1
func (ssp *sentrySpanProcessor) Shutdown(ctx context.Context) error {
	// ~ per the spec: "shutdown MUST include the effects of ForceFlush"
	sentrySpanMap.Clear()
	return ssp.ForceFlush(ctx)
}

// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/sdk.md#forceflush-1
func (ssp *sentrySpanProcessor) ForceFlush(ctx context.Context) error {
	return flushSpanProcessor(ssp.hub, ssp.ft)
}

func flushSpanProcessor(hub *sdk.Hub, ft time.Duration) error {
	// hub := sdk.GetHubFromContext(ctx)
	defer hub.Flush(ft)
	return nil
}

func getTraceParentContext(ctx context.Context) sdk.TraceParentContext {
	traceParentContext, ok := ctx.Value(sentryTraceParentContextKey{}).(sdk.TraceParentContext)
	if !ok {
		traceParentContext.Sampled = sdk.SampledUndefined
	}
	return traceParentContext
}

func updateTransactionWithOtelData(transaction *sdk.Span, s sdkTrace.ReadOnlySpan, hub *sdk.Hub) {
	// parse OTEL standard attributes
	attributes := map[attribute.Key]string{}
	resource := map[attribute.Key]string{}
	for _, kv := range s.Attributes() {
		attributes[kv.Key] = kv.Value.AsString()
	}
	for _, kv := range s.Resource().Attributes() {
		resource[kv.Key] = kv.Value.AsString()
	}
	transaction.SetContext("Open Telemetry", map[string]interface{}{
		"Attributes": attributes,
		"Resource":   resource,
	})

	// get span attributes
	spanAttributes := parseSpanAttributes(s)
	transaction.Name = spanAttributes.Description
	transaction.Op = spanAttributes.Op
	transaction.Source = spanAttributes.Source
	if spanAttributes.User != nil {
		hub.Scope().SetUser(*spanAttributes.User)
	}
}

func updateSpanWithOtelData(span *sdk.Span, s sdkTrace.ReadOnlySpan) {
	spanAttributes := parseSpanAttributes(s)
	span.Op = spanAttributes.Op
	span.Description = spanAttributes.Description
	span.SetData("otel.kind", s.SpanKind().String())
	for _, kv := range s.Attributes() {
		span.SetData(string(kv.Key), kv.Value.AsString())
	}
}

// Sentry event processor that attaches trace information to the error event.
//
// Caveat: `hint.Context` should contain a valid context populated by
// OpenTelemetry's span context.
func linkTraceContextToErrorEvent(event *sdk.Event, hint *sdk.EventHint) *sdk.Event {
	if hint == nil || hint.Context == nil {
		return event
	}
	// compare with the (unexported) sentry.transactionType
	if event.Type == "transaction" {
		return event
	}
	otelSpanContext := trace.SpanContextFromContext(hint.Context)
	var sentrySpan *sdk.Span
	if otelSpanContext.IsValid() {
		sentrySpan, _ = sentrySpanMap.Get(otelSpanContext.SpanID())
	}
	if sentrySpan == nil {
		return event
	}

	traceContext, found := event.Contexts["trace"]
	if !found {
		event.Contexts["trace"] = make(map[string]interface{})
		traceContext = event.Contexts["trace"]
	}
	traceContext["trace_id"] = sentrySpan.TraceID.String()
	traceContext["span_id"] = sentrySpan.SpanID.String()
	traceContext["parent_span_id"] = sentrySpan.ParentSpanID.String()
	return event
}

func isSentryRequestSpan(ctx context.Context, s sdkTrace.ReadOnlySpan) bool {
	// Look for the `http.url` attribute
	for _, attribute := range s.Attributes() {
		if attribute.Key == semConv.HTTPURLKey {
			return isSentryRequestURL(ctx, attribute.Value.AsString())
		}
	}
	return false
}

func isSentryRequestURL(ctx context.Context, url string) bool {
	// get current hub
	hub := sdk.GetHubFromContext(ctx)
	if hub == nil {
		hub = sdk.CurrentHub()
		if hub == nil {
			return false
		}
	}

	// get client
	client := hub.Client()
	if client == nil {
		return false
	}

	// get DSN
	dsn, err := sdk.NewDsn(client.Options().Dsn)
	if err != nil {
		return false
	}
	return strings.Contains(url, dsn.GetAPIURL().String())
}
