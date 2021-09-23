//go:build !go1.16
// +build !go1.16

package log

import (
	"io/ioutil"
	stdL "log"
)

// Discard returns a no-op handler that will discard all generated output.
func Discard() Logger {
	return &stdLogger{
		log:     stdL.New(ioutil.Discard, "", 0),
		discard: true,
	}
}
