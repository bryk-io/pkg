/*
Package otel provides OpenTelemetry instrumentation utilities for Go applications and libraries.

OpenTelemetry is an Observability framework and toolkit designed to create and
manage telemetry data such as traces, metrics, and logs. Crucially, OpenTelemetry
is vendor and tool-agnostic, meaning that it can be used with a broad variety
of Observability backends, including open source tools like Jaeger and Prometheus,
as well as commercial offerings.

For an application or component to emit useful telemetry data it needs to be
“instrumented”. In order to instrument your code in a way that is idiomatic for
OpenTelemetry you need to follow some conventions.

- OpenTelemetry is split into two parts: an API to instrument code with, and SDKs
that implement the API.

- If you’re instrumenting an app, you need to use the OpenTelemetry SDK for your
language. You’ll then use the SDK to initialize OpenTelemetry and the API to
instrument your code. This will emit telemetry from your app, and any library
you installed that also comes with instrumentation.

- If you’re instrumenting a library, only install the OpenTelemetry API package
for your language. Your library will not emit telemetry on its own. It will only
emit telemetry when it is part of an app that uses the OpenTelemetry SDK.
*/
package otel
