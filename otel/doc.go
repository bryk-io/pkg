/*
Package otel provide utilities to instrument applications using OpenTelemetry.

Proper instrumentation is crucial to monitor system performance, behavior, and
to detect problems, regressions and bugs. This practice is usually referred to
as observability. The 3 pillars of observability are: logs, metrics and traces.
All these details must be generated, collected and analyzed in a logical and
consistent manner.

In software, observability typically refers to telemetry produced by services
and is often divided into three major verticals:

- Tracing: Provides insight into the full lifecycles (i.e. traces) of requests
to the system, allowing you to pinpoint failures and performance issues.

- Metrics: Provide quantitative information about processes running inside the
system, including counters, gauges, and histograms.

- Logging: Provides insight into application-specific messages emitted by processes.

These verticals are tightly interconnected. Metrics can be used to pinpoint, for
example, a subset of misbehaving traces. Logs associated with those traces could
help to find the root cause of this behavior. And then new metrics can be configured,
based on this discovery, to catch this issue earlier next time. Other verticals
exist (continuous profiling, production debugging, etc.), however traces, metrics,
and logs are the three most well adopted across the industry.

These package simplify proper instrumentation of systems by integrating all 3
data sources into a single interface with the following characteristics:
1) easy to set up; 2) easy to use; 3) consistent (and mostly automatic) behavior.

	options := []OperatorOption{
		WithServiceName("operator-testing"),
		WithServiceVersion("0.1.0"),
		WithLogger(xlog.WithZero(true)),
		WithAttributes(Attributes{
			"custom.field":     "bar",
			"only.for.testing": true,
		}),
	}
	op, err := NewOperator(options...)
	if err != nil {
		panic(err)
	}

	// The operator instance can then be accessed and used as follows
	sp := op.Start(context.Background(), "task", SpanKindServer)
	defer sp.End()
	fmt.Println(sp.ID())

Traces

Instrumentation is collected at transaction level. A transaction is a unit of
work relevant enough to be registered, measured for performance and observed
for events, behavior and correctness. In the observability context a transaction
is named a "Span". A root span can be the source for additional spans (child),
in distributed systems these child spans can even be performed by remote components.
To properly preserve this parent -> child relationship, certain information
about the span state (i.e. its context) must be propagated when communication
occurs between different services and components.

A span is the building block of a trace and is a named, timed operation that
represents a piece of the workflow in the distributed system. Multiple spans
are pieced together to create a trace. Traces are often viewed as a "tree" of
spans that reflects the time that each span started and completed. It also shows
you the relationship between spans. A trace starts with a root span where the
request starts. This root span can have one or more child spans, and each one
of those child spans can have child spans.

OpenTelemetry offers two data types, Attribute and Event, which are incredibly
valuable as they help to contextualize what happens during the execution measured
by a single span.

Attributes are key-value pairs that can be freely added to a span to help in
analysis of the trace data.

Events are time-stamped strings that can be attached to a span, with an optional
set of Attributes that provide further description.

Spans in OpenTelemetry are generated by the Tracer, an object that tracks the
currently active span and allows you to create (or activate) new spans. Tracer
objects are configured with Propagator objects that support transferring the
context across process boundaries.

As spans are created and completed, the Tracer dispatches them to the OpenTelemetry
SDK’s Exporter, which is responsible for sending your spans to a backend system
for analysis.

Metrics

Metrics allow capturing measurements about the execution of a computer program
at run time. In OpenTelemetry, the Metrics API provides six metric instruments.
These instruments are created and defined through calls to a Meter API, which is
the user-facing entry point to the SDK. Each instrument supports a single function,
named to help convey the instrument's semantics, and is either synchronous or
asynchronous.

Exporter

The trace and metric data that your service or its dependencies emit are of
limited use unless you can actually collect that data somewhere for analysis
and alerting. The OpenTelemetry component responsible for batching and transporting
telemetry data to a backend system is known as an exporter.

Generally, instrumentation can be done at three different points: at the service
level, at its library dependencies, and at its platform dependencies. Integrating
at the service level involves declaring a dependency in your code on the appropriate
OpenTelemetry package and deploying it with your code. Library dependencies are
similar, except that libraries will generally only declare a dependency on the
OpenTelemetry API. Platform dependencies are the pieces of software you run to
provide utilities to your service. These will deploy their own copy of OpenTelemetry,
independent of your actions, but will also emit trace context that your service will
find useful.

Context

The ability to correlate events across service boundaries is one of the principle
concepts behind distributed tracing. To find these correlations, components in a
distributed system need to be able to collect, store, and transfer metadata referred
to as context. This context is divided into two types, span context and correlation
context.

Span context represents the data required for moving trace information across service
boundaries.

Correlation context carries user-defined properties. These properties are typically
data that you would like to eventually aggregate for correlation analysis or used to
filter your trace data, such as a customer identifier, process hostname, data region
or other telemetry that provide application-specific performance insights.

More information:
https://opentelemetry.lightstep.com/
*/
package otel
