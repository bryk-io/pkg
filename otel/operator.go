package otel

import (
	"context"
	"time"

	xlog "go.bryk.io/pkg/log"
	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/contrib/propagators/ot"
	"go.opentelemetry.io/otel"
	metricglobal "go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/propagation"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric/export"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	apitrace "go.opentelemetry.io/otel/trace"
)

// Operator instances provide a single point-of-control for observability
// requirements on a system, including: logs, metrics and traces.
type Operator struct {
	*Component                                      // Main component embedded
	log               xlog.Logger                   // Logger instance
	coreAttributes    Attributes                    // Resource attributes
	userAttributes    Attributes                    // User-provided additional attributes
	resource          *sdkresource.Resource         // OTEL resource definition
	exporter          sdktrace.SpanExporter         // Trace sync components
	metricExporter    sdkmetric.Exporter            // Metric sync components
	traceProvider     *sdktrace.TracerProvider      // Main traces provider
	metricProvider    *controller.Controller        // Main metrics provider
	propagator        propagation.TextMapPropagator // Default composite propagator
	tracerName        string                        // Name for the internal default tracer
	tracer            apitrace.Tracer               // Default internal tracer
	hostMetrics       bool                          // Capture standard host metrics
	runtimeMetrics    bool                          // Capture standard runtime metrics
	runtimeMetricsInt time.Duration                 // Runtime memory capture interval
	metricsPushInt    time.Duration                 // Push metrics interval
	prom              *prometheusSupport            // Prometheus support capabilities
	ctx               context.Context
}

// NewOperator create a new operator instance. This method is useful
// when monitoring several services, each with its own exporters or settings.
func NewOperator(options ...OperatorOption) (*Operator, error) {
	// Create instance and apply options.
	op := &Operator{
		log:               xlog.Discard(),    // discard logs
		coreAttributes:    coreAttributes(),  // standard env attributes
		userAttributes:    Attributes{},      // no custom attributes
		exporter:          new(noOpExporter), // discard traces and metrics
		runtimeMetricsInt: time.Duration(10) * time.Second,
		metricsPushInt:    time.Duration(5) * time.Second,
		ctx:               context.TODO(),
	}
	if err := op.setup(options...); err != nil {
		return nil, err
	}

	// Use the service name as tracer name. This will be reported as `otel.library.name`.
	op.tracerName, _ = op.coreAttributes.Get(lblSvcName).(string)

	// Attributes. Combine the default core attributes and the user provided data.
	// This attributes will be automatically used when logging messages and "inherited"
	// by all spans by adjusting the OTEL resource definition.
	attrs := join(op.coreAttributes, op.userAttributes)
	op.log = op.log.Sub(xlog.Fields(attrs))
	op.resource = sdkresource.NewSchemaless(attrs.Expand()...)

	// Initialize prometheus handler
	if op.prom != nil && op.prom.enabled {
		if err := op.prom.init(); err != nil {
			return nil, err
		}
	}

	// Prepare context propagation mechanisms.
	op.setupPropagators()

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
	}
	if op.metricProvider != nil {
		op.Component.MeterProvider = op.metricProvider
	}

	// Set OTEL error handler.
	otel.SetErrorHandler(errorHandler{ll: op.log})

	// Set OTEL globals.
	otel.SetTextMapPropagator(op.propagator) // propagator(s)
	otel.SetTracerProvider(op.traceProvider) // trace provider
	if op.metricProvider != nil {            // meter provider
		metricglobal.SetMeterProvider(op.metricProvider)
		op.captureStandardMetrics() // start collecting common metrics
	}
	return op, nil
}

// Shutdown notifies the operator of a pending halt to operations. All exporters
// will preform any cleanup or synchronization required while honoring all timeouts
// and cancellations contained in the provided context.
func (op *Operator) Shutdown(ctx context.Context) {
	_ = op.traceProvider.ForceFlush(ctx)
	_ = op.traceProvider.Shutdown(ctx)
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

// Apply provided configuration settings.
func (op *Operator) setup(options ...OperatorOption) error {
	for _, setting := range options {
		if err := setting(op); err != nil {
			return err
		}
	}
	return nil
}

// If you do not set a propagator, the default is to use a `NoOp` option, which
// means that the trace context will not be shared between multiple services. To
// avoid that, we set up a composite propagator that consist of a baggage propagator
// and trace context propagator. That way, both trace information (trace IDs, span
// IDs, etc.) and baggage will be propagated. B3 and OT are also enabled for
// backwards compatibility.
func (op *Operator) setupPropagators() {
	props := []propagation.TextMapPropagator{
		b3.New(),                   // b3
		ot.OT{},                    // ot-trace
		propagation.Baggage{},      // baggage
		propagation.TraceContext{}, // tracecontext
	}
	op.propagator = propagation.NewCompositeTextMapPropagator(props...)
}

// Create the metrics and traces provider elements for the operator instance.
func (op *Operator) setupProviders() error {
	// Build a span processing chain.
	spc := logSpans{
		log:  op.log,                                      // custom processor to generate logs
		Next: sdktrace.NewBatchSpanProcessor(op.exporter), // submit completed spans to the exporter
	}

	// Trace provider.
	// A trace provider is used to generate a tracer, and a tracer to create spans.
	// trace provider -> tracer -> span
	op.traceProvider = sdktrace.NewTracerProvider(
		sdktrace.WithResource(op.resource),             // adjust monitored resource
		sdktrace.WithSampler(sdktrace.AlwaysSample()),  // track all traces produced
		sdktrace.WithSpanLimits(sdktrace.SpanLimits{}), // use default span limits
		sdktrace.WithSpanProcessor(spc),                // set the span processing chain
	)

	// No metrics exporter was provided. Skip provider setup.
	if op.metricExporter == nil {
		return nil
	}

	// Metrics provider.
	op.metricProvider = controller.New(
		processor.NewFactory(
			simple.NewWithHistogramDistribution(),
			op.metricExporter,
		),
		controller.WithExporter(op.metricExporter),
		controller.WithCollectPeriod(op.metricsPushInt),
	)

	// Since we are using a push metrics controller, start the provider
	// automatically here.
	return op.metricProvider.Start(op.ctx)
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

// Simple error handler.
type errorHandler struct {
	ll xlog.Logger
}

// Handle any error deemed irremediable by the OpenTelemetry operator.
func (eh errorHandler) Handle(err error) {
	if err != nil {
		eh.ll.WithField("error.message", err.Error()).Warning("opentelemetry operator error")
	}
}
