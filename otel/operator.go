package otel

import (
	"context"
	"time"

	"go.bryk.io/pkg/log"
	apiErrors "go.bryk.io/pkg/otel/errors"
	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	metricGlobal "go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/propagation"
	metricController "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	metricExport "go.opentelemetry.io/otel/sdk/metric/export"
	metricProcessor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	metricSelector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
	sdkResource "go.opentelemetry.io/otel/sdk/resource"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	semConv "go.opentelemetry.io/otel/semconv/v1.7.0"
	apiTrace "go.opentelemetry.io/otel/trace"
)

// Operator provides a single point-of-control for observability
// requirements on a system, including: logs, metrics and traces.
type Operator struct {
	*Component                                        // main embedded component
	log               log.Logger                      // logger instance
	reporter          apiErrors.Reporter              // error reporter
	coreAttributes    Attributes                      // resource attributes
	userAttributes    Attributes                      // user-provided additional attributes
	resource          *sdkResource.Resource           // OTEL resource definition
	exporter          sdkTrace.SpanExporter           // trace sync components
	metricExporter    metricExport.Exporter           // metric sync components
	traceProvider     *sdkTrace.TracerProvider        // main traces provider
	metricProvider    *metricController.Controller    // main metrics provider
	propagator        propagation.TextMapPropagator   // default composite propagator
	tracerName        string                          // name for the internal default tracer
	tracer            apiTrace.Tracer                 // default internal tracer
	hostMetrics       bool                            // capture standard host metrics
	runtimeMetrics    bool                            // capture standard runtime metrics
	runtimeMetricsInt time.Duration                   // runtime memory capture interval
	metricsPushInt    time.Duration                   // push metrics interval
	spanLimits        sdkTrace.SpanLimits             // default span limits
	props             []propagation.TextMapPropagator // list of individual text map propagators
	sampler           sdkTrace.Sampler                // trace sampler strategy used
}

// NewOperator creates a new operator instance. Operators can be used
// to monitor individual services, each with its own exporters or settings.
func NewOperator(options ...OperatorOption) (*Operator, error) {
	// Create instance and apply options.
	op := &Operator{
		log:               log.Discard(),            // discard logs
		reporter:          apiErrors.NoOpReporter(), // discard reported errors
		coreAttributes:    coreAttributes(),         // standard env attributes
		userAttributes:    Attributes{},             // no custom attributes
		exporter:          new(noOpExporter),        // discard traces and metrics
		tracerName:        "go.bryk.io/pkg/otel",    // default value for `otel.library.name`
		sampler:           sdkTrace.AlwaysSample(),  // track all traces by default
		spanLimits:        sdkTrace.NewSpanLimits(),
		runtimeMetricsInt: time.Duration(10) * time.Second,
		metricsPushInt:    time.Duration(5) * time.Second,
		props: []propagation.TextMapPropagator{
			propagation.Baggage{},      // baggage
			propagation.TraceContext{}, // tracecontext
		},
	}
	if err := op.setup(options...); err != nil {
		return nil, err
	}

	// Attributes. Combine the default core attributes and the user provided data.
	// This attributes will be automatically used when logging messages and "inherited"
	// by all spans by adjusting the OTEL resource definition.
	attrs := join(op.coreAttributes, op.userAttributes)
	op.log = op.log.Sub(log.Fields(attrs))
	op.resource = sdkResource.NewWithAttributes(semConv.SchemaURL, attrs.Expand()...)

	// Prepare context propagation mechanisms.
	// If you do not set a propagator the default is to use a `NoOp` option, which
	// means that the trace context will not be shared between multiple services. To
	// avoid that, we set up a composite propagator that consist of a baggage propagator
	// and trace context propagator.
	op.propagator = propagation.NewCompositeTextMapPropagator(op.props...)

	// Prepare traces and metrics providers.
	if err := op.setupProviders(); err != nil {
		return nil, err
	}

	// Default internal tracer.
	op.tracer = op.traceProvider.Tracer(op.tracerName)

	// Create the default "main" component.
	op.Component = &Component{
		ot:         op.tracer,
		propagator: op.propagator,
		attrs:      Attributes{},
		Logger:     op.log,
		reporter:   op.reporter,
	}
	if op.metricProvider != nil {
		op.Component.MeterProvider = op.metricProvider
	}

	// Set internal OTEL error handler.
	otel.SetErrorHandler(errorHandler{ll: op.log})

	// Set OTEL globals.
	otel.SetTextMapPropagator(op.propagator) // propagator(s)
	otel.SetTracerProvider(op.traceProvider) // trace provider
	if op.metricProvider != nil {            // meter provider
		metricGlobal.SetMeterProvider(op.metricProvider)
		op.captureStandardMetrics() // start collecting common metrics
	}
	return op, nil
}

