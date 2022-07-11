package errors

import (
	"context"
	"time"
)

// Reporter instances allow to report error data to external services.
// Useful for tracking, compliance and advance telemetry solutions.
type Reporter interface {
	// Start a new operation. The operation instance can be used to collect
	// valuable information that could end up being reported in case of an
	// exception using the `Report` method on the operation instance.
	Start(ctx context.Context, name string, opts ...OperationOption) Operation

	// ToContext registers the operation `op` in the provided context instance.
	ToContext(ctx context.Context, op Operation) context.Context

	// FromContext recovers an operation instance stored in `ctx`; this method
	// returns `nil` if no operation was found in the provided context.
	FromContext(ctx context.Context) Operation

	// Inject set cross-cutting concerns from the operation into the carrier. Allows
	// to propagate operation details across service boundaries.
	Inject(op Operation, carrier Carrier)

	// Extract reads cross-cutting concerns from the carrier into a Context.
	Extract(ctx context.Context, carrier Carrier) context.Context

	// Flush should be used to send any buffered events to the tracking server,
	// blocking for at most the given timeout. Must return `false` if the timeout
	// was reached without successfully delivering the error data.
	Flush(timeout time.Duration) bool
}

// Operation instances are used to describe relevant tasks in your application.
// The instance can be used to collect additional contextual information that
// could end up being reported in case of an exception using `Capture`.
type Operation interface {
	// Context returns the operation underlying context instance.
	Context() context.Context

	// Level reported for the operation.
	Level(level string)

	// Tags adds/updates a group of key/value pairs as operation's metadata.
	Tags(tags map[string]interface{})

	// User can be used to declare the user associated with the operation. If used,
	// at least an ID or an IP address should be provided.
	User(usr User)

	// Segment registers additional contextual data under `key`.
	Segment(key string, data interface{})

	// Event can be used to register activity worth reporting; this usually
	// provides a progression of activity/tasks leading to a potential error
	// condition.
	Event(msg string, attributes ...map[string]interface{})

	// Report an exception. Usually only unrecoverable errors should be reported
	// at the end of the processing attempt of a given task. This will automatically
	// also finish the operation.
	Report(err error) string

	// Status value reported on the associated span. Valid values are:
	//   - ok (default)
	//   - error
	//   - canceled
	//   - aborted
	//   - unauthenticated
	Status(status string)

	// Finish will mark the operation as completed and trigger the reporting
	// mechanism.
	Finish()

	// TraceID returns the trace propagation value.
	TraceID() string

	// Inject set cross-cutting concerns from the operation into the carrier. Allows
	// to propagate operation details across service boundaries.
	Inject(mc Carrier)
}

// User describes the user associated with an operation. If this is used,
// at least an ID or an IP address should be provided.
type User struct {
	// Email address associated with a user account.
	Email string `json:"email,omitempty"`

	// Unique identifier for the user account; for example a UUID value.
	ID string `json:"id,omitempty"`

	// IP address obtained from an incoming HTTP/RPC requests.
	IPAddress string `json:"ip_address,omitempty"`

	// Username; must be unique and free from PII.
	Username string `json:"username,omitempty"`
}

// Carrier elements can be used to transfer cross-cutting concerns about an operation
// across service boundaries.
type Carrier interface {
	// Set a value.
	Set(key string, value string)

	// Get a previously set value.
	Get(key string) string
}

// OperationOption allows to adjust the behavior of error reporting operations
// at the moment of creation.
type OperationOption func(opt Operation)
