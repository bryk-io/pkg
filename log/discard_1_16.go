//go:build go1.16
// +build go1.16

package log

import (
	"io"
	stdL "log"

	"go.bryk.io/pkg/metadata"
)

// Discard returns a no-op handler that will discard all generated output.
func Discard() Logger {
	return &stdLogger{
		log:     stdL.New(io.Discard, "", 0),
		tags:    metadata.New(),
		fields:  metadata.New(),
		discard: true,
	}
}
