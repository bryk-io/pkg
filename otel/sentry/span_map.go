package sentry

import (
	sentryOtel "github.com/getsentry/sentry-go/otel"
)

// Map of Sentry spans to OpenTelemetry spans.
// Singleton instance.
var sentrySpanMap sentryOtel.SentrySpanMap

func init() {
	sentrySpanMap = sentryOtel.SentrySpanMap{}
	sentrySpanMap.Clear()
}
