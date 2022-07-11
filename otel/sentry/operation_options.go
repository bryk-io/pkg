package sentry

import (
	apiErrors "go.bryk.io/pkg/otel/errors"
	"go.bryk.io/pkg/otel/sentry/internal"
)

// AsTransaction reports the operation as its own transaction. Transactions
// are analyzed independently by Sentry and reported directly on traces.
func AsTransaction(name string) apiErrors.OperationOption {
	return internal.AsTransaction(name)
}

// ToContinue mark the operation as a continuation of an existing trace; this
// is commonly required when propagating a trace across service boundaries.
func ToContinue(traceID string) apiErrors.OperationOption {
	return internal.ToContinue(traceID)
}
