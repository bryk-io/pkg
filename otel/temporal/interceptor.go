package temporal

import (
	"context"
	"fmt"

	apiTrace "go.bryk.io/pkg/otel/api"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/log"
)

// based on the original: go.temporal.io/sdk/contrib/opentelemetry

const (
	// default HTTP header key used to transmit span context data.
	defaultHeaderKey = "_tracer-data"
)

type monitor struct {
	at                     apiTrace.Tracer
	mp                     propagation.TextMapPropagator
	interceptor.BaseTracer // embed base implementation
}

// NewTracingInterceptor creates an interceptor for setting on client
// options that implements OpenTelemetry tracing for workflows.
//
//	client.Options{
//		Interceptors: []interceptor.ClientInterceptor{
//			NewTracingInterceptor(),
//		},
//	}
func NewTracingInterceptor() interceptor.Interceptor {
	tracer := &monitor{
		at: apiTrace.GetTracer(),
		mp: otel.GetTextMapPropagator(),
	}
	return interceptor.NewTracingInterceptor(tracer)
}

func (m *monitor) Options() interceptor.TracerOptions {
	return interceptor.TracerOptions{
		SpanContextKey:          spanContextKey{},
		HeaderKey:               defaultHeaderKey,
		DisableSignalTracing:    true,
		DisableQueryTracing:     true,
		AllowInvalidParentSpans: false,
	}
}

func (m *monitor) GetLogger(logger log.Logger, ref interceptor.TracerSpanRef) log.Logger {
	span, ok := ref.(*tracerSpan)
	if !ok {
		return logger
	}
	spCtx := span.Unwrap().SpanContext()
	logger = log.With(logger,
		"TraceID", spCtx.TraceID(),
		"SpanID", spCtx.SpanID(),
	)
	return logger
}

func (m *monitor) UnmarshalSpan(kv map[string]string) (interceptor.TracerSpanRef, error) {
	ctx := trace.SpanContextFromContext(m.mp.Extract(context.Background(), textMapCarrier(kv)))
	if !ctx.IsValid() {
		return nil, fmt.Errorf("failed extracting OpenTelemetry span from map")
	}
	return &tracerSpanRef{SpanContext: ctx}, nil
}

func (m *monitor) MarshalSpan(span interceptor.TracerSpan) (map[string]string, error) {
	tp, ok := span.(*tracerSpan)
	if !ok {
		return nil, fmt.Errorf("invalid span type")
	}
	data := textMapCarrier{}
	m.mp.Inject(trace.ContextWithSpan(context.Background(), tp.Unwrap()), data)
	return map[string]string(data), nil
}

func (m *monitor) SpanFromContext(ctx context.Context) interceptor.TracerSpan {
	span := apiTrace.SpanFromContext(ctx)
	if !span.Unwrap().SpanContext().IsValid() {
		return nil
	}
	return &tracerSpan{Span: span}
}

func (m *monitor) ContextWithSpan(ctx context.Context, span interceptor.TracerSpan) context.Context {
	tp, ok := span.(*tracerSpan)
	if !ok {
		return ctx
	}
	return trace.ContextWithSpan(ctx, tp.Unwrap())
}

func (m *monitor) StartSpan(opts *interceptor.TracerStartSpanOptions) (interceptor.TracerSpan, error) {
	// Create context with parent
	var parent trace.SpanContext
	switch optParent := opts.Parent.(type) {
	case nil:
	case *tracerSpan:
		parent = optParent.Unwrap().SpanContext()
	case *tracerSpanRef:
		parent = optParent.SpanContext
	default:
		return nil, fmt.Errorf("unrecognized parent type %T", optParent)
	}
	ctx := context.Background()
	if parent.IsValid() {
		ctx = trace.ContextWithSpanContext(ctx, parent)
	}

	// Create span
	spanName := opts.Operation + ":" + opts.Name
	span := m.at.Start(ctx, spanName, apiTrace.WithStartOptions(trace.WithTimestamp(opts.Time)))

	// Set tags
	if len(opts.Tags) > 0 {
		tags := map[string]interface{}{}
		for k, v := range opts.Tags {
			tags[k] = v
		}
		span.SetAttributes(tags)
	}
	return &tracerSpan{Span: span}, nil
}
