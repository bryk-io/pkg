/*
Package sentry provides an OpenTelemetry exporter for Sentry.

An OpenTelemetry `Span` becomes a Sentry Transaction or Span. The
first Span sent through the Sentry `SpanProcessor` is a `Transaction`,
and any child Span gets attached to the first Transaction upon checking
the parent Span context. This is true for the OpenTelemetry root Span
and any top level Span in the system.

For example, a request sent from frontend to backend will create an
OpenTelemetry root Span with a corresponding Sentry Transaction. The
backend request will create a new Sentry Transaction for the OpenTelemetry
Span. The Sentry Transaction and Span are linked as a trace for navigation
and error tracking purposes.

More information:
https://docs.sentry.io/platforms/go/performance/instrumentation/opentelemetry
*/
package sentry
