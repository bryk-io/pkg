package pow

import (
	"context"
	"encoding/hex"
	"hash"
	"math"
	"math/big"

	"go.bryk.io/pkg/errors"
)

// Source elements can be subject of a Proof-of-Work validation round.
type Source interface {
	// Returns the current value of the "nonce" element.
	Nonce() int64

	// Set the value of the "nonce" element to a secure initial state.
	ResetNonce()

	// Adjust the value of the "nonce" element for a new processing round.
	IncrementNonce()

	// Returns a deterministic byte representation of the source instance.
	MarshalBinary() ([]byte, error)
}

// Solve the proof-of-work challenge for the source instance, i.e. finds
// an appropriate hash value for it based on the specified difficulty level.
// Since this is a potentially long-running operation it can be canceled at
// any time using the provided context.
func Solve(ctx context.Context, src Source, hm func() hash.Hash, difficulty uint) (<-chan string, error) {
	target, err := targetFor(hm, difficulty)
	if err != nil {
		return nil, err
	}
	res := make(chan string)
	src.ResetNonce()

	// start solve attempt in the background
	go func(target *big.Int) {
		defer close(res)
		digest := hm()
		var hashInt big.Int
		var err error
		var h, data []byte
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			// The nonce space is exhausted, stop the round
			if src.Nonce() == math.MaxInt64 {
				return
			}
			data, err = src.MarshalBinary()
			if err != nil {
				src.IncrementNonce()
				continue
			}
			digest.Reset()
			if _, err = digest.Write(data); err != nil {
				src.IncrementNonce()
				continue
			}
			h = digest.Sum(nil)
			hashInt.SetBytes(h)
			if hashInt.Cmp(target) == -1 {
				select {
				case res <- hex.EncodeToString(h):
				case <-ctx.Done():
				}
				return
			}
			src.IncrementNonce()
		}
	}(target)
	return res, nil
}

// Verify the source element satisfies the expected proof-of-work challenge
// based on the specified difficulty target.
func Verify(src Source, hm func() hash.Hash, difficulty uint) bool {
	target, err := targetFor(hm, difficulty)
	if err != nil {
		return false
	}
	data, err := src.MarshalBinary()
	if err != nil {
		return false
	}
	digest := hm()
	if _, err = digest.Write(data); err != nil {
		return false
	}
	var hashInt big.Int
	hashInt.SetBytes(digest.Sum(nil))
	return hashInt.Cmp(target) == -1
}

// Returns the target hash value for the specified difficulty level, i.e. the
// threshold the resulting digest must be smaller than. The difficulty cannot
// exceed the bit size of the digest provided by newDigest.
func targetFor(hm func() hash.Hash, difficulty uint) (*big.Int, error) {
	bits := uint(hm().Size() * 8)
	if difficulty > bits {
		return nil, errors.Errorf("difficulty %d exceeds digest size %d bits", difficulty, bits)
	}
	target := big.NewInt(1)
	target.Lsh(target, bits-difficulty)
	return target, nil
}
