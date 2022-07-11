package internal

import (
	"context"
	"fmt"

	sdk "github.com/getsentry/sentry-go"
)

// A Span is the building block of a Sentry transaction. Spans build
// up a tree structure of timed operations. The span tree makes up a
// transaction event that is sent to Sentry when the root span is
// finished.
type Span struct {
	sp *sdk.Span // underlying span instance
}

// Context returns the context containing the span.
func (s *Span) Context() context.Context {
	return s.sp.Context()
}

// Finish sets the span's end time, unless already set. If the span is
// the root of a span tree, Finish sends the span tree to Sentry as a
// transaction.
func (s *Span) Finish() {
	s.sp.Finish()
}

// TraceID returns the trace propagation value.
func (s *Span) TraceID() string {
	return s.sp.ToSentryTrace()
}

// Status sets the span current status value. Valid values:
//  - ok (default)
//  - error
//  - aborted
//  - canceled
//  - unauthenticated
func (s *Span) Status(status string) {
	s.sp.Status = getStatus(status)
}

// Tags adjust the metadata values included in the span. Tags can be
// used in the UI to filter, group and search for operations.
func (s *Span) Tags(tags map[string]interface{}) {
	for k, v := range tags {
		s.sp.SetTag(k, fmt.Sprintf("%v", v))
	}
}
