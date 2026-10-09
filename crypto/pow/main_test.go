package pow

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"log"
	"math/rand"
	"testing"
	"time"

	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/errors"
	"go.uber.org/goleak"
)

type src struct {
	nonce int64
	value []byte
}

func (s *src) Nonce() int64 {
	return s.nonce
}

func (s *src) ResetNonce() {
	s.nonce = 0
}

func (s *src) IncrementNonce() {
	s.nonce++
}

func (s *src) MarshalBinary() ([]byte, error) {
	if rand.Intn(1000) == 777 {
		return nil, errors.New("simulated random encoding error") // jackpot
	}
	return append(s.value, fmt.Appendf(nil, "%d", s.nonce)...), nil
}

func TestSolve(t *testing.T) {
	assert := tdd.New(t)
	defer goleak.VerifyNone(t)
	rec := &src{value: []byte("this is the value")}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	r1, err := Solve(ctx, rec, sha256.New, 16)
	assert.Nil(err, "solve error")
	log.Printf("hash found: %s", <-r1)
	log.Printf("total attempts: %d", rec.Nonce())
	assert.True(Verify(rec, sha256.New, 12), "verification error")
	assert.True(Verify(rec, sha256.New, 16), "verification error")

	r2, err := Solve(ctx, rec, sha256.New, 28)
	assert.Nil(err, "solve error")
	val, ok := <-r2
	if ok || val != "" {
		log.Printf("hash found: %s", val)
	} else {
		log.Printf("round 2 terminated via timeout =/")
	}
}

func TestInvalidDifficulty(t *testing.T) {
	assert := tdd.New(t)
	rec := &src{value: []byte("this is the value")}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Difficulty cannot exceed the digest size (256 bits for SHA-256).
	_, err := Solve(ctx, rec, sha256.New, 257)
	assert.NotNil(err, "difficulty above digest size should fail")
	assert.False(Verify(rec, sha256.New, 257), "difficulty above digest size should fail")

	// Difficulty of zero requires no work but must still be accepted.
	r, err := Solve(ctx, rec, sha256.New, 0)
	assert.Nil(err, "zero difficulty should succeed")
	select {
	case val, ok := <-r:
		assert.True(ok, "expected a solution")
		assert.NotEmpty(val, "expected a solution")
	case <-ctx.Done():
		t.Fatal("solution not produced for zero difficulty")
	}
}

func TestNonSHA256Digest(t *testing.T) {
	assert := tdd.New(t)
	rec := &src{value: []byte("this is the value")}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// A 512-bit digest supports difficulties up to 512.
	r, err := Solve(ctx, rec, sha512.New, 16)
	assert.Nil(err, "solve error")
	select {
	case val, ok := <-r:
		assert.True(ok, "expected a solution")
		assert.True(Verify(rec, sha512.New, 16), "verification error")
		assert.NotEmpty(val, "expected a solution")
	case <-ctx.Done():
		t.Fatal("solution not produced for SHA-512 round")
	}
}

// Run a protocol round to find a solution to a PoW challenge.
func ExampleSolve() {
	// Create a context with a maximum timeout of 10 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set the source for the round
	src := &src{}

	// Start the PoW round and wait for the result
	res, err := Solve(ctx, src, sha256.New, 16)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("solution found: %s", <-res)
}

// Verify an already produced solution to a PoW challenge.
func ExampleVerify() {
	// The source element to verify
	solved := &src{}
	fmt.Printf("source verification result: %v", Verify(solved, sha256.New, 12))
}
