package rpc

import (
	"context"

	"golang.org/x/time/rate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/tap"
)

type rateTap struct {
	limit *rate.Limiter
}

func (t *rateTap) handler(ctx context.Context, info *tap.Info) (context.Context, error) {
	if !t.limit.Allow() {
		return nil, status.Errorf(codes.ResourceExhausted, "service rate limit exceeded")
	}
	return ctx, nil
}

// WithRateLimit will attach a simple rate limit handler on the server with a burst
// of 20% on the provided value.
func newRateTap(limit uint32) *rateTap {
	return &rateTap{
		limit: rate.NewLimiter(rate.Limit(limit), int(limit/20)),
	}
}
