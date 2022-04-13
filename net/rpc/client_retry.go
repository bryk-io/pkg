package rpc

import (
	"time"

	grpcRetry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
)

// RetryOptions define the required parameters to execute an RPC call
// with a retry strategy.
type RetryOptions struct {
	// Max number of tries for the call before returning an error
	Attempts uint

	// Sets the RPC timeout per call (including initial call)
	PerRetryTimeout *time.Duration

	// Produces increasing intervals for each attempt
	BackoffExponential *time.Duration
}

// Retry specific failed RPC operations automatically.
func Retry(config *RetryOptions) []grpc.CallOption {
	var opts []grpc.CallOption
	if config.Attempts > 0 {
		opts = append(opts, grpcRetry.WithMax(config.Attempts))
	}
	if config.PerRetryTimeout != nil {
		opts = append(opts, grpcRetry.WithPerRetryTimeout(*config.PerRetryTimeout))
	}
	if config.BackoffExponential != nil {
		opts = append(opts, grpcRetry.WithBackoff(grpcRetry.BackoffExponential(*config.BackoffExponential)))
	}
	return opts
}
