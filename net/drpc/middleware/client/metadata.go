package client

import (
	"context"

	"storj.io/drpc"
	"storj.io/drpc/drpcmetadata"
)

// Metadata middleware adds the provided payload data to the context
// of every request (unary and stream) before sending it to the server.
func Metadata(payload map[string]string) Middleware {
	return func(next Interceptor) Interceptor {
		return md{
			payload: payload,
			next:    next,
		}
	}
}

type md struct {
	payload map[string]string
	next    Interceptor
}

func (m md) Invoke(ctx context.Context, rpc string, enc drpc.Encoding, in, out drpc.Message) error {
	ctx = drpcmetadata.AddPairs(ctx, m.payload)
	return m.next.Invoke(ctx, rpc, enc, in, out)
}

func (m md) NewStream(ctx context.Context, rpc string, enc drpc.Encoding) (drpc.Stream, error) {
	ctx = drpcmetadata.AddPairs(ctx, m.payload)
	return m.next.NewStream(ctx, rpc, enc)
}
