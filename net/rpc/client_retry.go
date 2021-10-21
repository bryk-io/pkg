package rpc

import (
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
)

// RetryCallOptions define the required parameters to execute an RPC call with a retry strategy.
type RetryCallOptions struct {
	// Max number of tries for the call before returning an error
	Attempts uint

	// Sets the RPC timeout per call (including initial call)
	PerRetryTimeout *time.Duration

	// Produces increasing intervals for each attempt
	BackoffExponential *time.Duration
}

// WithRetry allows to set automatic retry settings when invoking a specific RPC method.
func WithRetry(config *RetryCallOptions) []grpc.CallOption {
	var opts []grpc.CallOption
	if config.Attempts > 0 {
		opts = append(opts, grpc_retry.WithMax(config.Attempts))
	}
	if config.PerRetryTimeout != nil {
		opts = append(opts, grpc_retry.WithPerRetryTimeout(*config.PerRetryTimeout))
	}
	if config.BackoffExponential != nil {
		opts = append(opts, grpc_retry.WithBackoff(grpc_retry.BackoffExponential(*config.BackoffExponential)))
	}
	return opts
}
