package ulid

import (
	"testing"
	"time"

	tdd "github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	assert := tdd.New(t)

	var (
		id   ULID
		prev ULID
		err  error
	)
	prev, _ = New()
	for i := 0; i < 10; i++ {
		<-time.After(100 * time.Millisecond)
		id, err = New()
		assert.Nil(err)
		t.Logf("id: %s", id.String())
		t.Logf("bytes: %x", id.Bytes())
		t.Logf("stamp: %v", id.Timestamp())
		t.Logf("time: %v", id.Time())
		t.Logf("entropy: %v", id.Entropy())
		assert.True(id.Compare(id) == 0)
		assert.True(id.Compare(prev) >= 1)
		assert.True(prev.Compare(id) < 0)
		prev = id
	}
}
