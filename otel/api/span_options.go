package api

import (
	"go.bryk.io/pkg/otel"
	apiTrace "go.opentelemetry.io/otel/trace"
)

// SpanOption allow adjusting span settings at the moment of creation.
type SpanOption func(conf *spanConfig)

// Internal representation of a span configuration.
type spanConfig struct {
	kind  SpanKind
	opts  []apiTrace.SpanStartOption
	attrs otel.Attributes
}

// WithSpanKind adjust the `span.kind` value for the created span.
// When no value is provided, `unspecified` is used by default.
func WithSpanKind(kind SpanKind) SpanOption {
	return func(sc *spanConfig) {
		sc.kind = kind
	}
}

// WithAttributes adds additional metadata related to a specific
// task. These attributes are used to describe the work a Span represents.
// If multiple of these options are passed the attributes of each
// successive option will extend/override any previously set value.
func WithAttributes(attrs map[string]any) SpanOption {
	return func(sc *spanConfig) {
		sc.attrs = otel.Attributes(attrs)
	}
}

// WithStartOptions allows passing additional options to the span
// creation process.
func WithStartOptions(opts ...apiTrace.SpanStartOption) SpanOption {
	return func(sc *spanConfig) {
		sc.opts = append(sc.opts, opts...)
	}
}

func defaultSpanConf() *spanConfig {
	return &spanConfig{kind: SpanKindUnspecified}
}

func (sc *spanConfig) startOpts() (opts []apiTrace.SpanStartOption) {
	opts = append(sc.opts, sc.kind.option())
	if sc.attrs != nil {
		opts = append(opts, apiTrace.WithAttributes(sc.attrs.Expand()...))
	}
	return opts
}
