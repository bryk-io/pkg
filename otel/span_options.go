package otel

// SpanOption allow adjusting span settings at the moment of creation.
type SpanOption func(s *span)

// WithSpanKind adjust the kind value for the created span. When no
// value is provided "unspecified" is used by default.
func WithSpanKind(sk SpanKind) SpanOption {
	return func(s *span) {
		s.kind = sk
		s.opts = append(s.opts, sk.option())
	}
}

// WithSpanAttributes adds the attributes related to a specific task. These
// attributes are used to describe the work a Span represents. If multiple of
// these options are passed the attributes of each successive option will
// extend/override any previously set value.
func WithSpanAttributes(attrs Attributes) SpanOption {
	return func(s *span) {
		s.attrs.Join(attrs)
	}
}

// WithSpanBaggage allows setting arbitrary key/value pairs that you can use to
// propagate observability data between services.
func WithSpanBaggage(attrs Attributes) SpanOption {
	return func(s *span) {
		s.bgg = attrs
	}
}
