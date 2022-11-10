package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	sdk "github.com/getsentry/sentry-go"
	apiErrors "go.bryk.io/pkg/otel/errors"
)

// Used to store an operation instance as a context value.
type opContextKeyType int

const currentOpKey opContextKeyType = iota

// Operation instances are used to describe relevant tasks in your application.
// The instance can be used to collect additional contextual information that
// could end up being reported in case of an exception using `Capture`.
type Operation struct {
	Txn    string                 // transaction name
	Name   string                 // operation name
	ToCont string                 // operation is continuing an existing trace
	Opts   []sdk.SpanOption       // span options
	Sp     *Span                  // internal span
	Hub    *sdk.Hub               // operation hub
	Scope  *sdk.Scope             // operation scope
	Submit func(err error) string // report function
	usr    *apiErrors.User        // user data associated with the operation
	done   bool
	mu     sync.Mutex
}

// Context returns the operation underlying context instance.
func (op *Operation) Context() context.Context {
	return context.WithValue(context.Background(), currentOpKey, op)
}

// Report an exception. Usually only unrecoverable errors should be reported
// at the end of the processing attempt of a given task. This method returns
// the event identifier for the error report.
func (op *Operation) Report(err error) string {
	return op.Submit(err)
}

// Level reported for the operation.
func (op *Operation) Level(level string) {
	op.Scope.SetLevel(getLevel(level))
}

// Status value reported on the associated span. Valid values are:
//   - ok (default)
//   - error
//   - canceled
//   - aborted
//   - unauthenticated
func (op *Operation) Status(status string) {
	op.Sp.Status(status)
}

// User can be used to declare the user associated with the operation. If used,
// at least an ID or an IP address should be provided. If called multiple times
// the user data will be merged/updated.
func (op *Operation) User(usr apiErrors.User) {
	if op.usr == nil {
		op.usr = new(apiErrors.User)
	}
	mergeUserData(op.usr, usr)
	op.Scope.SetUser(sdkUser(*op.usr))
}

// Tags adds/updates a group of key/value pairs as operation's metadata.
func (op *Operation) Tags(tags map[string]interface{}) {
	t := make(map[string]string)
	for k, v := range tags {
		t[k] = fmt.Sprintf("%v", v)
	}
	op.Scope.SetTags(t)
	op.Sp.Tags(tags)
}

// Segment registers additional contextual data under `key`.
func (op *Operation) Segment(key string, data map[string]interface{}) {
	op.Scope.SetContext(key, data)
}

// Event can be used to register activity worth reporting; this usually
// provides a progression of activity/tasks leading to a potential error
// condition.
//
// There are some special attributes you can add to events:
//   - event.kind: set to "default" if not provided
//   - event.category: set to "event" if not provided
//   - event.level: set to "info" if not provided.
//   - event.data: provides additional payload data, "nil" by default
//
// event.kind values:
//   - debug: typically a log message
//   - info: provide additional details to help identify the root cause of an issue
//   - error: error/warning occurring prior to a reported exception
//   - navigation: `event.data` must include key `from` and `to`
//   - http: http requests started from the app; `event.data` can include `http.request`
//   - query: describe and report database interactions
//   - user: describe user interactions
//
// https://develop.sentry.dev/sdk/event-payloads/breadcrumbs/#breadcrumb-types
func (op *Operation) Event(msg string, attributes ...map[string]interface{}) {
	attrs := join(attributes...)

	// Default values
	kind := "default"
	level := "info"
	category := "event"
	data := make(map[string]interface{})
	if k, ok := attrs["event.kind"]; ok {
		kind = fmt.Sprintf("%v", k)
	}
	if lvl, ok := attrs["event.level"]; ok {
		level = fmt.Sprintf("%v", lvl)
	}
	if cat, ok := attrs["event.category"]; ok {
		category = fmt.Sprintf("%v", cat)
	}
	if dt, ok := attrs["event.data"]; ok {
		if js, err := json.Marshal(dt); err == nil {
			_ = json.Unmarshal(js, &data)
		}
	}

	// Add breadcrumb
	op.Scope.AddBreadcrumb(&sdk.Breadcrumb{
		Type:      kind,
		Category:  category,
		Message:   msg,
		Data:      data,
		Level:     getLevel(level),
		Timestamp: time.Now(),
	}, 100)
}

// Finish sets the span's end time and, if the span is the root of a span tree,
// sends the span tree to Sentry as a transaction.
func (op *Operation) Finish() {
	op.mu.Lock()
	defer op.mu.Unlock()
	if op.done {
		return
	}
	op.done = true
	op.Sp.Finish()
}

// TraceID returns the trace propagation value; it can be used with the `sentry-trace`
// HTTP header to manually propagate the operation context across service boundaries.
func (op *Operation) TraceID() string {
	return op.Sp.TraceID()
}

// Inject set cross-cutting concerns from the operation into the carrier. Allows
// to propagate operation details across service boundaries.
func (op *Operation) Inject(mc apiErrors.Carrier) {
	mc.Set("sentry-trace", op.TraceID())
}

func mergeUserData(sink *apiErrors.User, update apiErrors.User) {
	if update.ID != "" {
		sink.ID = update.ID
	}
	if update.Email != "" {
		sink.Email = update.Email
	}
	if update.IPAddress != "" {
		sink.IPAddress = update.IPAddress
	}
	if update.Username != "" {
		sink.Username = update.Username
	}
	if update.Name != "" {
		sink.Name = update.Name
	}
	if update.Segment != "" {
		sink.Segment = update.Segment
	}
	for k, v := range update.Data {
		sink.Data[k] = v
	}
}

func sdkUser(usr apiErrors.User) sdk.User {
	return sdk.User{
		ID:        usr.ID,
		Email:     usr.Email,
		IPAddress: usr.IPAddress,
		Username:  usr.Username,
		Name:      usr.Name,
		Segment:   usr.Segment,
		Data:      usr.Data,
	}
}