// Shutdown notifies the operator of a pending halt to operations. All exporters
// will preform any cleanup or synchronization required while honoring all timeouts
// and cancellations contained in the provided context.
func (op *Operator) Shutdown(ctx context.Context) {
	// Stop error reporter
	_ = op.reporter.Flush(5 * time.Second)

	// Stop trace provider and exporter
	_ = op.traceProvider.ForceFlush(ctx)
	_ = op.traceProvider.Shutdown(ctx)
	_ = op.exporter.Shutdown(ctx)

	// Stop metric provider
	if op.metricProvider != nil {
		_ = op.metricProvider.Stop(ctx)
	}
}

// MainComponent returns an access handler for the main observability component
// associated directly with the operator instance. This is useful when a certain
// application element requires access to the instrumentation API, but we want to
// limit its access to the operator handler.
func (op *Operator) MainComponent() *Component {
	return op.Component
}

// ErrorReporter returns the error reporting instance setup with the operator.
func (op *Operator) ErrorReporter() apiErrors.Reporter {
	return op.reporter
}

// Apply provided configuration settings.
func (op *Operator) setup(options ...OperatorOption) error {
	for _, setting := range options {
		if err := setting(op); err != nil {
			return err
		}
	}
	return nil
}

// Create the metrics and traces provider elements for the operator instance.
func (op *Operator) setupProviders() error {
	// Build a span processing chain.
	spc := logSpans{
		log:  op.log,                                      // custom `SpanProcessor` to generate logs
		Next: sdkTrace.NewBatchSpanProcessor(op.exporter), // submit completed spans to the exporter
	}

	// Trace provider.
	// A trace provider is used to generate a tracer, and a tracer to create spans.
	// trace provider -> tracer -> span
	op.traceProvider = sdkTrace.NewTracerProvider(
		sdkTrace.WithResource(op.resource),        // adjust monitored resource
		sdkTrace.WithSampler(op.sampler),          // set sampling strategy
		sdkTrace.WithRawSpanLimits(op.spanLimits), // use default span limits
		sdkTrace.WithSpanProcessor(spc),           // set the span processing chain
	)

	// If no metrics exporter was provided, skip provider setup.
	if op.metricExporter == nil {
		return nil
	}

	// Metrics provider.
	op.metricProvider = metricController.New(
		metricProcessor.NewFactory(
			metricSelector.NewWithHistogramDistribution(),
			op.metricExporter,
		),
		metricController.WithExporter(op.metricExporter),
		metricController.WithCollectPeriod(op.metricsPushInt),
	)

	// Since we are using a push metrics controller, start the provider
	// automatically here.
	return op.metricProvider.Start(context.Background())
}

// Start collection of host and runtime metrics, if enabled.
func (op *Operator) captureStandardMetrics() {
	// Capture host metrics.
	if op.hostMetrics {
		opts := []host.Option{
			host.WithMeterProvider(op.metricProvider),
		}
		if err := host.Start(opts...); err != nil {
			op.log.WithField("error.message", err.Error()).Warning("failed to start host metrics agent")
		}
	}

	// Capture runtime metrics.
	if op.runtimeMetrics {
		opts := []runtime.Option{
			runtime.WithMeterProvider(op.metricProvider),
			runtime.WithMinimumReadMemStatsInterval(op.runtimeMetricsInt),
		}
		if err := runtime.Start(opts...); err != nil {
			op.log.WithField("error.message", err.Error()).Warning("failed to start runtime metrics agent")
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
