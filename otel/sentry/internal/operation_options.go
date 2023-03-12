package internal

import (
	sdk "github.com/getsentry/sentry-go"
	apiErrors "go.bryk.io/pkg/otel/errors"
)

// AsTransaction reports the operation as its own transaction.
// Transactions are analyzed independently by Sentry and reported
// directly on traces.
func AsTransaction(name string) apiErrors.OperationOption {
	return func(opt apiErrors.Operation) {
		so, ok := opt.(*Operation)
		if !ok {
			return
		}
		so.Txn = name
		so.Opts = append(so.Opts, sdk.TransactionName(name))
	}
}

// ToContinue mark the operation as a continuation of an existing
// trace; this is commonly required when propagating a trace across
// service boundaries.
func ToContinue(traceID string) apiErrors.OperationOption {
	return func(opt apiErrors.Operation) {
		so, ok := opt.(*Operation)
		if !ok {
			return
		}
		so.ToCont = traceID
		so.Opts = append(so.Opts, sdk.ContinueFromTrace(traceID))
	}
}
