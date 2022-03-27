package otel

import (
	apiTrace "go.opentelemetry.io/otel/trace"
)

// SpanOption allow adjusting span settings at the moment of creation.
type SpanOption func(s *Span)

// WithSpanKind adjust the kind value for the created span. When no
// value is provided "unspecified" is used by default.
func WithSpanKind(sk SpanKind) SpanOption {
	return func(s *Span) {
		s.kind = sk
		s.opts = append(s.opts, sk.option())
	}
}

// WithSpanAttributes adds the attributes related to a specific task. These
// attributes are used to describe the work a Span represents. If multiple of
// these options are passed the attributes of each successive option will
// extend the attributes instead of overwriting. There is no guarantee of
// uniqueness in the resulting attributes.
func WithSpanAttributes(attrs Attributes) SpanOption {
	return func(s *Span) {
		s.attrs = attrs
		s.opts = append(s.opts, apiTrace.WithAttributes(attrs.Expand()...))
	}
}
