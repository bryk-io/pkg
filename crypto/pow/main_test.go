package pow

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"testing"
	"time"

	tdd "github.com/stretchr/testify/assert"
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
	return append(s.value, []byte(fmt.Sprintf("%d", s.nonce))...), nil
}

func TestSolve(t *testing.T) {
	assert := tdd.New(t)
	defer goleak.VerifyNone(t)
	rec := &src{value: []byte("this is the value")}
	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()

	r1 := Solve(ctx, rec, sha256.New(), 16)
	log.Printf("hash found: %s", <-r1)
	log.Printf("total attempts: %d", rec.Nonce())
	assert.True(Verify(rec, sha256.New(), 12), "verification error")
	assert.True(Verify(rec, sha256.New(), 16), "verification error")

	r2 := Solve(ctx, rec, sha256.New(), 28)
	val, ok := <-r2
	if ok || val != "" {
		log.Printf("hash found: %s", val)
	} else {
		log.Printf("round 2 terminated via timeout =/")
	}
}

// Run a protocol round to find a solution to a PoW challenge.
func ExampleSolve() {
	// Create a context with a maximum timeout of 10 seconds
	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()

	// Set the source for the round
	src := &src{}

	// Start the PoW round and wait for the result
	res := Solve(ctx, src, sha256.New(), 16)
	fmt.Printf("solution found: %x", <-res)
}

// Verify an already produced solution to a PoW challenge.
func ExampleVerify() {
	// The source element to verify
	solved := &src{}
	fmt.Printf("source verification result: %v", Verify(solved, sha256.New(), 12))
}
