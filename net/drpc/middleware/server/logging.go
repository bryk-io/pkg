package server

import (
	"fmt"
	"io"
	"strings"
	"time"

	"go.bryk.io/pkg/errors"
	xlog "go.bryk.io/pkg/log"
	"go.bryk.io/pkg/metadata"
	"storj.io/drpc"
	"storj.io/drpc/drpcmetadata"
)

// Logging produce output for the processed RPC requests tagged with
// standard ECS details by default. Fields can be extended by providing
// a hook function.
func Logging(logger xlog.Logger, hook LoggingHook) Middleware {
	return func(next drpc.Handler) drpc.Handler {
		return logging{
			ll:   logger,
			hook: hook,
			next: next,
		}
	}
}

// LoggingHook provides a mechanism to extend a message fields just
// before is submitted.
type LoggingHook func(fields *metadata.MD, stream drpc.Stream)

type wrappedStream struct {
	drpc.Stream
	ll xlog.Logger
}

func (ws wrappedStream) MsgSend(msg drpc.Message, enc drpc.Encoding) (err error) {
	ws.ll.Debug("send message")
	if err = ws.Stream.MsgSend(msg, enc); err != nil {
		ws.ll.WithField("error", err.Error()).Warning("send message failed")
	}
	return
}

func (ws wrappedStream) MsgRecv(msg drpc.Message, enc drpc.Encoding) (err error) {
	ws.ll.Debug("receive message")
	if err = ws.Stream.MsgRecv(msg, enc); err != nil && !errors.Is(err, io.EOF) {
		ws.ll.WithField("error", err.Error()).Warning("receive message failed")
	}
	return
}

type logging struct {
	ll   xlog.Logger
	hook LoggingHook
	next drpc.Handler
}

func (md logging) HandleRPC(stream drpc.Stream, rpc string) error {
	// Get basic request details
	fields := getFields(stream, rpc)
	if md.hook != nil {
		md.hook(&fields, stream)
	}

	// Start message
	md.ll.WithFields(fields.Values()).Info(rpc)

	// Process request
	start := time.Now().UTC()
	ws := wrappedStream{
		Stream: stream,
		ll:     md.ll.Sub(fields.Values()),
	}
	err := md.next.HandleRPC(ws, rpc)
	end := time.Now().UTC()
	lapse := end.Sub(start)

	// Additional details
	fields.Set("duration", lapse.String())
	fields.Set("duration_ms", fmt.Sprintf("%.3f", lapse.Seconds()*1000))
	fields.Set("event.start", start.Nanosecond())
	fields.Set("event.end", end.Nanosecond())
	fields.Set("event.duration", lapse.Nanoseconds())

	// End message
	if err != nil {
		fields.Set("error", err.Error())
		md.ll.WithFields(fields.Values()).Errorf("%s failed", rpc)
		return err
	}
	md.ll.WithFields(fields.Values()).Infof("%s completed", rpc)
	return nil
}

func getFields(stream drpc.Stream, rpc string) metadata.MD {
	fields := metadata.New()
	segments := strings.Split(rpc, "/")
	if len(segments) == 3 {
		fields.Set("rpc.system", "drpc")
		fields.Set("rpc.service", segments[1])
		fields.Set("rpc.method", segments[2])
	}
	if md, ok := drpcmetadata.Get(stream.Context()); ok {
		for k, v := range md {
			fields.Set(k, v)
		}
	}
	return fields
}
