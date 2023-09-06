package otelgrpc

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"go.bryk.io/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semConv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	grpcCodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

/*
The contents of this file are a fork of the original package:
go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc (v0.43.0)

To include some changes/adjustments:
- Simpler API surface
- Support for "X-Real-IP"
- Support for PROXY protocol
- Additional span attributes
*/

// Semantic conventions for attribute keys for gRPC.
const (
	// instrumentationName is the name of this instrumentation package.
	instrumentationName = "go.bryk.io/otel/grpc"

	// Type of message transmitted or received.
	rpcMessageNameKey = attribute.Key("message.name")

	// Type of message transmitted or received.
	rpcMessageTypeKey = attribute.Key("message.type")

	// Identifier of message transmitted or received.
	rpcMessageIDKey = attribute.Key("message.id")

	// The uncompressed size of the message transmitted or received in bytes.
	rpcMessageUncompressedSizeKey = attribute.Key("message.uncompressed_size")

	// rpcStatusCodeKey is convention for numeric status code of a gRPC request.
	rpcStatusCodeKey = attribute.Key("rpc.grpc.status_code")
)

// Semantic conventions for common RPC attributes.
var (
	// Semantic convention for gRPC as the remote system.
	rpcSystemGRPC = semConv.RPCSystemKey.String(semConv.RPCSystemGRPC.Value.AsString())

	// Semantic conventions for RPC message types.
	rpcMessageTypeSent     = rpcMessageTypeKey.String("SENT")
	rpcMessageTypeReceived = rpcMessageTypeKey.String("RECEIVED")
)

type messageType attribute.KeyValue

func version() string {
	return "0.43.0"
}

// Event adds an event of the messageType to the span associated with the
// passed context with id and size (if message is a proto message).
func (m messageType) Event(ctx context.Context, id int, message interface{}) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	if pm, ok := message.(proto.Message); ok {
		span.AddEvent("message", trace.WithAttributes(
			attribute.KeyValue(m),
			rpcMessageIDKey.Int(id),
			rpcMessageNameKey.String(string(proto.MessageName(pm))),
			rpcMessageUncompressedSizeKey.Int(proto.Size(pm)),
		))
	} else {
		span.AddEvent("message", trace.WithAttributes(
			attribute.KeyValue(m),
			rpcMessageIDKey.Int(id),
		))
	}
}

var (
	messageSent     = messageType(rpcMessageTypeSent)
	messageReceived = messageType(rpcMessageTypeReceived)
)

// UnaryClientInterceptor returns a grpc.UnaryClientInterceptor suitable
// for use in a grpc.Dial call.
func unaryClientInterceptor() grpc.UnaryClientInterceptor {
	tracer := otel.GetTracerProvider().Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(version()),
	)

	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, callOpts ...grpc.CallOption) error { // nolint: lll
		// ! apply filter on 'method' to skip instrumentation
		// ! if filter(method) {
		// !	return invoker(ctx, method, req, reply, cc, callOpts...)
		// ! }
		// get metdata
		requestMetadata, _ := metadata.FromOutgoingContext(ctx)
		metadataCopy := requestMetadata.Copy()

		// create span
		name, attr := spanInfo(method, cc.Target())
		var span trace.Span
		ctx, span = tracer.Start(
			ctx,
			name,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(attr...),
		)
		defer span.End()

		// prepare outgoing context
		inject(ctx, &metadataCopy)
		ctx = metadata.NewOutgoingContext(ctx, metadataCopy)

		// invoke method
		messageSent.Event(ctx, 1, req)
		err := invoker(ctx, method, req, reply, cc, callOpts...)
		messageReceived.Event(ctx, 1, reply)

		// report final status
		if err != nil {
			s, _ := status.FromError(err)
			span.SetStatus(codes.Error, s.Message())
			span.SetAttributes(statusCodeAttr(s.Code()))
		} else {
			span.SetAttributes(statusCodeAttr(grpcCodes.OK))
		}

		return err
	}
}

type streamEventType int

type streamEvent struct {
	Type streamEventType
	Err  error
}

const (
	receiveEndEvent streamEventType = iota
	errorEvent
)

// clientStream  wraps around the embedded grpc.ClientStream, and intercepts the
// `RecvMsg` and `SendMsg` method call.
type clientStream struct {
	grpc.ClientStream

	desc              *grpc.StreamDesc
	events            chan streamEvent
	eventsDone        chan struct{}
	finished          chan error
	receivedMessageID int
	sentMessageID     int
}

