package sentry

// Based on the original: github.com/getsentry/sentry-go/otel

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sdk "github.com/getsentry/sentry-go"
	"go.bryk.io/pkg/errors"
	"go.bryk.io/pkg/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	semConv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

type sentrySpanProcessor struct {
	hub          *sdk.Hub
	flushTimeout time.Duration
	maxEvents    int
	errCodec     errors.Codec
}

// Singleton instance of the Sentry span processor.
// At the moment we do not support multiple instances.
var sentrySpanProcessorInstance *sentrySpanProcessor

func newSentrySpanProcessor(hub *sdk.Hub, ft time.Duration, maxEvents int) sdkTrace.SpanProcessor {
	if sentrySpanProcessorInstance != nil {
		return sentrySpanProcessorInstance
	}
	sdk.AddGlobalEventProcessor(linkTraceContextToErrorEvent)
	sentrySpanProcessorInstance := &sentrySpanProcessor{
		hub:          hub,
		flushTimeout: ft,
		maxEvents:    maxEvents,
		errCodec:     errors.CodecJSON(false), // ! make this configurable
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

	// Add span events as bredcrumbs
	for _, ev := range s.Events() {
		if bc := asBreadcrumb(ev); bc != nil {
			ssp.hub.Scope().AddBreadcrumb(bc, ssp.maxEvents)
		}
	}

	// Report span error(s), if any
	for _, ev := range s.Events() {
		if err := extractError(ev, ssp.errCodec); err != nil {
			ssp.hub.WithScope(func(scope *sdk.Scope) {
				scope.SetContext("trace", traceContext(sentrySpan).Map())
				ssp.hub.Client().CaptureException(err, &sdk.EventHint{OriginalException: err}, scope)
			})
		}
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
	return flushSpanProcessor(ssp.hub, ssp.flushTimeout)
}

func flushSpanProcessor(hub *sdk.Hub, ft time.Duration) error {
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
		attributes[kv.Key] = asString(kv.Value)
	}
	for _, kv := range s.Resource().Attributes() {
		resource[kv.Key] = asString(kv.Value)
	}
	otelCtx := map[string]interface{}{"Resource": resource}
	if len(attributes) > 0 {
		otelCtx["Attributes"] = attributes
	}
	transaction.SetContext("Open Telemetry", otelCtx)

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
		span.SetData(string(kv.Key), asString(kv.Value))
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

// Event (s) can be used to register activity worth reporting; this
// usually describes an activity/tasks progression leading to a
// potential error condition.
//
// There are some special attributes you can add to events:
//   - event.kind: set to "debug" if not provided
//   - event.category: set to "event" if not provided
//   - event.data: provides additional payload data, "nil" by default
//   - event.level: set to "debug" if not provided. Defines event's severity;
//     allowed values are, from highest to lowest: fatal, error, warning, info, and debug.
//
// event.kind values:
//   - debug: typically a log message
//   - info: provide additional details to help identify the root cause of an issue
//   - query: describe and report database interactions
//   - ui: a user interaction with your app's UI.
//   - user: describe user interactions
//   - transaction: describe a tracing event
//   - error: error/warning occurring prior to a reported exception
//   - navigation: `event.data` must include key `from` and `to`
//   - http: http requests started from the app; `event.data` can include attributes
//     such as: `method`, `url`, `status_code`, `reason`, `headers`, `cookies`
//
// https://develop.sentry.dev/sdk/event-payloads/breadcrumbs/#breadcrumb-types
func asBreadcrumb(ev sdkTrace.Event) *sdk.Breadcrumb {
	// don't report exceptions as events; these will be reported independently
	if ev.Name == "exception" {
		return nil
	}

	attrs := otel.Attributes{}
	attrs.Load(ev.Attributes)
	kind := "debug"
	level := "debug"
	category := "event"
	data := make(map[string]interface{})
	if k, ok := attrs["event.kind"]; ok {
		kind = fmt.Sprintf("%v", k)
	}
	if lvl, ok := attrs["event.level"]; ok {
		level = fmt.Sprintf("%v", lvl)
	}
	if cat, ok := attrs["event.category"]; ok {
		category = fmt.Sprintf("%v", cat)
	}
	if dt, ok := attrs["event.data"]; ok {
		if payload, ok := dt.(string); ok {
			_ = json.Unmarshal([]byte(payload), &data)
		}
	}
	return &sdk.Breadcrumb{
		Type:      kind,
		Category:  category,
		Message:   ev.Name,
		Data:      data,
		Level:     getLevel(level),
		Timestamp: ev.Time,
	}
}

// Look for "sentry.error" data in "exception" events to be restored
// and reported on Sentry.
func extractError(ev sdkTrace.Event, codec errors.Codec) error {
	// only process exception events
	if ev.Name != "exception" {
		return nil
	}

	// look for error details in event attributes
	attrs := otel.Attributes{}
	attrs.Load(ev.Attributes)
	errMSg := attrs.Get("exception.message") // simple error message
	errPayload := attrs.Get("sentry.error")  // original error report

	// no error report available but a simple error message;
	// return simple formatted error
	if errPayload == nil && errMSg != nil {
		return errors.WithStackAt(fmt.Errorf("%s", errMSg), 4)
	}

	// attempt to recover error instance from report
	payload, ok := errPayload.(string)
	if !ok {
		return nil // invalid error payload
	}
	ok, recErr := codec.Unmarshal([]byte(payload))
	if !ok {
		return nil // failed to decode error payload
	}
	return recErr // return recovered error
}
