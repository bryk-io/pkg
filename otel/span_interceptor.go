package otel

import apiTrace "go.opentelemetry.io/otel/trace"

// SpanInterceptor allows to add custom logic to the span lifecycle.
type SpanInterceptor interface {
	// Event (s) can be used to register activity worth reporting; this
	// usually describes an activity/tasks progression leading to a
	// potential error condition.
	Event(ctx apiTrace.SpanContext, message string, attributes ...map[string]interface{})

	// ReportError should be used to report an error condition to an
	// external processor.
	ReportError(ctx apiTrace.SpanContext, err error, attributes ...map[string]interface{})
}

type noOpSpanInterceptor struct{}

func (si *noOpSpanInterceptor) Event(_ apiTrace.SpanContext, _ string, _ ...map[string]interface{}) {
	// do nothing
}

func (si *noOpSpanInterceptor) ReportError(_ apiTrace.SpanContext, _ error, _ ...map[string]interface{}) {
	// do nothing
}
