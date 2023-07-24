package api

import (
	apiTrace "go.opentelemetry.io/otel/trace"
)

// SpanKind indicates the nature and/or owner of the traced operation.
type SpanKind string

const (
	// SpanKindUnspecified is the default value used when no span kind
	// is explicitly set.
	SpanKindUnspecified SpanKind = "unspecified"

	// SpanKindInternal should be used for internal-only tasks.
	SpanKindInternal SpanKind = "internal"

	// SpanKindServer should be used for server-side operations.
	SpanKindServer SpanKind = "server"

	// SpanKindClient should be used for client-side operations.
	SpanKindClient SpanKind = "client"

	// SpanKindConsumer should be used when an operation starts
	// by receiving a message from an MQ broker.
	SpanKindConsumer SpanKind = "consumer"

	// SpanKindProducer should be used when an operation involves
	// the publishing of a message to an MQ broker.
	SpanKindProducer SpanKind = "producer"
)

func (sk SpanKind) option() apiTrace.SpanStartOption {
	switch sk {
	case SpanKindInternal:
		return apiTrace.WithSpanKind(apiTrace.SpanKindInternal)
	case SpanKindServer:
		return apiTrace.WithSpanKind(apiTrace.SpanKindServer)
	case SpanKindClient:
		return apiTrace.WithSpanKind(apiTrace.SpanKindClient)
	case SpanKindConsumer:
		return apiTrace.WithSpanKind(apiTrace.SpanKindConsumer)
	case SpanKindProducer:
		return apiTrace.WithSpanKind(apiTrace.SpanKindProducer)
	case SpanKindUnspecified:
		return apiTrace.WithSpanKind(apiTrace.SpanKindUnspecified)
	default:
		return apiTrace.WithSpanKind(apiTrace.SpanKindUnspecified)
	}
}

func (sk SpanKind) String() string {
	switch sk {
	case SpanKindInternal:
		return "internal"
	case SpanKindServer:
		return "server"
	case SpanKindClient:
		return "client"
	case SpanKindConsumer:
		return "consumer"
	case SpanKindProducer:
		return "producer"
	case SpanKindUnspecified:
		return "unspecified"
	default:
		return "unspecified"
	}
}
