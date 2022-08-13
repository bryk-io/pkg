package errors

import (
	"fmt"
	"io"
	"regexp"
)

// Placeholder used for "redacted out" parameters.
const piiMarker = "‹×›"

// RegEx used to identify and swap escape "verbs" on format strings.
// https://pkg.go.dev/fmt
const escapeCode = `%[+\w]+`

var ecr *regexp.Regexp

func init() {
	ecr = regexp.MustCompile(escapeCode)
}

// SensitiveMessage returns a redactable message container. The `args`
// included can be redacted out of the message; specially useful when
// used as an error or log message. The returned message container can
// be specially formatted using the standard escape codes defined by
// `fmt.Formatter`. The following verbs are supported:
//
//	%s   return the redacted version of the message.
//	%v   return the redacted version of the message.
//	%+v  return the full version of the message.
func SensitiveMessage(format string, args ...interface{}) Redactable {
	return pii{
		format:   format,
		args:     append([]interface{}{}, args...),
		redacted: ecr.ReplaceAllString(format, piiMarker),
	}
}

// Redactable represents a message that contains some form of private
// or sensitive information that should be redacted when used as an error
// or log message.
type Redactable interface {
	// Redact the sensitive details out of the message.
	Redact() string

	// Disclose the sensitive details included in the message. Use with care.
	Disclose() string
}

type pii struct {
	format   string
	args     []interface{}
	redacted string
}

// Redact the sensitive details out of the message.
func (m pii) Redact() string {
	return m.redacted
}

// Disclose the sensitive details included in the message. Use with care.
func (m pii) Disclose() string {
	return fmt.Sprintf(m.format, m.args...)
}

// Format the message using the escape codes defined by fmt.Formatter.
// The following verbs are supported:
//
//	%s   return the redacted version of the message.
//	%v   return the redacted version of the message.
//	%+v  return the full version of the message.
func (m pii) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = fmt.Fprint(s, m.Disclose())
			return
		}
		fallthrough
	case 's':
		_, _ = io.WriteString(s, m.Redact())
	}
}