var _ = proto.Marshal

func (w *clientStream) RecvMsg(m interface{}) error {
	err := w.ClientStream.RecvMsg(m)

	if err == nil && !w.desc.ServerStreams {
		w.sendStreamEvent(receiveEndEvent, nil)
	} else if errors.Is(err, io.EOF) {
		w.sendStreamEvent(receiveEndEvent, nil)
	} else if err != nil {
		w.sendStreamEvent(errorEvent, err)
	} else {
		w.receivedMessageID++
		messageReceived.Event(w.Context(), w.receivedMessageID, m)
	}

	return err
}

func (w *clientStream) SendMsg(m interface{}) error {
	err := w.ClientStream.SendMsg(m)

	w.sentMessageID++
	messageSent.Event(w.Context(), w.sentMessageID, m)
	if err != nil {
		w.sendStreamEvent(errorEvent, err)
	}

	return err
}

func (w *clientStream) Header() (metadata.MD, error) {
	md, err := w.ClientStream.Header()
	if err != nil {
		w.sendStreamEvent(errorEvent, err)
	}
	return md, err
}

func (w *clientStream) CloseSend() error {
	err := w.ClientStream.CloseSend()
	if err != nil {
		w.sendStreamEvent(errorEvent, err)
	}
	return err
}

func wrapClientStream(ctx context.Context, s grpc.ClientStream, desc *grpc.StreamDesc) *clientStream {
	events := make(chan streamEvent)
	eventsDone := make(chan struct{})
	finished := make(chan error)

	go func() {
		defer close(eventsDone)

		for {
			select {
			case event := <-events:
				switch event.Type {
				case receiveEndEvent:
					finished <- nil
					return
				case errorEvent:
					finished <- event.Err
					return
				}
			case <-ctx.Done():
				finished <- ctx.Err()
				return
			}
		}
	}()

	return &clientStream{
		ClientStream: s,
		desc:         desc,
		events:       events,
		eventsDone:   eventsDone,
		finished:     finished,
	}
}

func (w *clientStream) sendStreamEvent(eventType streamEventType, err error) {
	select {
	case <-w.eventsDone:
	case w.events <- streamEvent{Type: eventType, Err: err}:
	}
}

// StreamClientInterceptor returns a grpc.StreamClientInterceptor suitable
// for use in a grpc.Dial call.
func streamClientInterceptor() grpc.StreamClientInterceptor {
	tracer := otel.GetTracerProvider().Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(version()),
	)

	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, callOpts ...grpc.CallOption) (grpc.ClientStream, error) { // nolint: lll
		// ! apply filter on 'method' to skip instrumentation
		// ! if filter(method) {
		// !	return streamer(ctx, desc, cc, method, callOpts...)
		// ! }
		// get metdata
		requestMetadata, _ := metadata.FromOutgoingContext(ctx)
		metadataCopy := requestMetadata.Copy()

		// create span
		var span trace.Span
		name, attr := spanInfo(method, cc.Target())
		ctx, span = tracer.Start(ctx, name, trace.WithSpanKind(trace.SpanKindClient), trace.WithAttributes(attr...))

		// prepare outgoing context
		inject(ctx, &metadataCopy)
		ctx = metadata.NewOutgoingContext(ctx, metadataCopy)

		// open stream
		s, err := streamer(ctx, desc, cc, method, callOpts...)
		if err != nil {
			grpcStatus, _ := status.FromError(err)
			span.SetStatus(codes.Error, grpcStatus.Message())
			span.SetAttributes(statusCodeAttr(grpcStatus.Code()))
			span.End()
			return s, err
		}

		// wrap stream to intercept events
		stream := wrapClientStream(ctx, s, desc)
		go func() {
			err := <-stream.finished
			if err != nil {
				s, _ := status.FromError(err)
				span.SetStatus(codes.Error, s.Message())
				span.SetAttributes(statusCodeAttr(s.Code()))
			} else {
				span.SetAttributes(statusCodeAttr(grpcCodes.OK))
			}
			span.End()
		}()
		return stream, nil
	}
}

