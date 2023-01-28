package errors

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// Error is an error with an attached stacktrace. It can be used
// wherever the builtin error interface is expected.
type Error struct {
	ts     int64                  // UNIX timestamp (in milliseconds)
	err    error                  // root error value
	prev   error                  // previous error in the chain, present only on wrapped errors
	prefix string                 // prefix value when presenting error in simple textual form
	frames []StackFrame           // error stacktrace
	hints  []string               // additional contextual information
	events []Event                // events associated to the error
	tags   map[string]interface{} // additional metadata details
	mu     sync.Mutex
}

// Event instances can be used to provided additional contextual information
// for an error.
type Event struct {
	// Kind can be used to group specific events into categories or groups.
	Kind string `json:"kind,omitempty"`

	// Short and concise description of the event.
	Message string `json:"message,omitempty"`

	// UNIX timestamp (in milliseconds).
	Stamp int64 `json:"stamp,omitempty"`

	// Additional data associated with an event.
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// Error returns the underlying error's message.
func (e *Error) Error() string {
	msg := e.err.Error()
	if e.prefix != "" {
		msg = fmt.Sprintf("%s: %s", e.prefix, msg)
	}
	return msg
}

// Unwrap returns the next error in the error chain. If there is no next
// error, Unwrap returns nil.
func (e *Error) Unwrap() error {
	return e.prev
}

// Cause of the error. Obtained by traversing the entire error stack until
// an error with a `cause` value of 'nil'. Errors without cause are expected
// to be the root error of a failure condition.
func (e *Error) Cause() error {
	if e.prev == nil {
		// when no previous error is available we hit the root
		// of the chain
		return e.err
	}
	var ce hasCause
	if As(e.prev, &ce) {
		return ce.Cause()
	}
	return e
}

// StackTrace returns the frames in the callers stack.
func (e *Error) StackTrace() []StackFrame {
	return e.frames
}

// PortableTrace returns the frames in the callers stack attempting
// to remove any paths specific to the local system, making the
// information a bit more readable and portable.
func (e *Error) PortableTrace() []StackFrame {
	fr := make([]StackFrame, len(e.frames))
	copy(fr, e.frames)
	for i := range fr {
		fr[i].File = printFile(fr[i].File)
	}
	return fr
}

// AddHint registers additional information on the error instance.
func (e *Error) AddHint(hint string) {
	e.mu.Lock()
	if e.hints == nil {
		e.hints = []string{}
	}
	e.hints = append(e.hints, hint)
	e.mu.Unlock()
}

// AddEvent registers an additional event on the error instance.
func (e *Error) AddEvent(ev Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.events == nil {
		e.events = []Event{}
	}
	if ev.Stamp == 0 {
		ev.Stamp = time.Now().UnixMilli()
	}
	e.events = append(e.events, ev)
}

// SetTag registers a specific key/value pair on the error instance; replacing
// any previously set values under the same key.
func (e *Error) SetTag(key string, value interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.tags == nil {
		e.tags = make(map[string]interface{})
	}
	e.tags[key] = value
}

// Stamp returns error creation UNIX timestamp (in milliseconds).
func (e *Error) Stamp() int64 {
	return e.ts
}

// Tags provide additional context to an error in the form of arbitrary
// key/value pairs. If no tags are set on the error instance this method
// returns `nil`.
func (e *Error) Tags() map[string]interface{} {
	return e.tags
}

// Hints provide additional context to an error in the form of meaningful
// text messages. If no hints are set on the error instance this method
// returns `nil`.
func (e *Error) Hints() []string {
	return e.hints
}

// Events associated to the error, if any. Events usually provide valuable
// information on when/how an exception occurred.
func (e *Error) Events() []Event {
	return e.events
}

// Format error values using the escape codes defined by fmt.Formatter.
// The following verbs are supported:
//
//	%s   error message. Simply prints the basic error message as a
//	     string representation.
//	%v   basic format. Print the error including its stackframe formatted
//	     as in the standard library `runtime/debug.Stack()`.
//	%+v  extended format. Returns the stackframe formatted as in the
//	     standard library `runtime/debug.Stack()` but replacing the values
//	     for `GOPATH` and `GOROOT` on file paths. This makes the traces
//	     more portable and avoid exposing (noisy) local system details.
func (e *Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 's':
		_, _ = io.WriteString(s, e.Error())
	case 'v':
		str := fmt.Sprintf("%s\n", e.Error())
		if s.Flag('+') {
			for i, frame := range e.StackTrace() {
				str += fmt.Sprintf("‹%d› %+v", i, frame)
			}
			if len(e.hints) > 0 {
				str += "‹hints›\n"
				for _, h := range e.hints {
					str += fmt.Sprintf("\t- %s\n", h)
				}
			}
			if len(e.tags) > 0 {
				str += "‹tags›\n"
				for k, v := range e.tags {
					str += fmt.Sprintf("\t- %s=%v\n", k, v)
				}
			}
			if len(e.events) > 0 {
				str += "‹events›\n"
				for _, ev := range e.events {
					str += fmt.Sprintf("\t- (%s) %s\n", ev.Kind, ev.Message)
				}
			}
		} else {
			for _, frame := range e.StackTrace() {
				str += fmt.Sprintf("%v", frame)
			}
		}
		_, _ = io.WriteString(s, str)
	}
}
