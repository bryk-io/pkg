package sdk

import (
	"context"
	"os"
	"time"

	"go.bryk.io/pkg/log"
	"go.bryk.io/pkg/otel"
	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	apiOtel "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdkMetric "go.opentelemetry.io/otel/sdk/metric"
	sdkResource "go.opentelemetry.io/otel/sdk/resource"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
)

// Instrumentation represents an OpenTelemetry instrumented application.
type Instrumentation struct {
	log               log.Logger                      // logger instance
	attrs             otel.Attributes                 // user-provided additional attributes
	resource          *sdkResource.Resource           // OTEL resource definition
	spanProcessors    []sdkTrace.SpanProcessor        // span processing chain
	traceExporter     sdkTrace.SpanExporter           // trace sink components
	metricExporter    sdkMetric.Reader                // metric sink components
	traceProvider     *sdkTrace.TracerProvider        // main traces provider
	meterProvider     *sdkMetric.MeterProvider        // main metrics provider
	propagator        propagation.TextMapPropagator   // default composite propagator
	hostMetrics       bool                            // capture standard host metrics
	runtimeMetrics    bool                            // capture standard runtime metrics
	runtimeMetricsInt time.Duration                   // runtime memory capture interval
	spanLimits        sdkTrace.SpanLimits             // default span limits
	props             []propagation.TextMapPropagator // list of individual text map propagators
	sampler           sdkTrace.Sampler                // trace sampler strategy used
	exemplars         bool                            // enable exemplar support
}

// Setup a new OpenTelemetry instrumented application.
func Setup(options ...Option) (*Instrumentation, error) {
	// Create app instance and apply options.
	app := &Instrumentation{
		log:               log.Discard(),           // discard logs
		attrs:             otel.Attributes{},       // no custom attributes
		traceExporter:     new(noOpExporter),       // discard traces and metrics
		sampler:           sdkTrace.AlwaysSample(), // track all traces by default
		spanLimits:        sdkTrace.NewSpanLimits(),
		runtimeMetricsInt: time.Duration(10) * time.Second,
		spanProcessors:    []sdkTrace.SpanProcessor{},
		props: []propagation.TextMapPropagator{
			propagation.Baggage{},      // headers: baggage
			propagation.TraceContext{}, // headers: traceparent, tracestate
		},
	}
	for _, setting := range options {
		setting(app)
	}

	// Setup OTEL resource and collect its attributes. The setup process
	// automatically collects environment information.
	var err error
	app.resource, err = setupResource(app.attrs)
	if err != nil {
		return nil, err
	}
	attrs := otel.Attributes{}
	attrs.Load(app.resource.Attributes())
	app.log = app.log.Sub(attrs)

	// Prepare context propagation mechanisms.
	// If you do not set a propagator the default is to use a `NoOp` option, which
	// means that the trace context will not be shared between multiple services. To
	// avoid that, we set up a composite propagator that consist of a baggage propagator
	// and trace context propagator.
	app.propagator = propagation.NewCompositeTextMapPropagator(app.props...)

	// Prepare traces and metrics providers.
	app.setupProviders()

	// Set OTEL globals.
	apiOtel.SetErrorHandler(errorHandler{ll: app.log}) // error handler
	apiOtel.SetTextMapPropagator(app.propagator)       // propagator(s)
	apiOtel.SetTracerProvider(app.traceProvider)       // trace provider
	if app.meterProvider != nil {
		apiOtel.SetMeterProvider(app.meterProvider) // metric provider
		app.captureStandardMetrics()                // start collecting common metrics
	}
	return app, nil
}

// Logger returns the application's logger instance.
func (app *Instrumentation) Logger() log.Logger {
	return app.log
}

// Flush immediately exports all spans that have not yet been exported for all
// the registered span processors and shut down them down. No further data will
// be captured or processed after this call.
func (app *Instrumentation) Flush(ctx context.Context) {
	// Stop trace provider and exporter
	_ = app.traceProvider.ForceFlush(ctx)
	_ = app.traceProvider.Shutdown(ctx)
	_ = app.traceExporter.Shutdown(ctx)

	// Stop metric provider
	if app.meterProvider != nil {
		_ = app.meterProvider.Shutdown(ctx)
	}
}

// Create the metrics and traces providers.
func (app *Instrumentation) setupProviders() {
	// Custom span processor to generate logs.
	spc := logSpans{
		log:  app.log,                                           // custom `SpanProcessor` to generate logs
		Next: sdkTrace.NewBatchSpanProcessor(app.traceExporter), // submit completed spans to the exporter
	}

	// Trace provider options.
	tpOpts := []sdkTrace.TracerProviderOption{sdkTrace.WithResource(app.resource), // adjust monitored resource
		sdkTrace.WithSampler(app.sampler),          // set sampling strategy
		sdkTrace.WithRawSpanLimits(app.spanLimits), // use default span limits
		sdkTrace.WithSpanProcessor(spc),            // set the span processing chain
	}
	for _, sp := range app.spanProcessors {
		tpOpts = append(tpOpts, sdkTrace.WithSpanProcessor(sp))
	}

	// Create and register the global trace provider.
	// A trace provider is used to generate a tracer, and a tracer to create spans.
	// trace provider -> tracer -> span
	app.traceProvider = sdkTrace.NewTracerProvider(tpOpts...)

	// If no metrics exporter was provided, skip provider setup.
	if app.metricExporter == nil {
		return
	}

	// Enable exemplar support.
	// https://github.com/open-telemetry/opentelemetry-go/blob/main/sdk/metric/internal/x/README.md#exemplars
	if app.exemplars {
		if err := os.Setenv("OTEL_GO_X_EXEMPLAR", "true"); err != nil {
			app.log.WithField("error.message", err.Error()).Warning("failed to enable exemplar support")
		}
		if err := os.Setenv("OTEL_METRICS_EXEMPLAR_FILTER", "always_on"); err != nil {
			app.log.WithField("error.message", err.Error()).Warning("failed to enable exemplar support")
		}
	}

	// Create meter provider instance using the provided "reader".
	metricProviderOpts := []sdkMetric.Option{
		sdkMetric.WithReader(app.metricExporter),
		sdkMetric.WithResource(app.resource),
	}
	app.meterProvider = sdkMetric.NewMeterProvider(metricProviderOpts...)
}

// Start collection of host and runtime metrics, if enabled.
func (app *Instrumentation) captureStandardMetrics() {
	// Nothing to do if no meter provider is available.
	if app.meterProvider == nil {
		return
	}

	// Capture host metrics.
	if app.hostMetrics {
		opts := []host.Option{
			host.WithMeterProvider(app.meterProvider),
		}
		if err := host.Start(opts...); err != nil {
			app.log.WithField("error.message", err.Error()).Warning("failed to start host metrics agent")
		}
	}

	// Capture runtime metrics.
	if app.runtimeMetrics {
		opts := []runtime.Option{
			runtime.WithMeterProvider(app.meterProvider),
			runtime.WithMinimumReadMemStatsInterval(app.runtimeMetricsInt),
		}
		if err := runtime.Start(opts...); err != nil {
			app.log.WithField("error.message", err.Error()).Warning("failed to start runtime metrics agent")
		}
	}
}

// Simple internal OTEL error handler.
type errorHandler struct {
	ll log.Logger
}

// Handle any error deemed irremediable by the OpenTelemetry operator.
func (eh errorHandler) Handle(err error) {
	if err != nil {
		eh.ll.WithField("error.message", err.Error()).Warning("opentelemetry operator error")
	}
}
