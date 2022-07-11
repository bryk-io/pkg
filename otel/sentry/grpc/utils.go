package sentrygrpc

import (
	"context"
	"strings"

	apiErrors "go.bryk.io/pkg/otel/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

const (
	msgRecv = "message received"
	msgSent = "message sent"
)

func spanInfo(fullMethod string) (string, map[string]interface{}) {
	attrs := map[string]interface{}{"rpc.system": "grpc"}
	name := strings.TrimLeft(fullMethod, "/")
	parts := strings.SplitN(name, "/", 2)
	if len(parts) != 2 {
		// invalid format, does not follow `/package.service/method`
		return name, attrs
	}
	if service := parts[0]; service != "" {
		attrs["rpc.service"] = service
	}
	if method := parts[1]; method != "" {
		attrs["rpc.method"] = method
	}
	return name, attrs
}

func reportEvent(op apiErrors.Operation, ev event) {
	op.Event(ev.desc, ev.attributes())
}

type event struct {
	id      int
	level   string
	desc    string
	payload interface{}
	err     error
}

func (ev event) attributes() map[string]interface{} {
	level := "debug"
	if ev.level != "" {
		level = ev.level
	}
	if ev.err != nil {
		level = "warning"
	}
	attrs := map[string]interface{}{
		"event.level": level,
		"event.kind":  "console",
	}
	data := map[string]interface{}{}
	if ev.id != 0 {
		data["message.id"] = ev.id
	}
	if p, ok := ev.payload.(proto.Message); ok {
		data["message.uncompressed_size"] = proto.Size(p)
	}
	attrs["event.data"] = data
	return attrs
}

func wrapClientStream(
	ctx context.Context,
	op apiErrors.Operation,
	s grpc.ClientStream,
	desc *grpc.StreamDesc) *clientStream {
	ws := &clientStream{
		op:           op,
		ctx:          ctx,
		desc:         desc,
		done:         make(chan error),
		close:        make(chan struct{}),
		events:       make(chan event),
		ClientStream: s,
	}
	go ws.handleEvents()
	return ws
}

func wrapServerStream(ctx context.Context, op apiErrors.Operation, ss grpc.ServerStream) *serverStream {
	ws := &serverStream{
		op:           op,
		ctx:          ctx,
		events:       make(chan event),
		close:        make(chan struct{}),
		ServerStream: ss,
	}
	go ws.handleEvents()
	return ws
}
