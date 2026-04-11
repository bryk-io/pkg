package errors

import (
	"fmt"
	"testing"

	tdd "github.com/stretchr/testify/assert"
)

func TestJoin(t *testing.T) {
	assert := tdd.New(t)

	t.Run("NilErrors", func(t *testing.T) {
		// All nil errors should return nil
		assert.Nil(Join(nil, nil, nil))
		assert.Nil(Join())
		assert.Nil(Join(nil))
	})

	t.Run("SingleError", func(t *testing.T) {
		// Single error should be wrapped
		err := Join(New("single error"))
		assert.NotNil(err)
		assert.Contains(err.Error(), "single error")
		assert.Contains(err.Error(), "joined error")
	})

	t.Run("MultipleErrors", func(t *testing.T) {
		// Multiple errors should be joined
		err1 := New("error one")
		err2 := New("error two")
		err3 := New("error three")

		joined := Join(err1, err2, err3)
		assert.NotNil(joined)
		assert.Contains(joined.Error(), "error one")
		assert.Contains(joined.Error(), "error two")
		assert.Contains(joined.Error(), "error three")
	})

	t.Run("MixedWithNil", func(t *testing.T) {
		// Mix of nil and non-nil errors
		err1 := New("error one")
		err2 := New("error two")

		joined := Join(nil, err1, nil, err2, nil)
		assert.NotNil(joined)
		assert.Contains(joined.Error(), "error one")
		assert.Contains(joined.Error(), "error two")
	})

	t.Run("UnwrapInterface", func(t *testing.T) {
		// joinedError should implement Unwrap() []error
		err1 := New("error one")
		err2 := New("error two")
		joined := Join(err1, err2)

		var je *joinedError
		assert.True(As(joined, &je))
		unwrapped := je.Unwrap()
		assert.Len(unwrapped, 2)
		assert.Equal("error one", unwrapped[0].Error())
		assert.Equal("error two", unwrapped[1].Error())
	})

	t.Run("StandardLibraryCompatibility", func(t *testing.T) {
		assert := tdd.New(t)
		// Should work with standard library errors.Is and errors.As
		targetErr := New("target error")
		err1 := New("error one")
		err2 := Wrap(targetErr, "wrapped")

		joined := Join(err1, err2)
		assert.NotNil(joined)

		// errors.Is should find the target in the joined error
		assert.True(Is(joined, targetErr))
	})

	t.Run("PreservesStructuredData", func(t *testing.T) {
		// Create errors with structured data
		err1 := New("error one")
		var e1 *Error
		assert.True(As(err1, &e1))
		e1.SetTag("key1", "value1")
		e1.AddHint("hint one")
		e1.AddEvent(Event{Kind: "test", Message: "event one"})

		err2 := New("error two")
		var e2 *Error
		assert.True(As(err2, &e2))
		e2.SetTag("key2", "value2")
		e2.AddHint("hint two")

		joined := Join(err1, err2)

		var je *joinedError
		assert.True(As(joined, &je))

		// Check tags are merged with index prefix
		tags := je.Tags()
		assert.NotNil(tags)
		assert.Equal("value1", tags["err.0.key1"])
		assert.Equal("value2", tags["err.1.key2"])

		// Check hints are merged with index prefix
		hints := je.Hints()
		assert.NotNil(hints)
		assert.Contains(hints[0], "err.0:")
		assert.Contains(hints[0], "hint one")

		// Check events are merged
		events := je.Events()
		assert.NotNil(events)
		assert.Contains(events[0].Kind, "err.0.")
	})

	t.Run("HasStackTrace", func(t *testing.T) {
		err1 := New("error one")
		err2 := New("error two")
		joined := Join(err1, err2)

		var je *joinedError
		assert.True(As(joined, &je))
		assert.NotEmpty(je.StackTrace())
	})

	t.Run("FormatVerbose", func(t *testing.T) {
		err1 := New("error one")
		err2 := New("error two")
		joined := Join(err1, err2)

		// Test %s format
		simple := fmt.Sprintf("%s", joined)
		assert.Contains(simple, "error one")
		assert.Contains(simple, "error two")

		// Test %v format
		verbose := fmt.Sprintf("%v", joined)
		assert.Contains(verbose, "error one")
		// Should include stack trace
		assert.Contains(verbose, "TestJoin")
	})

	t.Run("FormatExtended", func(t *testing.T) {
		err1 := New("error one")
		var e1 *Error
		assert.True(As(err1, &e1))
		e1.SetTag("test", "value")

		err2 := New("error two")
		joined := Join(err1, err2)

		// Test %+v format
		extended := fmt.Sprintf("%+v", joined)
		assert.Contains(extended, "error one")
		assert.Contains(extended, "error two")
		assert.Contains(extended, "‹tags›")
		assert.Contains(extended, "‹wrapped errors›")
	})
}

func TestCombine(t *testing.T) {
	assert := tdd.New(t)

	t.Run("NilOther", func(t *testing.T) {
		// When other is nil, should return original error unchanged
		original := New("original error")
		result := Combine(original, nil)
		assert.Equal(original, result)
	})

	t.Run("NilErr", func(t *testing.T) {
		// When err is nil, should return nil regardless of other
		other := New("other error")
		result := Combine(nil, other)
		assert.Nil(result)
	})

	t.Run("NonErrorType", func(t *testing.T) {
		// When err is not an *Error, should return it unchanged
		original := fmt.Errorf("standard error")
		other := New("other error")
		result := Combine(original, other)
		assert.Equal(original, result)
	})

	t.Run("SimpleErrorCombination", func(t *testing.T) {
		// Combine two errors - other error becomes a hint
		primary := New("primary error")
		other := New("other error")

		combined := Combine(primary, other)
		assert.NotNil(combined)

		var ce *Error
		assert.True(As(combined, &ce))
		assert.Contains(ce.Error(), "primary error")

		// Other error should be in hints
		hints := ce.Hints()
		assert.Len(hints, 1)
		assert.Contains(hints[0], "related error:")
		assert.Contains(hints[0], "other error")
	})

	t.Run("PreserveStructuredData", func(t *testing.T) {
		// Create primary error with structured data
		primary := New("primary error")
		var pe *Error
		assert.True(As(primary, &pe))
		pe.SetTag("primary_tag", "primary_value")
		pe.AddHint("primary hint")

		// Create other error with structured data
		other := New("other error")
		var oe *Error
		assert.True(As(other, &oe))
		oe.SetTag("other_tag", "other_value")
		oe.AddHint("other hint")
		oe.AddEvent(Event{Kind: "test", Message: "other event"})

		// Combine them
		combined := Combine(primary, other)

		var ce *Error
		assert.True(As(combined, &ce))

		// Primary data should be intact
		tags := ce.Tags()
		assert.Equal("primary_value", tags["primary_tag"])

		// Other's data should be merged with "related." prefix
		assert.Equal("other_value", tags["related.other_tag"])

		hints := ce.Hints()
		assert.Contains(hints, "primary hint")
		assert.Contains(hints, "related: other hint")

		events := ce.Events()
		assert.Len(events, 1)
		assert.Equal("related.test", events[0].Kind)
		assert.Equal("other event", events[0].Message)
	})

	t.Run("DoesNotAffectCauseAnalysis", func(t *testing.T) {
		// Combined error should not match the other error in Is()
		targetErr := New("target error")
		primary := Wrap(New("primary"), "wrapped")
		other := Wrap(targetErr, "other wrapped")

		combined := Combine(primary, other)

		// Combined should not match target error
		assert.False(Is(combined, targetErr))

		// But should still match primary
		assert.True(Is(combined, primary))
	})
}