// UnaryServerInterceptor returns a grpc.UnaryServerInterceptor suitable
// for use in a grpc.NewServer call.
func unaryServerInterceptor() grpc.UnaryServerInterceptor {
	tracer := otel.GetTracerProvider().Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(version()),
	)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) { // nolint: lll
		// ! apply filter on 'method' to skip instrumentation
		// ! if filter(info.FullMethod) {
		// !	return handler(ctx, req)
		// ! }
		// get metdata
		requestMetadata, _ := metadata.FromIncomingContext(ctx)
		metadataCopy := requestMetadata.Copy()

		// preserve baggage and span context
		bags, spanCtx := extract(ctx, &metadataCopy)
		ctx = baggage.ContextWithBaggage(ctx, bags)

		// create span
		name, attr := spanInfo(info.FullMethod, peerFromCtx(ctx))
		ctx, span := tracer.Start(
			trace.ContextWithRemoteSpanContext(ctx, spanCtx),
			name,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(attr...),
		)
		defer span.End()

		// process request and set final span status
		messageReceived.Event(ctx, 1, req)
		resp, err := handler(ctx, req)
		if err != nil {
			s, _ := status.FromError(err)
			statusCode, msg := serverStatus(s)
			span.SetStatus(statusCode, msg)
			span.SetAttributes(statusCodeAttr(s.Code()))
			messageSent.Event(ctx, 1, s.Proto())
		} else {
			span.SetAttributes(statusCodeAttr(grpcCodes.OK))
			messageSent.Event(ctx, 1, resp)
		}
		return resp, err
	}
}

// serverStream wraps around the embedded grpc.ServerStream, and intercepts
// the `RecvMsg` and `SendMsg` method call.
type serverStream struct {
	grpc.ServerStream
	ctx               context.Context
	receivedMessageID int
	sentMessageID     int
}

func (w *serverStream) Context() context.Context {
	return w.ctx
}

func (w *serverStream) RecvMsg(m interface{}) error {
	err := w.ServerStream.RecvMsg(m)
	if err == nil {
		w.receivedMessageID++
		messageReceived.Event(w.Context(), w.receivedMessageID, m)
	}
	return err
}

func (w *serverStream) SendMsg(m interface{}) error {
	err := w.ServerStream.SendMsg(m)
	w.sentMessageID++
	messageSent.Event(w.Context(), w.sentMessageID, m)
	return err
}

func wrapServerStream(ctx context.Context, ss grpc.ServerStream) *serverStream {
	return &serverStream{
		ServerStream: ss,
		ctx:          ctx,
	}
}

// StreamServerInterceptor returns a grpc.StreamServerInterceptor suitable
// for use in a grpc.NewServer call.
func streamServerInterceptor() grpc.StreamServerInterceptor {
	tracer := otel.GetTracerProvider().Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(version()),
	)

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error { // nolint: lll
		// ! apply filter on 'method' to skip instrumentation
		// ! if filter(info.FullMethod) {
		// !	return handler(srv, wrapServerStream(ctx, ss))
		// ! }
		// get metdata
		ctx := ss.Context()
		requestMetadata, _ := metadata.FromIncomingContext(ctx)
		metadataCopy := requestMetadata.Copy()

		// preserve baggage and span context
		bags, spanCtx := extract(ctx, &metadataCopy)
		ctx = baggage.ContextWithBaggage(ctx, bags)

		// create span
		name, attr := spanInfo(info.FullMethod, peerFromCtx(ctx))
		ctx, span := tracer.Start(
			trace.ContextWithRemoteSpanContext(ctx, spanCtx),
			name,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(attr...),
		)
		defer span.End()

		// process request and set final span status
		err := handler(srv, wrapServerStream(ctx, ss))
		if err != nil {
			s, _ := status.FromError(err)
			statusCode, msg := serverStatus(s)
			span.SetStatus(statusCode, msg)
			span.SetAttributes(statusCodeAttr(s.Code()))
		} else {
			span.SetAttributes(statusCodeAttr(grpcCodes.OK))
		}
		return err
	}
}

// spanInfo returns a span name and all appropriate attributes from the gRPC
// method and peer address.
func spanInfo(fullMethod, peerAddress string) (string, []attribute.KeyValue) {
	attrs := []attribute.KeyValue{rpcSystemGRPC}
	name, mAttrs := parseFullMethod(fullMethod)
	attrs = append(attrs, mAttrs...)
	attrs = append(attrs, peerAttr(peerAddress)...)
	return name, attrs
}

