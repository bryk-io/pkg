package shamir

import (
	"fmt"
	"testing"

	tdd "github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestSplit(t *testing.T) {
	assert := tdd.New(t)
	t.Run("Invalid", func(t *testing.T) {
		var err error
		secret := []byte("test")

		// No parts or threshold
		_, err = Split(secret, 0, 0)
		assert.NotNil(err, "parts and threshold are required")

		// Threshold longer than the number of parts
		_, err = Split(secret, 2, 3)
		assert.NotNil(err, "threshold > parts")

		// Too many parts
		_, err = Split(secret, 1000, 3)
		assert.NotNil(err, "too many parts")

		// Too small threshold, should be at least 2
		_, err = Split(secret, 10, 1)
		assert.NotNil(err, "threshold > 1")

		// Empty secret
		_, err = Split(nil, 3, 2)
		assert.NotNil(err, "empty secret")
	})

	secret := []byte("test")
	out, err := Split(secret, 5, 3)
	assert.Nil(err, "split error")
	assert.Equal(5, len(out), "wrong parts count")

	for _, share := range out {
		assert.Equal(len(secret)+ShareOverhead, len(share), "share is too large")
	}
}

func TestCombine(t *testing.T) {
	assert := tdd.New(t)
	t.Run("Invalid", func(t *testing.T) {
		var err error

		// Not enough parts
		_, err = Combine(nil)
		assert.NotNil(err, "no parts to process")

		// Mis-match in length
		parts := [][]byte{
			[]byte("foo"),
			[]byte("ba"),
		}
		_, err = Combine(parts)
		assert.NotNil(err, "parts of different length")

		// Too short
		parts = [][]byte{
			[]byte("f"),
			[]byte("b"),
		}
		_, err = Combine(parts)
		assert.NotNil(err, "parts are too short")

		// Duplicate parts
		parts = [][]byte{
			[]byte("foo"),
			[]byte("foo"),
		}
		_, err = Combine(parts)
		assert.NotNil(err, "duplicate parts")
	})

	secret := []byte("test")
	out, err := Split(secret, 5, 3)
	assert.Nil(err, "split error")

	// There is 5*4*3 possible choices, brute force them all
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			if j == i {
				continue
			}
			for k := 0; k < 5; k++ {
				if k == i || k == j {
					continue
				}
				parts := [][]byte{out[i], out[j], out[k]}
				restored, err := Combine(parts)
				assert.Nil(err, "combine error")
				assert.Equal(secret, restored, "bad result")
			}
		}
	}
}

func ExampleSplit() {
	secret := []byte("super-secure-secret")
	parts, err := Split(secret, 5, 3)
	if err != nil {
		panic(err)
	}
	fmt.Printf("secret splitted on %d parts", len(parts))
}

func ExampleCombine() {
	parts := [][]byte{[]byte("part-1"), []byte("part-2"), []byte("part-3")}
	restored, err := Combine(parts)
	if err != nil {
		panic(err)
	}
	fmt.Printf("restored secret: %x", restored)
}
