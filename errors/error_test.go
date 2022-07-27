package errors

import (
	"encoding/json"
	"fmt"
	"testing"

	tdd "github.com/stretchr/testify/assert"
)

func TestErrorUsage(t *testing.T) {
	assert := tdd.New(t)

	// Create a custom error object and an error instance with it
	a1 := &customErrorA{msg: "a-1"}
	e1 := New(a1)

	// Type comparisons for base error
	assert.False(Is(e1, &customErrorA{msg: "a-2"}), "not equal using custom evaluation")
	assert.True(Is(e1, &customErrorA{msg: "a-1"}), "equal to custom object")
	assert.True(Is(e1, New(&customErrorA{msg: "a-1"})), "equal to new instance")
	assert.Equal(Cause(e1), a1, "unwrap custom error object")

	// Create a second custom error object and combine it with the first.
	var ew *Error
	b1 := New(&customErrorB{msg: "b-1"})  // new error from custom error object
	e2 := Combine(b1, e1)                 // combine both
	assert.False(Is(e2, e1))              // first error don't influence cause analysis
	assert.True(As(e2, &ew))              // type casting should work
	assert.Equal(ew.hints[0], e1.Error()) // first error is available as hint
}

func TestReport(t *testing.T) {
	assert := tdd.New(t)

	// Create a custom error object and an error instance with it
	a1 := &customErrorA{msg: "a-1"}
	e1 := New(a1)

	// Create a second custom error object and combine it with the first.
	b1 := New(&customErrorB{msg: "b-1"})
	e2 := Combine(b1, e1)
	var e3 *Error
	As(e2, &e3)
	e3.SetTag("user", "rick")
	e3.SetTag("dimension", "c-137")
	e3.AddEvent(Event{
		Kind:    "debug",
		Message: "sample event",
		Attributes: map[string]interface{}{
			"payload": "event-value-goes-here",
		},
	})

	cc := new(jsonCodec)
	js, err := Report(e3, cc)
	assert.Nil(err, "failed to generate report")
	t.Logf("%s", js)
}

// Sample JSON codec.
type jsonCodec struct{}

func (c *jsonCodec) Marshal(err error) ([]byte, error) {
	data := map[string]interface{}{
		"error": err.Error(),
	}
	var oe *Error
	if As(err, &oe) {
		data["stamp"] = oe.Stamp()
		data["trace"] = oe.StackTrace()
		data["hints"] = oe.Hints()
		data["tags"] = oe.Tags()
		if ev := oe.Events(); ev != nil {
			data["events"] = ev
		}
	}
	return json.MarshalIndent(data, "", "  ")
}

type customErrorA struct{ msg string }
type customErrorB struct{ msg string }

func (c customErrorA) Is(target error) bool {
	var e *customErrorA
	if As(target, &e) {
		return e.msg == c.msg
	}
	return false
}

func (c customErrorA) Error() string {
	return fmt.Sprintf("error type a; with msg=%s", c.msg)
}

func (c *customErrorB) Error() string {
	return fmt.Sprintf("error type b; with msg=%s", c.msg)
}
