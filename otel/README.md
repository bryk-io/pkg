# OpenTelemetry Instrumentation

Proper instrumentation is crucial to monitor system performance, behavior, and
to detect problems, regressions and bugs. This practice is usually referred to
as observability.

In software, observability typically refers to telemetry produced by services
and is often divided into three major verticals that must be generated, collected
and analyzed in a logical and consistent manner.

- __Tracing__: Provides insight into the full lifecycles (i.e. traces) of requests
to the system, allowing you to pinpoint failures and performance issues.

- __Metrics__: Provide quantitative information about processes running inside the
system, including counters, gauges, and histograms.

- __Logging__: Provides insight into application-specific messages emitted by processes.

OpenTelemetry is a collection of tools, APIs, and SDKs. With the purpose to
instrument code and generate, collect, and export telemetry data (metrics,
logs, and traces) to help analyze software’s performance and behavior; and doing
it in a way that is "vendor-agnostic".

These package simplify proper instrumentation of systems by integrating all 3
data sources into a single interface with the following characteristics:

- Easy to set up
- Easy to use
- Consistent (and mostly automatic) behavior

## SDK and API

For an application or component to emit useful telemetry data it needs to be
“instrumented”. In order to instrument your code in a way that is idiomatic for
OpenTelemetry you need to follow some conventions.

- OpenTelemetry is split into two parts: an API to instrument code with, and SDKs
  that implement the API.

- If you’re instrumenting a library, only install the OpenTelemetry API package
  for your language. Your library will not emit telemetry on its own. It will only
  emit telemetry when it is part of an app that uses the OpenTelemetry SDK.

- If you’re instrumenting an app, you need to use the OpenTelemetry SDK for your
  language. You’ll then use the SDK to initialize OpenTelemetry and the API to
  instrument your code. This will emit telemetry from your app, and any library
  you installed that also comes with instrumentation.

## 1. Instrumenting Your Application

The simplest way to reason about instrumenting your application is at the
transaction/event level. A transaction is a unit of work relevant enough to
be registered, measured for performance and observed for events, behavior and
correctness. In the observability context a transaction is named a "Span". A
root span can be the source for additional (child) spans, in distributed systems
these child spans can even be performed by remote components.

To properly preserve this parent -> child relationship, certain information
about the span state (i.e. its context) must be propagated when communication
occurs between/across different services and components.

A span is the building block of a trace and is a named, timed operation that
represents a piece of the workflow in the distributed system. Multiple spans
are pieced together to create a trace. Traces are often viewed as a "tree" of
spans that reflects the time that each span started and completed. It also shows
you the relationship between spans. A trace starts with a root span where the
request starts. This root span can have one or more child spans, and each one
of those child spans can have child spans.

To instrument a partion of your code, for example a function being executed,
you use the API package to create a new span using the `Start` method.
You MUST always mark the Spans you create as done when appropriate using its
`End` method.

```go
task := api.Start(context.Background(), "my-task")
defer task.End(nil)
```

You can optionally, but usually, use the `WithAttributes` option to add
additional contextual metadata to a task.

```go
// Operation fields
fields := Attributes{
  "user.id":           "testing-user",
  "user.email":        "rick@c137.com",
  "task.value.string": "bar",
  "task.value.int":    120,
  "task.value.bool":   true,
  "task.value.float":  1.456,
  "task.value.list":   []string{"foo", "bar"},
}
task := api.Span(context.Background(), "my-task", api.WithAttributes(fields))
defer task.End(nil)
```

During the execution of a given task you might wanna register relevant events.
You can think of this as logs produced during the execution of a task.

```go
// My code is doing some intermediary task. If it fails, report it with
// proper severity level.
task.Event("performing X operation")
task.End(someFunctionReturningErrorOrNil())
```

To create a span as child of an existing parent/root span you simply need to use
the original's context value when creating the new one.

```go
// Create a root span.
root := api.Span(context.Background(), "root-task")

// Create a child span using root's `Context`. This is usually
// done by passing the context to a different function.
child := api.Span(root.Context(), "child-task")
// do some important stuff / external function returns
// and mark child as done.
child.End(nil)

// Mark root as done.
root.End(nil)
```

This also works when passing around span's `Context` across service boundaries;
for example, inside an HTTP handler.

```go
func(res http.ResponseWriter, req *http.Request) {
  // Start a new span directly on the handler function. The context in the HTTP
  // request contains the span data if generated by an instrumented HTTP client.
  child := api.Start(req.Context(), "handler operation")
  defer child.End(nil)
  // ... do all the operations required ...
}
```

## 2. Enabling Instrumentation

Even if some portion of code is instrumented, no data will be produced and collected
unless the instrumention is enabled/activated using a specific implementation. This
can be easily done using the `sdk` package.

The are several options that can be used to customize the generated telemetry data.
Some of the most commonly used are:

- `WithServiceName`: Adjust the `service.name` attribute reported on all spans.
- `WithServiceVersion`: Adjust the `service.version` attribute reported on all spans.
- `WithResourceAttributes`: Add extra metadata attributes relevant to your service;
  these attributes will be inherited by all spans produced.
- `WithExporter`: All data collected by the operator needs to be send somewhere for
  processing, storage and consumption. This is referred to as an exporter. You can
  use any collector you want. For example a simple "standard output" exporter via
  `WithExporterStdout` or a more advanced [OTEL Collector](https://opentelemetry.io/docs/collector/)
  via the `WithExporterOTLP`. If no collector is specified the data will be discarded
  by default.

Review the documentation for details on several other options available.

```go
// instrumentation options
settings := []sdk.Option{
  sdk.WithServiceName("my-service"),
  sdk.WithServiceVersion("0.1.0"),
  sdk.WithSpanLimits(sdkTrace.NewSpanLimits()),
  sdk.WithSampler(sdkTrace.ParentBased(sdkTrace.TraceIDRatioBased(0.9))),
  sdk.WithExporter(traceExp),
  sdk.WithMetricReader(sdkMetric.NewPeriodicReader(metricExp)),
  sdk.WithHostMetrics(),
  sdk.WithRuntimeMetrics(time.Duration(10) * time.Second),
  sdk.WithResourceAttributes(otel.Attributes{"resource.level.field": "bar"}),
  sdk.WithBaseLogger(log.WithZero(log.ZeroOptions{
    PrettyPrint: true,
    ErrorField:  "error.message",
  })),
}

// enable/activate the instrumentation setup
telemetry, _ := sdk.Setup(settings...)
defer telemetry.Flush(context.Background())
```
