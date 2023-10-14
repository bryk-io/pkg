package temporal

import (
	apiTrace "go.bryk.io/pkg/otel/api"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/interceptor"
)

type spanContextKey struct{}

type tracerSpanRef struct{ trace.SpanContext }

type tracerSpan struct{ apiTrace.Span }

func (t *tracerSpan) Finish(opts *interceptor.TracerFinishSpanOptions) {
	t.End(opts.Error)
}

type textMapCarrier map[string]string

func (t textMapCarrier) Get(key string) string        { return t[key] }
func (t textMapCarrier) Set(key string, value string) { t[key] = value }
func (t textMapCarrier) Keys() []string {
	ret := make([]string, 0, len(t))
	for k := range t {
		ret = append(ret, k)
	}
	return ret
}
