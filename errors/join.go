package errors

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// Join returns an error that wraps the given errors. Any nil error values are
// discarded. Join returns nil if every value in errs is nil. Otherwise, Join
// returns an error that wraps all the non-nil errors, with a stack trace pointing
// to the line of code that called this function.
//
// The error returned by Join implements the Unwrap() []error method, making it
// compatible with errors.Is and errors.As from the standard library.
//
// Unlike the standard errors.Join, this implementation preserves structured error
// details (stack traces, tags, hints, events) from any *Error instances in the
// errs slice.
func Join(errs ...error) error {
	// Filter out nil errors
	var nonNil []error
	for _, err := range errs {
		if err != nil {
			nonNil = append(nonNil, err)
		}
	}

	if len(nonNil) == 0 {
		return nil
	}

	if len(nonNil) == 1 {
		// Single error - wrap it to preserve the join point in the stack
		return Wrap(nonNil[0], "joined error")
	}

	// Create a multi-error that holds all errors
	je := &joinedError{
		errs:   nonNil,
		ts:     time.Now().UnixMilli(),
		frames: getStack(1),
	}

	// Merge structured data from any *Error instances
	for i, err := range nonNil {
		var oe *Error
		if As(err, &oe) {
			// Merge tags with index prefix
			if tags := oe.Tags(); len(tags) > 0 {
				for k, v := range tags {
					je.setTag(fmt.Sprintf("err.%d.%s", i, k), v)
				}
			}
			// Merge hints with index prefix
			if hints := oe.Hints(); len(hints) > 0 {
				for _, h := range hints {
					je.addHint(fmt.Sprintf("err.%d: %s", i, h))
				}
			}
			// Merge events with index prefix
			if events := oe.Events(); len(events) > 0 {
				for _, ev := range events {
					evCopy := ev
					evCopy.Kind = fmt.Sprintf("err.%d.%s", i, ev.Kind)
					je.addEvent(evCopy)
				}
			}
		}
	}

	return je
}

// joinedError represents multiple errors joined together.
type joinedError struct {
	errs   []error
	ts     int64
	frames []StackFrame
	hints  []string
	events []Event
	tags   map[string]interface{}
	mu     sync.Mutex
}

func (e *joinedError) Error() string {
	msgs := make([]string, len(e.errs))
	for i, err := range e.errs {
		msgs[i] = err.Error()
	}
	return fmt.Sprintf("joined %d errors: [ %s ]", len(e.errs), strings.Join(msgs, "; "))
}

// Unwrap returns the wrapped errors as a slice, compatible with standard library.
func (e *joinedError) Unwrap() []error {
	return e.errs
}

// StackTrace returns the frames in the callers stack.
func (e *joinedError) StackTrace() []StackFrame {
	return e.frames
}

// Stamp returns error creation UNIX timestamp (in milliseconds).
func (e *joinedError) Stamp() int64 {
	return e.ts
}

// Tags returns a copy of the tags map.
func (e *joinedError) Tags() map[string]interface{} {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.tags == nil {
		return nil
	}
	tagsCopy := make(map[string]interface{}, len(e.tags))
	for k, v := range e.tags {
		tagsCopy[k] = v
	}
	return tagsCopy
}

func (e *joinedError) setTag(key string, value interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.tags == nil {
		e.tags = make(map[string]interface{})
	}
	e.tags[key] = value
}

// Hints returns a copy of the hints slice.
func (e *joinedError) Hints() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.hints == nil {
		return nil
	}
	return append([]string{}, e.hints...)
}

func (e *joinedError) addHint(hint string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.hints == nil {
		e.hints = []string{}
	}
	e.hints = append(e.hints, hint)
}

// Events returns a copy of the events slice.
func (e *joinedError) Events() []Event {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.events == nil {
		return nil
	}
	return append([]Event{}, e.events...)
}

func (e *joinedError) addEvent(ev Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.events == nil {
		e.events = []Event{}
	}
	e.events = append(e.events, ev)
}

// Format implements fmt.Formatter for joined errors.
func (e *joinedError) Format(s fmt.State, verb rune) {
	switch verb {
	case 's':
		_, _ = io.WriteString(s, e.Error())
	case 'v':
		str := e.Error() + "\n"
		if s.Flag('+') {
			for i, frame := range e.StackTrace() {
				str += fmt.Sprintf("‹%d› %+v", i, frame)
			}
			if hints := e.Hints(); len(hints) > 0 {
				str += "‹hints›\n"
				for _, h := range hints {
					str += fmt.Sprintf("\t- %s\n", h)
				}
			}
			if tags := e.Tags(); len(tags) > 0 {
				str += "‹tags›\n"
				for k, v := range tags {
					str += fmt.Sprintf("\t- %s=%v\n", k, v)
				}
			}
			if events := e.Events(); len(events) > 0 {
				str += "‹events›\n"
				for _, ev := range events {
					str += fmt.Sprintf("\t- (%s) %s\n", ev.Kind, ev.Message)
				}
			}
			// List all wrapped errors
			str += "‹wrapped errors›\n"
			for i, err := range e.errs {
				str += fmt.Sprintf("\t[%d] %s\n", i, err.Error())
			}
		} else {
			for _, frame := range e.StackTrace() {
				str += fmt.Sprintf("%v", frame)
			}
		}
		_, _ = io.WriteString(s, str)
	}
}

// multiWrapper is the interface used by errors that wrap multiple errors.
// This is compatible with Go 1.20's errors.Join implementation.
type multiWrapper interface {
	Unwrap() []error
}
