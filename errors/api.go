package errors

import (
	stdErrors "errors"
	"fmt"
	"reflect"
	"time"
)

// New returns a new root error (i.e., without a cause) instance from
// the given value. If the provided `e` value is:
//   - An `Error` instance created with this package it will be returned as-is.
//   - An `error` value, will be set as the root cause for the new error instance.
//   - Any other value, will be passed to fmt.Errorf("%v") and the resulting error
//     value set as the root cause for the new error instance.
//
// The stacktrace will point to the line of code that called this function.
func New(e interface{}) error {
	if e == nil {
		return nil
	}

	var err error
	switch e := e.(type) {
	case *Error:
		return e
	case error:
		err = e
	default:
		err = fmt.Errorf("%v", e)
	}

	return &Error{
		ts:     time.Now().UnixMilli(),
		err:    err,
		prev:   nil,
		frames: getStack(1),
	}
}

// Opaque returns an error with the same formatting as `err` but that
// does not match `err` and cannot be unwrapped. This will essentially drop
// existing error context, useful when requiring a processing "barrier".
// This method returns a new root error (i.e., without a cause) instance.
func Opaque(err error) error {
	if err == nil {
		return nil
	}
	return &Error{
		ts:     time.Now().UnixMilli(),
		err:    stdErrors.New(err.Error()),
		prev:   nil,
		frames: getStack(1),
	}
}

// WithStack returns a new root error (i.e., without a cause) instance
// which stacktrace will point to the line of code that called this function.
func WithStack(err error) error {
	if err == nil {
		return nil
	}
	return &Error{
		ts:     time.Now().UnixMilli(),
		err:    err,
		prev:   nil,
		frames: getStack(1),
	}
}

// Errorf returns a new root error (i.e., without a cause) instance which
// stacktrace will point to the line of code that called this function.
//
// If the format specifier includes a `%w` verb with an error operand,
// the returned error will implement an Unwrap method returning the operand.
// It is invalid to include more than one `%w` verb or to supply it with an
// operand that does not implement the `error` interface. The `%w` verb is
// otherwise a synonym for `%v`.
func Errorf(format string, args ...interface{}) error {
	return &Error{
		ts:     time.Now().UnixMilli(),
		err:    fmt.Errorf(format, args...),
		prev:   nil,
		frames: getStack(1),
	}
}

// Wrapf returns a wrapped version of the provided error using a formatted
// string as prefix.
func Wrapf(err error, format string, args ...interface{}) error {
	return Wrap(err, fmt.Sprintf(format, args...))
}

// Wrap a given error into another one, this allows to create or expand an
// error cause chain. The provided `e` error will be registered as the root
// cause for the returned error instances. If `e` includes a stacktrace, it
// will be preserved.
func Wrap(e error, prefix string) error {
	if e == nil {
		return nil
	}

	// preserve original error stacktrace if available, otherwise
	// generate a new one pointing where this function was called
	frames := getStack(1)
	var se HasStack
	if As(e, &se) {
		frames = se.StackTrace()
	}

	return &Error{
		ts:     time.Now().UnixMilli(),
		err:    &Error{err: e},
		prev:   e,
		prefix: prefix,
		frames: frames,
	}
}

// Unwrap unpacks wrapped errors. If its argument's type has an `Unwrap`
// method, it calls the method once. Otherwise, it returns nil.
func Unwrap(err error) error {
	var we isWrapper
	if As(err, &we) {
		return we.Unwrap()
	}
	return nil
}

// Cause will recursively retrieve the topmost error which does not
// provide a cause, which is assumed to be the original failure condition.
func Cause(err error) error {
	var ce hasCause
	if As(err, &ce) {
		return ce.Cause()
	}
	return nil
}

// As unwraps `err` sequentially looking for an error that can be assigned
// to `target`, which must be a pointer. If it succeeds, it performs the
// assignment and returns true. Otherwise, it returns false. `target` must
// be a pointer to an interface or to a type implementing the error interface.
func As(err error, target interface{}) bool {
	if target == nil {
		return false
	}
	return stdErrors.As(err, target)
}

// Is detects whether the error is equal to a given error. Errors
// are considered equal by this function if:
//   - Are both the same object
//   - If `src` provides a custom `Is(e error) bool` implementation
//     it will be used and the result returned
//   - If `target` provides a custom `Is(e error) bool` implementation
//     it will be used and the result returned
//   - Comparison is true between `target` and `src` cause
//   - Comparison is true between `src` and `target` cause
func Is(src, target error) bool {
	// Are both the same object?
	if reflect.DeepEqual(src, target) {
		return true
	}

	// Compare with `src` cause
	var csE *Error
	if As(src, &csE) {
		return Is(csE.err, target)
	}

	// Compare with `target` cause
	var ctE *Error
	if As(target, &ctE) {
		return Is(src, ctE.err)
	}

	// Use custom 'Is' method on the source element, if available
	var cs comparableError
	if As(src, &cs) {
		if cs.Is(target) {
			return true
		}
	}

	// Use custom 'Is' method on the target element, if available
	var ct comparableError
	if As(target, &ct) {
		if ct.Is(src) {
			return true
		}
	}

	return false
}

// IsAny detects whether the error is equal to any of the provided target
// errors. This method uses the same equality rules as `Is`.
func IsAny(src error, target ...error) bool {
	for _, t := range target {
		if Is(src, t) {
			return true
		}
	}
	return false
}

// Combine the error given as first argument with an annotation that carries
// the error given as second argument. The second error does not participate
// in cause analysis (Is, IsAny, ...) and is only revealed when reporting out
// the error.
//
// Considerations:
//   - If `other` is nil, the first error is returned as-is.
//   - If `err` doesn't support adding additional details, it is returned as-is.
func Combine(err, other error) error {
	if err == nil || other == nil {
		return err
	}
	var oe *Error
	if As(err, &oe) {
		oe.AddHint(other.Error(), true)
		return oe
	}
	return err
}

// HasStack is implemented by error types that natively
// provide robust stack traces.
type HasStack interface {
	StackTrace() []StackFrame
}

type isWrapper interface {
	Unwrap() error
}

type hasCause interface {
	Cause() error
}

type comparableError interface {
	Is(target error) bool
}
