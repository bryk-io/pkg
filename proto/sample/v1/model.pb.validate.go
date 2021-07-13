// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: sample/v1/model.proto

package samplev1

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/anypb"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = anypb.Any{}
)

// Validate checks the field values on Pong with the rules defined in the proto
// definition for this message. If any rules are violated, an error is returned.
func (m *Pong) Validate() error {
	if m == nil {
		return nil
	}

	// no validation rules for Ok

	return nil
}

// PongValidationError is the validation error returned by Pong.Validate if the
// designated constraints aren't met.
type PongValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e PongValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e PongValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e PongValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e PongValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e PongValidationError) ErrorName() string { return "PongValidationError" }

// Error satisfies the builtin error interface
func (e PongValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sPong.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = PongValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = PongValidationError{}

// Validate checks the field values on HealthResponse with the rules defined in
// the proto definition for this message. If any rules are violated, an error
// is returned.
func (m *HealthResponse) Validate() error {
	if m == nil {
		return nil
	}

	// no validation rules for Alive

	return nil
}

// HealthResponseValidationError is the validation error returned by
// HealthResponse.Validate if the designated constraints aren't met.
type HealthResponseValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e HealthResponseValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e HealthResponseValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e HealthResponseValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e HealthResponseValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e HealthResponseValidationError) ErrorName() string { return "HealthResponseValidationError" }

// Error satisfies the builtin error interface
func (e HealthResponseValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sHealthResponse.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = HealthResponseValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = HealthResponseValidationError{}

// Validate checks the field values on Response with the rules defined in the
// proto definition for this message. If any rules are violated, an error is returned.
func (m *Response) Validate() error {
	if m == nil {
		return nil
	}

	if l := utf8.RuneCountInString(m.GetName()); l < 2 || l > 5 {
		return ResponseValidationError{
			field:  "Name",
			reason: "value length must be between 2 and 5 runes, inclusive",
		}
	}

	return nil
}

// ResponseValidationError is the validation error returned by
// Response.Validate if the designated constraints aren't met.
type ResponseValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ResponseValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ResponseValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ResponseValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ResponseValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ResponseValidationError) ErrorName() string { return "ResponseValidationError" }

// Error satisfies the builtin error interface
func (e ResponseValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sResponse.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ResponseValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ResponseValidationError{}

// Validate checks the field values on DummyResponse with the rules defined in
// the proto definition for this message. If any rules are violated, an error
// is returned.
func (m *DummyResponse) Validate() error {
	if m == nil {
		return nil
	}

	// no validation rules for Ok

	return nil
}

// DummyResponseValidationError is the validation error returned by
// DummyResponse.Validate if the designated constraints aren't met.
type DummyResponseValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e DummyResponseValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e DummyResponseValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e DummyResponseValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e DummyResponseValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e DummyResponseValidationError) ErrorName() string { return "DummyResponseValidationError" }

// Error satisfies the builtin error interface
func (e DummyResponseValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sDummyResponse.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = DummyResponseValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = DummyResponseValidationError{}
