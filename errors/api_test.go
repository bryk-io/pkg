package errors

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	tdd "github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	assert := tdd.New(t)
	err := New("foo")
	assert.Equal("foo", err.Error())
	err = New(fmt.Errorf("foo"))
	assert.Equal("foo", err.Error())
}

func TestWrap(t *testing.T) {
	assert := tdd.New(t)

	t.Run("WithoutPrefix", func(t *testing.T) {
		e := func() error {
			return Wrap(New("hi"), "")
		}()
		assert.NotNil(e)
		assert.Equal("hi", e.Error(), "with a string failed")
		assert.Nil(Wrap(nil, ""), "with nil failed")
		assert.Equal("yo", Wrap(fmt.Errorf("yo"), "").Error(), "with an error failed")
	})

	t.Run("WithPrefix", func(t *testing.T) {
		e := func() error {
			return Wrap(New("hi"), "level-1")
		}()
		assert.NotNil(e, "constructor failed")
		assert.Equal("level-1: hi", e.Error(), "with a string failed")
		assert.Equal("level-1: yo", Wrap(fmt.Errorf("yo"), "level-1").Error(), "with error failed")

		var (
			original *Error
			prefixed *Error
		)
		assert.True(As(Wrap(e, "level-2"), &prefixed))
		assert.True(As(e, &original), "invalid error type")
		assert.True(reflect.DeepEqual(prefixed.StackTrace(), original.StackTrace()), "wrong stack")
		assert.True(reflect.DeepEqual(prefixed.frames, original.frames), "wrong frames")
		assert.Equal("level-2: level-1: hi", prefixed.Error(), "wrong prefix")
		assert.False(original.Error() == prefixed.Error(), "wrap changed the original error")
		assert.Nil(Wrap(nil, "level-2"), "with nil failed")
	})
}

func TestUnwrap(t *testing.T) {
	assert := tdd.New(t)
	e1 := sampleA()
	assert.Equal(4, strings.Count(e1.Error(), ":"), "prefix markers")
	assert.Equal(3, strings.Count(Unwrap(e1).Error(), ":"), "unwrap one time")
	assert.Equal("deep error", Cause(e1).Error(), "get last level error")
}

func TestOpaque(t *testing.T) {
	assert := tdd.New(t)
	e1 := sampleA()
	assert.NotNil(Unwrap(e1), "original can be unwrapped")
	assert.NotNil(Cause(e1), "original can be inspected for cause")

	op := Opaque(e1)
	assert.Nil(Unwrap(op), "opaque can't be unwrapped")
	assert.True(Is(Cause(op), op), "opaque is root error")

	var se1 HasStack
	var se2 HasStack
	assert.True(As(e1, &se1))
	assert.True(As(op, &se2))
	assert.NotEqual(se1.StackTrace(), se2.StackTrace(), "opaque values should drop stack traces")
}

func TestFormat(t *testing.T) {
	assert := tdd.New(t)
	e1 := sampleA()

	type tests struct {
		src       string
		multiline bool
	}

	t.Run("Error", func(t *testing.T) {
		table := []tests{
			// simple string, without stacktrace
			{
				src:       fmt.Sprintf("%s", e1),
				multiline: false,
			},
			// with stacktrace
			{
				src:       fmt.Sprintf("%v", e1),
				multiline: true,
			},
			// with stacktrace, replacing local Go env paths (more portable)
			{
				src:       fmt.Sprintf("%+v", e1),
				multiline: true,
			},
		}
		for _, tt := range table {
			assert.Equal(tt.multiline, strings.Contains(tt.src, "\n"))
		}
	})

	t.Run("StackFrame", func(t *testing.T) {
		// Manually retrieve frames
		var err *Error
		assert.True(As(e1, &err))
		frames := err.StackTrace()
		for i, f := range frames {
			fmt.Printf("‹%d› %+v", i, f)
		}

		// Different frame formatting options
		regular := fmt.Sprintf("%s", frames[1])
		portable := fmt.Sprintf("%+v", frames[1])
		assert.Equal(regular, fmt.Sprintf("%v", frames[1]))
		fmt.Println(portable)
	})
}

func TestIs(t *testing.T) {
	assert := tdd.New(t)
	assert.True(Is(io.EOF, io.EOF), "io.EOF is not io.EOF")
	assert.True(Is(io.EOF, New(io.EOF)), "io.EOF is not New(io.EOF)")
	assert.True(Is(New(io.EOF), New(io.EOF)), "New(io.EOF) is not New(io.EOF)")
	assert.False(Is(nil, io.EOF), "nil is an error")
	assert.False(Is(io.EOF, fmt.Errorf("io.EOF")), "io.EOF is fmt.Errorf")

	t.Run("WithCustomError", func(t *testing.T) {
		customErr := errorWithCustomIs{
			Key: "TestForFun",
			Err: io.EOF,
		}

		shouldMatch := errorWithCustomIs{
			Key: "TestForFun",
		}

		shouldNotMatch := errorWithCustomIs{Key: "notOk"}

		assert.False(Is(customErr, shouldNotMatch), "customErr is a notOk customError")
		assert.False(Is(customErr, New(shouldNotMatch)), "customErr is a New(notOk customError)")
		assert.False(Is(New(customErr), shouldNotMatch), "New(customErr) is a notOk customError")
		assert.False(Is(New(customErr), New(shouldNotMatch)), "New(customErr) is a New(notOk customError)")
		assert.True(Is(customErr, customErr), "same error comparison failed")
		assert.True(Is(customErr, shouldMatch), "customErr is not a TestForFun customError")
		assert.True(Is(customErr, New(shouldMatch)), "customErr is not a New(TestForFun customError)")
		assert.True(Is(New(customErr), shouldMatch), "New(customErr) is not a TestForFun customError")
		assert.True(Is(New(customErr), New(shouldMatch)), "New(customErr) is not a New(TestForFun customError)")

		// convenience method to simplify several comparisons
		assert.False(IsAny(customErr, io.EOF, io.ErrNoProgress, io.ErrClosedPipe))
		assert.True(IsAny(customErr, io.EOF, io.ErrNoProgress, io.ErrClosedPipe, shouldMatch))
	})
}

func sampleA() error { return Wrap(sampleB(), "a") }
func sampleB() error { return Wrap(sampleC(), "b") }
func sampleC() error { return Wrap(sampleD(), "c") }
func sampleD() error { return Wrap(sampleE(), "d") }
func sampleE() error { return New("deep error") }

type errorWithCustomIs struct {
	Key string
	Err error
}

func (e errorWithCustomIs) Error() string {
	return "[" + e.Key + "]: " + e.Err.Error()
}

func (e errorWithCustomIs) Is(target error) bool {
	var te errorWithCustomIs
	return As(target, &te) && te.Key == e.Key
}
