package otel

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
// extend/override any previously set value.
func WithSpanAttributes(attrs Attributes) SpanOption {
	return func(s *Span) {
		s.attrs.Join(attrs)
	}
}

// WithSpanBaggage allows setting arbitrary key/value pairs that you can use to
// propagate observability data between services. Baggage attributes are usually
// not reported/captured by exporters (like Jaeger or Zipkin). You can force the
// propagation by duplicating the baggage as span attributes.
func WithSpanBaggage(attrs Attributes, propagate bool) SpanOption {
	return func(s *Span) {
		s.bgg = attrs
		s.bggPropagate = propagate
	}
}
