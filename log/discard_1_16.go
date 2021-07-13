// +build go1.16

package log

import (
	"io"
	stdL "log"
)

// Discard returns a no-op handler that will discard all generated output.
func Discard() Logger {
	return &stdLogger{
		log:     stdL.New(io.Discard, "", 0),
		discard: true,
	}
}
