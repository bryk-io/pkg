package sse

import (
	"time"

	xlog "go.bryk.io/pkg/log"
)

// StreamOption provide a functional-style mechanism to adjust the behavior
// of a stream operator instance.
type StreamOption func(st *Stream) error

// WithMessageRetry adjust the `retry` message value, in milliseconds, set
// by the stream for all send messages and events. Default value is `2000`.
func WithMessageRetry(retry uint) StreamOption {
	return func(st *Stream) error {
		st.mu.Lock()
		st.retry = retry
		st.mu.Unlock()
		return nil
	}
}

// WithSendTimeout adjust the maximum time to wait for message delivery on
// send operations. Default value is 2 seconds.
func WithSendTimeout(timeout time.Duration) StreamOption {
	return func(st *Stream) error {
		st.mu.Lock()
		st.timeout = timeout
		st.mu.Unlock()
		return nil
	}
}

// WithLogger set the log handler for the stream operator. Logs are discarded
// by default.
func WithLogger(logger xlog.Logger) StreamOption {
	return func(st *Stream) error {
		st.mu.Lock()
		st.log = logger
		st.mu.Unlock()
		return nil
	}
}
