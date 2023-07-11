package sentry

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sdk "github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel/propagation"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	apiTrace "go.opentelemetry.io/otel/trace"
)

// Reporter provides a Sentry integration for OpenTelemetry.
//
// An OpenTelemetry `Span` becomes a Sentry Transaction or Span. The
// first Span sent through the Sentry `SpanProcessor` is a `Transaction`,
// and any child Span gets attached to the first Transaction upon checking
// the parent Span context. This is true for the OpenTelemetry root Span
// and any top level Span in the system. For example, a request sent
// from frontend to backend will create an OpenTelemetry root Span with
// a corresponding Sentry Transaction. The backend request will create a
// new Sentry Transaction for the OpenTelemetry Span. The Sentry Transaction
// and Span are linked as a trace for navigation and error tracking purposes.
//
// More information:
// https://docs.sentry.io/platforms/go/performance/instrumentation/opentelemetry
type Reporter struct {
	hub    *sdk.Hub
	client *sdk.Client
	opts   *Options
}

// Options defines the configuration settings for the Sentry reporter.
type Options struct {
	// Project DSN provided by Sentry. If the DSN is not set, the client
	// is effectively disabled.
	DSN string `mapstructure:"dsn" yaml:"dsn" json:"dsn"`

	// Environment identifier used for transactions and events.
	Environment string `mapstructure:"environment" yaml:"environment" json:"environment"`

	// Release identifier. Must be unique across all services.
	// Usual format is:
	//   service-name@version+commit-hash
	//
	// Some Sentry features are built around releases, and, thus, reporting
	// events with a non-empty release improves the product experience.
	// See https://docs.sentry.io/product/releases/.
	Release string `mapstructure:"release" yaml:"release" json:"release"`

	// Whether to capture performance-related data.
	EnablePerformanceMonitoring bool `mapstructure:"performance_monitoring" yaml:"performance_monitoring" json:"performance_monitoring"` // nolint: lll

	// The sample rate for sampling traces in the range [0.0, 1.0].
	TracesSampleRate float64 `mapstructure:"traces_sample_rate" yaml:"traces_sample_rate" json:"traces_sample_rate"`

	// The sample rate for profiling traces in the range [0.0, 1.0].
	// Relative to `TracesSampleRate`; i.e., it is a ratio of profiled
	// traces out of all sampled traces.
	ProfilingSampleRate float64 `mapstructure:"profiling_sample_rate" yaml:"profiling_sample_rate" json:"profiling_sample_rate"` // nolint: lll

	// The maximum time to wait for events to be sent before shutdown.
	FlushTimeout time.Duration `mapstructure:"flush_timeout" yaml:"flush_timeout" json:"flush_timeout"`
}

// NewReporter returns a new Sentry reporter instance.
func NewReporter(opts *Options) (*Reporter, error) {
	err := sdk.Init(sdk.ClientOptions{
		Dsn:                opts.DSN,
		Debug:              false,
		Release:            opts.Release,
		Environment:        opts.Environment,
		EnableTracing:      opts.EnablePerformanceMonitoring,
		TracesSampleRate:   opts.TracesSampleRate,
		ProfilesSampleRate: opts.ProfilingSampleRate,
		AttachStacktrace:   true,
		Integrations: func(list []sdk.Integration) []sdk.Integration {
			var filtered []sdk.Integration
			for _, el := range list {
				// Remove default 'contextify' implementation
				if el.Name() == "ContextifyFrames" {
					continue
				}
				filtered = append(filtered, el)
			}
			// Add custom event processor
			return append(filtered, newEventProcessor())
		},
	})
	if err != nil {
		return nil, err
	}
	if opts.FlushTimeout == 0 {
		opts.FlushTimeout = 2 * time.Second // default flush timeout
	}
	return &Reporter{
		hub:    sdk.CurrentHub(),
		client: sdk.CurrentHub().Client(),
		opts:   opts,
	}, nil
}

// Context returns a new context instance with the current Sentry
// hub attached.
//
// https://docs.sentry.io/platforms/go/enriching-events/scopes
func (sr *Reporter) Context() context.Context {
	return sdk.SetHubOnContext(context.Background(), sdk.CurrentHub())
}

// Propagator returns a carrier than handles Sentry-specific
// details across service boundaries.
func (sr *Reporter) Propagator() propagation.TextMapPropagator {
	return newSentryPropagator()
}

// SpanProcessor handles the link between OpenTelemetry spans and
// Sentry transactions.
func (sr *Reporter) SpanProcessor() sdkTrace.SpanProcessor {
	return newSentrySpanProcessor(sr.hub, sr.opts.FlushTimeout)
}

// Event (s) can be used to register activity worth reporting; this
// usually describes an activity/tasks progression leading to a
// potential error condition.
//
// There are some special attributes you can add to events:
//   - event.kind: set to "default" if not provided
//   - event.category: set to "event" if not provided
//   - event.level: set to "info" if not provided.
//   - event.data: provides additional payload data, "nil" by default
//
// event.kind values:
//   - debug: typically a log message
//   - info: provide additional details to help identify the root cause of an issue
//   - error: error/warning occurring prior to a reported exception
//   - navigation: `event.data` must include key `from` and `to`
//   - http: http requests started from the app; `event.data` can include `http.request`
//   - query: describe and report database interactions
//   - user: describe user interactions
//
// https://develop.sentry.dev/sdk/event-payloads/breadcrumbs/#breadcrumb-types
func (sr *Reporter) Event(ctx apiTrace.SpanContext, message string, attributes ...map[string]interface{}) {
	if _, ok := sentrySpanMap.Get(ctx.SpanID()); !ok {
		return // nothing to do
	}

	// default values
	attrs := join(attributes...)
	kind := "default"
	level := "info"
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
		if js, err := json.Marshal(dt); err == nil {
			_ = json.Unmarshal(js, &data)
		}
	}

	// add event as breadcrumb to current scope
	sr.hub.Scope().AddBreadcrumb(&sdk.Breadcrumb{
		Type:      kind,
		Category:  category,
		Message:   message,
		Data:      data,
		Level:     getLevel(level),
		Timestamp: time.Now(),
	}, 100)
}

// ReportError should be used to report an error condition to Sentry.
// If the provided context contains a valid Sentry transaction, the
// error will be linked to it.
func (sr *Reporter) ReportError(ctx apiTrace.SpanContext, err error, attributes ...map[string]interface{}) {
	sp, ok := sentrySpanMap.Get(ctx.SpanID())
	if !ok {
		// capture exception without trying to link it to specific trace
		sr.hub.CaptureException(err)
		return
	}
	// link exception to trace context
	scope := sdk.NewScope()
	scope.SetContext("trace", traceContext(sp).Map())
	sr.client.CaptureException(err, &sdk.EventHint{OriginalException: err}, scope)

	// mark span as errored
	sp.Status = sdk.SpanStatusInternalError
}
