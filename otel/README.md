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

## How To

The instrumentation process consist of basically 2 steps.

1. Configure an "Operator" instance according to your project's requirements.

2. Use the "Operator" to create "Components" that can be used to monitor specific
   portions/modules of your application. Most of the time you might require a single
   module for your application as a whole.

## Setting up an Operator

An Operator instance can be created with the `NewOperator` method. The are several
options that can be used to customize its behavior. Some of the most commonly used
are:

- `WithServiceName`: Adjust the `service.name` attribute reported on all spans.
- `WithServiceVersion`: Adjust the `service.version` attribute reported on all spans.
- `WithResourceAttributes`: Add extra metadata attributes relevant to your service;
  these attributes will be inherited by all spans produced.
- `WithExporter`: All data collected by the operator needs to be send somewhere for
  processing, storage and consumption. This is referred to as an exporter. You can
  use any collector you want. For example a simple "standard output" exporter via `WithExporterStdout` or a more advanced [OTEL Collector](https://opentelemetry.io/docs/collector/) via the `WithExporterOTLP`. If no collector is specified the data
  will be discarded by default.

Review the documentation for details on several other options available.

## Instrumenting Your Application

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

### Using a Span

To instrument a partion of your code, for example a function being executed,
you use the operator instance to create a new span using the `Span` method.
You MUST always mark the Spans you create as done when appropriate using its
`End` method.

```go
task := op.Span(context.Background(), "my-task")
defer task.End()
```

You can optionally, but usually, use the `WithSpanAttributes` option to add
additional contextual metadata to a task. Attributes set on a root span are
automatically inherited by all its child spans.

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
task := op.Span(context.Background(), "my-task", WithSpanAttributes(fields))
defer task.End(nil)
```

During the execution of a given task you might wanna register relevant events
or errors to span. You can think of this as logs produced during the execution
of a task.

```go
task := op.Span(context.Background(), "my-task")
defer task.End(nil)

// My code is doing some intermediary task. If it fails, report it with
// proper severity level.
task.Event("performing X operation")
err := someOtherWorkToDo()
if err != nil {
  task.Error(log.Error, err)
}
task.End(err) // report the task as a failure
```

### Creating Child Spans

To create a span as child of an existing parent/root span you simply need to use
the original's context value when creating the new one.

```go
// Create a root span.
root := op.Span(context.Background(), "root-task")

// Create a child span using root's `Context`. This is usually
// done by passing the context to a different function.
child := op.Span(root.Context(), "root-task")
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
  child := op.Start(req.Context(), "handler operation")
  defer child.End(nil)
  // ... do all the operations required ...
}
```