// peerAttr returns attributes about the peer address.
func peerAttr(addr string) []attribute.KeyValue {
	host, p, err := net.SplitHostPort(addr)
	if err != nil {
		return []attribute.KeyValue(nil)
	}

	if host == "" {
		host = "127.0.0.1"
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		return []attribute.KeyValue(nil)
	}

	var attr []attribute.KeyValue
	if ip := net.ParseIP(host); ip != nil {
		attr = []attribute.KeyValue{
			semConv.NetSockPeerAddr(host),
			semConv.NetSockPeerPort(port),
		}
	} else {
		attr = []attribute.KeyValue{
			semConv.NetPeerName(host),
			semConv.NetPeerPort(port),
		}
	}

	return attr
}

// peerFromCtx returns a peer address from a context, if one exists.
func peerFromCtx(ctx context.Context) string {
	// Get peer details
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}
	_, port, err := net.SplitHostPort(p.Addr.String())
	if err != nil {
		return ""
	}

	// Look for proxy protocol details propagated as metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if addr := md.Get("x-real-ip"); len(addr) > 0 {
			return fmt.Sprintf("%s:%s", addr[0], port)
		}
		if addr := md.Get("x-forwarded-for"); len(addr) > 0 {
			return fmt.Sprintf("%s:%s", addr[0], port)
		}
	}
	return p.Addr.String()
}

// statusCodeAttr returns status code attribute based on given gRPC code.
func statusCodeAttr(c grpcCodes.Code) attribute.KeyValue {
	return rpcStatusCodeKey.Int64(int64(c))
}

// ParseFullMethod returns a span name following the OpenTelemetry semantic
// conventions as well as all applicable span attribute.KeyValue attributes based
// on a gRPC FullMethod.
func parseFullMethod(fullMethod string) (string, []attribute.KeyValue) {
	if !strings.HasPrefix(fullMethod, "/") {
		// invalid format, does not follow `/package.service/method`.
		return fullMethod, nil
	}
	name := fullMethod[1:]
	pos := strings.LastIndex(name, "/")
	if pos < 0 {
		// invalid format, does not follow `/package.service/method`.
		return name, nil
	}
	service, method := name[:pos], name[pos+1:]
	var attrs []attribute.KeyValue
	if service != "" {
		attrs = append(attrs, semConv.RPCServiceKey.String(service))
	}
	if method != "" {
		attrs = append(attrs, semConv.RPCMethodKey.String(method))
	}
	return name, attrs
}

// serverStatus returns a span status code and message for a given gRPC
// status code. It maps specific gRPC status codes to a corresponding span
// status code and message. This function is intended for use on the server
// side of a gRPC connection.
//
// If the gRPC status code is Unknown, DeadlineExceeded, Unimplemented,
// Internal, Unavailable, or DataLoss, it returns a span status code of Error
// and the message from the gRPC status. Otherwise, it returns a span status
// code of Unset and an empty message.
func serverStatus(grpcStatus *status.Status) (codes.Code, string) {
	switch grpcStatus.Code() {
	case grpcCodes.Unknown,
		grpcCodes.DeadlineExceeded,
		grpcCodes.Unimplemented,
		grpcCodes.Internal,
		grpcCodes.Unavailable,
		grpcCodes.DataLoss:
		return codes.Error, grpcStatus.Message()
	default:
		return codes.Unset, ""
	}
}

type metadataSupplier struct {
	metadata *metadata.MD
}

// assert that metadataSupplier implements the TextMapCarrier interface.
var _ propagation.TextMapCarrier = &metadataSupplier{}

func (s *metadataSupplier) Get(key string) string {
	values := s.metadata.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (s *metadataSupplier) Set(key string, value string) {
	s.metadata.Set(key, value)
}

func (s *metadataSupplier) Keys() []string {
	out := make([]string, 0, len(*s.metadata))
	for key := range *s.metadata {
		out = append(out, key)
	}
	return out
}

// Inject injects correlation context and span context into the gRPC
// metadata object. This function is meant to be used on outgoing
// requests.
func inject(ctx context.Context, md *metadata.MD) {
	otel.GetTextMapPropagator().Inject(ctx, &metadataSupplier{
		metadata: md,
	})
}

// Extract returns the correlation context and span context that
// another service encoded in the gRPC metadata object with Inject.
// This function is meant to be used on incoming requests.
func extract(ctx context.Context, md *metadata.MD) (baggage.Baggage, trace.SpanContext) {
	ctx = otel.GetTextMapPropagator().Extract(ctx, &metadataSupplier{
		metadata: md,
	})
	return baggage.FromContext(ctx), trace.SpanContextFromContext(ctx)
}
