package sentry

// Based on the original: github.com/getsentry/sentry-go/otel

import (
	"sync"

	sdk "github.com/getsentry/sentry-go"
	apiTrace "go.opentelemetry.io/otel/trace"
)

// Map of Sentry spans to OpenTelemetry spans.
// Singleton instance.
var sentrySpanMap spanMap

func init() {
	sentrySpanMap = spanMap{}
	sentrySpanMap.Clear()
}

// spanMap is a mapping between OpenTelemetry spans and Sentry spans.
// It helps Sentry span processor and propagator to keep track of
// unfinished Sentry spans and to establish parent-child links between
// spans.
type spanMap struct {
	db map[apiTrace.SpanID]*sdk.Span
	mu sync.RWMutex
}

func (sm *spanMap) Get(otelSpandID apiTrace.SpanID) (*sdk.Span, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	sp, ok := sm.db[otelSpandID]
	return sp, ok
}

func (sm *spanMap) Set(otelSpandID apiTrace.SpanID, sentrySpan *sdk.Span) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.db[otelSpandID] = sentrySpan
}

func (sm *spanMap) Delete(otelSpandID apiTrace.SpanID) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.db, otelSpandID)
}

func (sm *spanMap) Clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.db = make(map[apiTrace.SpanID]*sdk.Span)
}

func (sm *spanMap) Len() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.db)
}
