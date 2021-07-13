package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	xlog "go.bryk.io/pkg/log"
	"storj.io/drpc"
	"storj.io/drpc/drpcmetadata"
)

// Logging produce output for the processed RPC requests tagged with
// standard ECS details by default. Fields can be extended by providing
// a hook function.
func Logging(logger xlog.Logger, hook LoggingHook) Middleware {
	return func(next Interceptor) Interceptor {
		return logging{
			ll:   logger,
			hook: hook,
			next: next,
		}
	}
}

// LoggingHook provides a mechanism to extend a message fields just
// before is submitted.
type LoggingHook func(ctx context.Context, rpc string, fields *xlog.Fields)

type logging struct {
	ll   xlog.Logger
	hook LoggingHook
	next Interceptor
}

func (md logging) Invoke(ctx context.Context, rpc string, enc drpc.Encoding, in, out drpc.Message) error {
	// Get basic request details
	fields := getFields(ctx, rpc)
	fields.Set("rpc.kind", "unary")
	if md.hook != nil {
		md.hook(ctx, rpc, &fields)
	}

	// Start message
	md.ll.WithFields(fields).Info(rpc)

	// Process request
	start := time.Now().UTC()
	err := md.next.Invoke(ctx, rpc, enc, in, out)
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
		md.ll.WithFields(fields).Errorf("%s failed", rpc)
	} else {
		md.ll.WithFields(fields).Infof("%s completed", rpc)
	}
	return err
}

func (md logging) NewStream(ctx context.Context, rpc string, enc drpc.Encoding) (drpc.Stream, error) {
	// Get basic request details
	fields := getFields(ctx, rpc)
	fields.Set("rpc.kind", "stream")
	if md.hook != nil {
		md.hook(ctx, rpc, &fields)
	}

	// Start message
	md.ll.WithFields(fields).Info(rpc)

	// Process request
	start := time.Now().UTC()
	fields.Set("event.start", start.Nanosecond())
	st, err := md.next.NewStream(ctx, rpc, enc)

	// Error message
	if err != nil {
		fields.Set("error", err.Error())
		md.ll.WithFields(fields).Errorf("%s failed", rpc)
		return st, err
	}

	// Delay end message for when the stream is closed
	go func() {
		// Wait for stream
		<-st.Context().Done()

		// End message
		end := time.Now().UTC()
		lapse := end.Sub(start)
		fields.Set("duration", lapse.String())
		fields.Set("duration_ms", fmt.Sprintf("%.3f", lapse.Seconds()*1000))
		fields.Set("event.end", end.Nanosecond())
		fields.Set("event.duration", lapse.Nanoseconds())
		md.ll.WithFields(fields).Infof("%s completed", rpc)
	}()
	return st, err
}

func getFields(ctx context.Context, rpc string) xlog.Fields {
	fields := xlog.Fields{}
	segments := strings.Split(rpc, "/")
	if len(segments) == 3 {
		fields.Set("rpc.system", "drpc")
		fields.Set("rpc.service", segments[1])
		fields.Set("rpc.method", segments[2])
	}
	if md, ok := drpcmetadata.Get(ctx); ok {
		for k, v := range md {
			fields.Set(k, v)
		}
	}
	return fields
}
