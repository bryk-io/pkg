package errors

import (
	"fmt"
	"strings"
	"testing"

	tdd "github.com/stretchr/testify/assert"
)

func TestSensitiveMessage(t *testing.T) {
	assert := tdd.New(t)
	var msg string
	secret := SensitiveMessage("my name is %s (or %03d)", "bond", 7)

	// Format with '%s' (redacted)
	msg = fmt.Sprintf("%s", secret)
	assert.Equal(0, strings.Count(msg, "bond"))
	assert.Equal(0, strings.Count(msg, "007"))
	assert.Equal(2, strings.Count(msg, piiMarker))

	// Format with '%v' (redacted)
	msg = fmt.Sprintf("%v", secret)
	assert.Equal(0, strings.Count(msg, "bond"))
	assert.Equal(0, strings.Count(msg, "007"))
	assert.Equal(2, strings.Count(msg, piiMarker))

	// Format with '%+v' (disclosed)
	msg = fmt.Sprintf("%+v", secret)
	assert.Equal(1, strings.Count(msg, "bond"))
	assert.Equal(1, strings.Count(msg, "007"))
	assert.Equal(0, strings.Count(msg, piiMarker))

	// Use a redactable message to create an error instance
	t.Run("AsError", func(t *testing.T) {
		e1 := New(secret)

		// Obtaining the textual representation of the error won't leak
		// any secret details.
		msg = e1.Error()
		assert.Equal(0, strings.Count(msg, "bond"))
		assert.Equal(0, strings.Count(msg, "007"))
		assert.Equal(2, strings.Count(msg, piiMarker))

		// Printing the error stacktrace won't leak any secret details
		msg = fmt.Sprintf("%v", e1)
		assert.Equal(0, strings.Count(msg, "bond"))
		assert.Equal(0, strings.Count(msg, "007"))
		assert.Equal(2, strings.Count(msg, piiMarker))

		// Printing the portable version of error stacktrace won't leak
		// any secret details
		msg = fmt.Sprintf("%+v", e1)
		assert.Equal(0, strings.Count(msg, "bond"))
		assert.Equal(0, strings.Count(msg, "007"))
		assert.Equal(2, strings.Count(msg, piiMarker))
	})
}
