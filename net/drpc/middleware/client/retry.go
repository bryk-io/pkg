package client

import (
	"context"
	"time"

	xlog "go.bryk.io/pkg/log"
	"storj.io/drpc"
)

// Retry failed requests up to the `max` number of attempts specified. Retried
// errors will be logged as warnings along with details about the specific
// attempt. Multiple tries introduce an increasingly longer backoff delay to
// account for transient failures on the remote. The specific delay for each
// attempt is calculated (in ms) as: `delay * (factor * attempt_number)`.
func Retry(max uint, ll xlog.Logger) Middleware {
	return func(next Interceptor) Interceptor {
		return retry{
			tries:  0,
			limit:  max,
			delay:  300,
			factor: 0.85,
			log:    ll,
			next:   next,
		}
	}
}

type retry struct {
	tries  uint        // number of attempts per-request
	limit  uint        // max number of tries
	factor float32     // backoff factor
	delay  uint        // initial delay value
	log    xlog.Logger // logger
	next   Interceptor // chained interceptor
}

func (md retry) Invoke(ctx context.Context, rpc string, enc drpc.Encoding, in, out drpc.Message) error {
	defer func() {
		// reset attempts counter
		md.tries = 0
	}()
	for {
		md.tries++
		err := md.next.Invoke(ctx, rpc, enc, in, out)
		if err == nil {
			return nil
		}

		// Operation fields
		fields := xlog.Fields{
			"error":         err.Error(),
			"retry.attempt": md.tries,
		}

		// Verify limit
		if md.tries == md.limit {
			md.log.WithFields(fields).Warning("retry: max attempts exceeded")
			return err
		}

		// Automatic retry with backoff factor
		// pause = delay * (factor * tries)
		pause := time.Duration(float32(md.delay)*(md.factor*float32(md.tries))) * time.Millisecond
		fields.Set("retry.pause_ms", pause.Milliseconds())
		fields.Set("retry.pause", pause.String())
		md.log.WithFields(fields).Debug("retry: delaying new attempt")
		<-time.After(pause)
		md.log.WithFields(fields).Warning("retry: re-submitting request")
		continue
	}
}

func (md retry) NewStream(ctx context.Context, rpc string, enc drpc.Encoding) (drpc.Stream, error) {
	defer func() {
		// reset attempts counter
		md.tries = 0
	}()
	for {
		md.tries++
		st, err := md.next.NewStream(ctx, rpc, enc)
		if err == nil {
			return st, nil
		}

		// Operation fields
		fields := xlog.Fields{
			"error":         err.Error(),
			"retry.attempt": md.tries,
		}

		// Verify limit
		if md.tries == md.limit {
			md.log.WithFields(fields).Warning("retry: max attempts exceeded")
			return nil, err
		}

		// Automatic retry with backoff factor
		// pause = delay * (factor * tries)
		pause := time.Duration(float32(md.delay)*(md.factor*float32(md.tries))) * time.Millisecond
		fields.Set("retry.pause_ms", pause.Milliseconds())
		fields.Set("retry.pause", pause.String())
		md.log.WithFields(fields).Debug("retry: delaying new attempt")
		<-time.After(pause)
		md.log.WithFields(fields).Warning("retry: re-submitting request")
		continue
	}
}
