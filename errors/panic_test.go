package errors

import (
	"testing"

	tdd "github.com/stretchr/testify/assert"
)

var samplePanic = `panic: sample error with '‹×›' [recovered]
	panic: sample error with '‹×›'

goroutine 19 [running]:
testing.tRunner.func1.1(0x1229020, 0xc00010e190)
	/usr/local/Cellar/go/1.15.3/libexec/src/testing/testing.go:1072 +0x46a
testing.tRunner.func1(0xc000102900)
	/usr/local/Cellar/go/1.15.3/libexec/src/testing/testing.go:1075 +0x636
panic(0x1229020, 0xc00010e190)
	/usr/local/Cellar/go/1.15.3/libexec/src/runtime/panic.go:975 +0x47a
go.bryk.io/pkg/errors.TestRedactable(0xc000102900)
	/Users/ben/Documents/go/src/go.bryk.io/pkg/errors/api_test.go:22 +0x1af
testing.tRunner(0xc000102900, 0x1250740)
	/usr/local/Cellar/go/1.15.3/libexec/src/testing/testing.go:1123 +0x203
created by testing.(*T).Run
	/usr/local/Cellar/go/1.15.3/libexec/src/testing/testing.go:1168 +0x5bc`

func TestParsePanic(t *testing.T) {
	assert := tdd.New(t)
	src, err := ParsePanic(samplePanic)
	assert.Nil(err, "parse failed")
	assert.Equal(6, len(src.StackTrace()), "incomplete stack trace")
}

func TestFromRecover(t *testing.T) {
	assert := tdd.New(t)
	defer func() {
		recovered := FromRecover(recover())
		assert.NotNil(recovered, "parse failed")
		assert.True(len(recovered.StackTrace()) > 5, "invalid stack trace")
		t.Logf("%+v", recovered)
	}()
	_ = a()
}

func a() error { return b() }

func b() error { return c() }

func c() error { panic("cool programs never panic!!!") }
