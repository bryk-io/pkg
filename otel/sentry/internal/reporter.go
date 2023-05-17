package internal

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/getsentry/sentry-go"
	apiErrors "go.bryk.io/pkg/otel/errors"
)

// Reporter implementation to submit error data to a Sentry server.
// More information: https://sentry.io/
type Reporter struct {
	Hub    *sdk.Hub
	Client *sdk.Client
}

// NewReporter returns a new error reporter instance.
// More information: https://sentry.io/
func NewReporter(dsn, env, release string) (apiErrors.Reporter, error) {
	hub := sdk.NewHub(nil, sdk.NewScope())
	client, err := sdk.NewClient(sdk.ClientOptions{
		Dsn:              dsn,
		Debug:            false,
		Release:          release,
		Environment:      env,
		TracesSampleRate: 1.0,
		AttachStacktrace: true,
		Integrations: func(list []sdk.Integration) []sdk.Integration {
			var filtered []sdk.Integration
			for _, el := range list {
				// Remove default 'contextify' implementation
				if el.Name() == "ContextifyFrames" {
					continue
				}
				filtered = append(filtered, el)
			}
			// Add custom event processor
			return append(filtered, newEventProcessor())
		},
	})
	if err != nil {
		return nil, err
	}
	hub.BindClient(client)
	return &Reporter{
		Hub:    hub,
		Client: client,
	}, nil
}

// Start a new operation.
//
// Sentry processing rules:
//   - operations with `tx` names are reported directly in the trace view
//   - operations without `tx` names are reported as child spans and
//     viewable on the event details page
//   - operations using `continue` are always reported as a child spans
func (sr *Reporter) Start(ctx context.Context, name string, opts ...apiErrors.OperationOption) apiErrors.Operation {
	// Operation internals
	scope := sdk.NewScope()
	hub := sdk.NewHub(sr.Client, scope)

	// Check for operation options in the provided context
	if ctxOpts, ok := ctx.Value(spanOptsKey).([]apiErrors.OperationOption); ok {
		opts = append(ctxOpts, opts...)
	}

	// Bare operation instance
	op := new(Operation)
	op.Name = name // use provided name as default operation identifier
	for _, opt := range opts {
		opt(op)
	}

	// Get parent operation reference. Child spans use parent's scope and
	// context by default.
	parent, isChild := ctx.Value(currentOpKey).(*Operation)
	if isChild {
		scope = parent.Scope
		ctx = parent.Sp.Context()
	}

	// Create new root span context
	if !isChild {
		if !sdk.HasHubOnContext(ctx) {
			ctx = sdk.SetHubOnContext(ctx, hub)
		}
		// Parent spans MUST have a transaction associated, otherwise are
		// reported as "unlabeled" transactions. If no provided by the user
		// we use `name` for the transaction and start an operation named `root`.
		if op.Txn == "" {
			op.Txn = name
			op.Name = "root"
			op.Opts = append(op.Opts, sdk.WithTransactionName(name))
		}
	}

	// Spans continuing a remote trace always use a new base context
	if op.ToCont != "" {
		if !sdk.HasHubOnContext(ctx) {
			ctx = sdk.SetHubOnContext(ctx, hub)
		}
	}

	// Finish operation setup
	op.Scope = scope
	op.Hub = hub
	op.Sp = &Span{sp: sdk.StartSpan(ctx, op.Name, op.Opts...)}
	op.Sp.Status("ok")
	op.Submit = func(err error) string {
		return fmt.Sprintf("%v", hub.CaptureException(err))
	}
	return op
}

// Flush waits until the underlying transport sends any buffered events
// to the sentry server, blocking for at most the given `timeout`. It
// returns false if the timeout was reached. In that case, some events
// may not have been sent. Flush should be called before terminating
// the program to avoid unintentionally dropping events.
func (sr *Reporter) Flush(timeout time.Duration) bool {
	return sr.Hub.Flush(timeout)
}

// ToContext registers the operation `op` in the provided context instance.
func (sr *Reporter) ToContext(ctx context.Context, op apiErrors.Operation) context.Context {
	return ToContext(ctx, op)
}

// FromContext recovers an operation instance stored in `ctx`; this method
// returns `nil` if no operation was found in the provided context.
func (sr *Reporter) FromContext(ctx context.Context) apiErrors.Operation {
	return FromContext(ctx)
}

// Inject set cross-cutting concerns from the operation into the carrier.
// Allows to propagate operation details across service boundaries.
func (sr *Reporter) Inject(op apiErrors.Operation, carrier apiErrors.Carrier) {
	Inject(op, carrier)
}

// Extract reads cross-cutting concerns from the carrier into a Context.
func (sr *Reporter) Extract(ctx context.Context, carrier apiErrors.Carrier) context.Context {
	return Extract(ctx, carrier)
}
